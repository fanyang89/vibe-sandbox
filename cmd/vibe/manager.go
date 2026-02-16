package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func newManager(root string) (*manager, error) {
	repoRoot, err := detectRepoRoot()
	if err != nil {
		return nil, err
	}

	sandboxRoot := resolveSandboxRoot(repoRoot, root)
	if !filepath.IsAbs(sandboxRoot) {
		sandboxRoot = filepath.Join(repoRoot, sandboxRoot)
	}
	metaDir := filepath.Join(sandboxRoot, "meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return nil, fmt.Errorf("create metadata dir: %w", err)
	}

	return &manager{repoRoot: repoRoot, sandboxRoot: sandboxRoot, metaDir: metaDir}, nil
}

func resolveSandboxRoot(repoRoot, root string) string {
	if root != "" {
		return root
	}

	defaultRoot := filepath.Join(repoRoot, defaultSandboxDir)
	legacyRoot := filepath.Join(repoRoot, legacySandboxDir)

	if _, err := os.Stat(defaultRoot); err == nil {
		return defaultRoot
	}
	if _, err := os.Stat(legacyRoot); err == nil {
		return legacyRoot
	}
	return defaultRoot
}

func detectRepoRoot() (string, error) {
	topLevel, err := gitOutputFn("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not in git repository: %w", err)
	}
	commonDir, err := gitOutputFn("", "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Clean(filepath.Join(topLevel, commonDir))
	}
	if filepath.Base(commonDir) == ".git" {
		return filepath.Dir(commonDir), nil
	}
	return filepath.Clean(topLevel), nil
}

func (m *manager) createSandbox(name, baseRef, branchPrefix string) (*sandboxMeta, error) {
	if !validName(name) {
		return nil, fmt.Errorf("invalid sandbox name %q", name)
	}
	if branchPrefix == "" {
		branchPrefix = defaultBranchPrefix
	}

	metaPath := m.metaPath(name)
	if _, err := os.Stat(metaPath); err == nil {
		return nil, fmt.Errorf("sandbox %q already exists", name)
	}

	worktree := filepath.Join(m.sandboxRoot, name)
	if _, err := os.Stat(worktree); err == nil {
		return nil, fmt.Errorf("worktree path already exists: %s", worktree)
	}

	branch := fmt.Sprintf("%s/%s", branchPrefix, name)
	if err := runCommandFn(m.repoRoot, os.Stdout, os.Stderr, "git", "worktree", "add", "-b", branch, worktree, baseRef); err != nil {
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	meta := &sandboxMeta{
		Name:      name,
		Branch:    branch,
		BaseRef:   baseRef,
		Worktree:  worktree,
		Container: containerName(name),
		CreatedAt: time.Now().Format(time.RFC3339),
	}
	if err := m.saveSandbox(meta); err != nil {
		_ = runCommandFn(m.repoRoot, io.Discard, io.Discard, "git", "worktree", "remove", worktree, "--force")
		return nil, err
	}
	return meta, nil
}

func (m *manager) destroySandbox(meta *sandboxMeta, force, deleteBranch bool) error {
	_ = commandOutputNoErrFn("", "docker", "rm", "-f", meta.Container)

	removeArgs := []string{"worktree", "remove", meta.Worktree}
	if force {
		removeArgs = append(removeArgs, "--force")
	}
	if err := runCommandFn(m.repoRoot, os.Stdout, os.Stderr, "git", removeArgs...); err != nil {
		return fmt.Errorf("remove worktree: %w", err)
	}

	if deleteBranch {
		branchDeleteFlag := "-d"
		if force {
			branchDeleteFlag = "-D"
		}
		if err := runCommandFn(m.repoRoot, os.Stdout, os.Stderr, "git", "branch", branchDeleteFlag, meta.Branch); err != nil {
			return fmt.Errorf("delete branch: %w", err)
		}
	}

	if err := os.Remove(m.metaPath(meta.Name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove metadata: %w", err)
	}
	return nil
}

func (m *manager) destroyAllSandboxes(force, deleteBranch bool) (int, error) {
	metas, err := m.listSandboxes()
	if err != nil {
		return 0, err
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Name < metas[j].Name })

	var (
		count    int
		failures []string
	)
	for i := range metas {
		meta := metas[i]
		if err := m.destroySandbox(&meta, force, deleteBranch); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", meta.Name, err))
			continue
		}
		count++
		fmt.Printf("cleaned sandbox %s\n", meta.Name)
	}
	if len(failures) > 0 {
		return count, fmt.Errorf("failed to clean some sandboxes:\n%s", strings.Join(failures, "\n"))
	}
	return count, nil
}

func (m *manager) loadSandbox(name string) (*sandboxMeta, error) {
	path := m.metaPath(name)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("sandbox %q not found", name)
		}
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	var meta sandboxMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return nil, fmt.Errorf("decode metadata: %w", err)
	}
	return &meta, nil
}

func (m *manager) saveSandbox(meta *sandboxMeta) error {
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.metaPath(meta.Name) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, m.metaPath(meta.Name)); err != nil {
		return err
	}
	return nil
}

func (m *manager) listSandboxes() ([]sandboxMeta, error) {
	entries, err := os.ReadDir(m.metaDir)
	if err != nil {
		return nil, fmt.Errorf("read metadata dir: %w", err)
	}

	metas := make([]sandboxMeta, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(m.metaDir, entry.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var meta sandboxMeta
		if err := json.Unmarshal(b, &meta); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		metas = append(metas, meta)
	}
	return metas, nil
}

func (m *manager) metaPath(name string) string {
	return filepath.Join(m.metaDir, name+".json")
}
