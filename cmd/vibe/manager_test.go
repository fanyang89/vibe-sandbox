package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadAndListSandbox(t *testing.T) {
	m := newTestManager(t)
	meta := &sandboxMeta{
		Name:      "feat-a",
		Branch:    "codex/feat-a",
		BaseRef:   "main",
		Worktree:  filepath.Join(m.sandboxRoot, "feat-a"),
		Container: "codex-sb-feat-a",
		CreatedAt: "2026-01-01T00:00:00Z",
	}

	if err := m.saveSandbox(meta); err != nil {
		t.Fatalf("saveSandbox returned error: %v", err)
	}

	got, err := m.loadSandbox("feat-a")
	if err != nil {
		t.Fatalf("loadSandbox returned error: %v", err)
	}
	if got.Name != meta.Name || got.Branch != meta.Branch || got.Worktree != meta.Worktree {
		t.Fatalf("loaded metadata mismatch: got %+v want %+v", got, meta)
	}

	metas, err := m.listSandboxes()
	if err != nil {
		t.Fatalf("listSandboxes returned error: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "feat-a" {
		t.Fatalf("listSandboxes = %+v, want one sandbox", metas)
	}
}

func TestLoadSandboxNotFound(t *testing.T) {
	m := newTestManager(t)
	_, err := m.loadSandbox("missing")
	if err == nil || !strings.Contains(err.Error(), `sandbox "missing" not found`) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestLoadSandboxBadJSON(t *testing.T) {
	m := newTestManager(t)
	if err := os.WriteFile(m.metaPath("bad"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}

	_, err := m.loadSandbox("bad")
	if err == nil || !strings.Contains(err.Error(), "decode metadata") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestListSandboxesIgnoresNonJSONAndDirs(t *testing.T) {
	m := newTestManager(t)
	meta := &sandboxMeta{Name: "ok", Branch: "codex/ok", BaseRef: "main"}
	if err := m.saveSandbox(meta); err != nil {
		t.Fatalf("saveSandbox: %v", err)
	}
	if err := os.WriteFile(filepath.Join(m.metaDir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write notes: %v", err)
	}
	if err := os.Mkdir(filepath.Join(m.metaDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	metas, err := m.listSandboxes()
	if err != nil {
		t.Fatalf("listSandboxes returned error: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "ok" {
		t.Fatalf("listSandboxes = %+v, want one valid metadata file", metas)
	}
}

func TestListSandboxesDecodeError(t *testing.T) {
	m := newTestManager(t)
	path := filepath.Join(m.metaDir, "broken.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write broken json: %v", err)
	}

	_, err := m.listSandboxes()
	if err == nil || !strings.Contains(err.Error(), "decode ") || !strings.Contains(err.Error(), path) {
		t.Fatalf("expected decode error with file path, got %v", err)
	}
}

func TestMetaPath(t *testing.T) {
	m := newTestManager(t)
	got := m.metaPath("x")
	want := filepath.Join(m.metaDir, "x.json")
	if got != want {
		t.Fatalf("metaPath = %q, want %q", got, want)
	}
}

func TestCreateSandboxRejectsInvalidName(t *testing.T) {
	m := newTestManager(t)
	_, err := m.createSandbox("bad/name", "main", "")
	if err == nil || !strings.Contains(err.Error(), "invalid sandbox name") {
		t.Fatalf("expected invalid name error, got %v", err)
	}
}

func TestCreateSandboxRejectsExistingMetadata(t *testing.T) {
	m := newTestManager(t)
	if err := os.WriteFile(m.metaPath("dup"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	_, err := m.createSandbox("dup", "main", "")
	if err == nil || !strings.Contains(err.Error(), `sandbox "dup" already exists`) {
		t.Fatalf("expected existing metadata error, got %v", err)
	}
}

func TestCreateSandboxRejectsExistingWorktreePath(t *testing.T) {
	m := newTestManager(t)
	if err := os.MkdirAll(filepath.Join(m.sandboxRoot, "dup"), 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	_, err := m.createSandbox("dup", "main", "")
	if err == nil || !strings.Contains(err.Error(), "worktree path already exists") {
		t.Fatalf("expected existing worktree error, got %v", err)
	}
}

func TestCreateSandboxSuccess(t *testing.T) {
	origRun := runCommandFn
	t.Cleanup(func() { runCommandFn = origRun })

	m := newTestManager(t)
	var gotDir string
	var gotName string
	var gotArgs []string
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		gotDir = dir
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	meta, err := m.createSandbox("feat-1", "main", "")
	if err != nil {
		t.Fatalf("createSandbox returned error: %v", err)
	}
	if meta.Branch != defaultBranchPrefix+"/feat-1" {
		t.Fatalf("branch = %q, want default prefix branch", meta.Branch)
	}
	if meta.Container != containerName("feat-1") {
		t.Fatalf("container = %q", meta.Container)
	}
	if gotDir != m.repoRoot || gotName != "git" {
		t.Fatalf("unexpected git invocation: dir=%q name=%q", gotDir, gotName)
	}
	wantArgs := []string{"worktree", "add", "-b", defaultBranchPrefix + "/feat-1", filepath.Join(m.sandboxRoot, "feat-1"), "main"}
	if !equalStrings(gotArgs, wantArgs) {
		t.Fatalf("git args = %+v, want %+v", gotArgs, wantArgs)
	}
	if _, err := m.loadSandbox("feat-1"); err != nil {
		t.Fatalf("saved metadata missing: %v", err)
	}
}

func TestCreateSandboxSaveFailureRunsCleanup(t *testing.T) {
	origRun := runCommandFn
	t.Cleanup(func() { runCommandFn = origRun })

	m := newTestManager(t)
	m.metaDir = filepath.Join(m.sandboxRoot, "missing-meta")

	var calls [][]string
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		if name != "git" {
			return fmt.Errorf("unexpected command %q", name)
		}
		calls = append(calls, append([]string(nil), args...))
		return nil
	}

	_, err := m.createSandbox("feat-2", "main", "")
	if err == nil {
		t.Fatal("expected saveSandbox failure")
	}
	if len(calls) != 2 {
		t.Fatalf("expected add + cleanup calls, got %d (%+v)", len(calls), calls)
	}
	if !equalStrings(calls[1], []string{"worktree", "remove", filepath.Join(m.sandboxRoot, "feat-2"), "--force"}) {
		t.Fatalf("cleanup args = %+v", calls[1])
	}
}

func TestDestroySandboxSuccess(t *testing.T) {
	origRun := runCommandFn
	origNoErr := commandOutputNoErrFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputNoErrFn = origNoErr
	})

	m := newTestManager(t)
	meta := &sandboxMeta{
		Name:      "feat-1",
		Branch:    "codex/feat-1",
		BaseRef:   "main",
		Worktree:  filepath.Join(m.sandboxRoot, "feat-1"),
		Container: "codex-sb-feat-1",
	}
	if err := m.saveSandbox(meta); err != nil {
		t.Fatalf("saveSandbox: %v", err)
	}

	var dockerCalls [][]string
	commandOutputNoErrFn = func(dir, name string, args ...string) string {
		dockerCalls = append(dockerCalls, append([]string{name}, args...))
		return ""
	}

	var gitCalls [][]string
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		if name != "git" {
			return fmt.Errorf("unexpected command %q", name)
		}
		gitCalls = append(gitCalls, append([]string(nil), args...))
		return nil
	}

	if err := m.destroySandbox(meta, true, true); err != nil {
		t.Fatalf("destroySandbox returned error: %v", err)
	}
	if len(dockerCalls) != 1 || !equalStrings(dockerCalls[0], []string{"docker", "rm", "-f", "codex-sb-feat-1"}) {
		t.Fatalf("unexpected docker cleanup calls: %+v", dockerCalls)
	}
	if len(gitCalls) != 2 {
		t.Fatalf("expected 2 git calls, got %d (%+v)", len(gitCalls), gitCalls)
	}
	if !equalStrings(gitCalls[0], []string{"worktree", "remove", meta.Worktree, "--force"}) {
		t.Fatalf("unexpected worktree remove args: %+v", gitCalls[0])
	}
	if !equalStrings(gitCalls[1], []string{"branch", "-D", meta.Branch}) {
		t.Fatalf("unexpected branch delete args: %+v", gitCalls[1])
	}
	if _, err := os.Stat(m.metaPath(meta.Name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("metadata should be removed, stat err=%v", err)
	}
}

func TestDestroySandboxWorktreeError(t *testing.T) {
	origRun := runCommandFn
	origNoErr := commandOutputNoErrFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputNoErrFn = origNoErr
	})

	m := newTestManager(t)
	meta := &sandboxMeta{Name: "x", Branch: "codex/x", Worktree: filepath.Join(m.sandboxRoot, "x"), Container: "codex-sb-x"}
	commandOutputNoErrFn = func(dir, name string, args ...string) string { return "" }
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		return errors.New("boom")
	}

	err := m.destroySandbox(meta, false, false)
	if err == nil || !strings.Contains(err.Error(), "remove worktree") {
		t.Fatalf("expected remove worktree error, got %v", err)
	}
}

func TestDestroySandboxDeleteBranchError(t *testing.T) {
	origRun := runCommandFn
	origNoErr := commandOutputNoErrFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputNoErrFn = origNoErr
	})

	m := newTestManager(t)
	meta := &sandboxMeta{Name: "x", Branch: "codex/x", Worktree: filepath.Join(m.sandboxRoot, "x"), Container: "codex-sb-x"}
	commandOutputNoErrFn = func(dir, name string, args ...string) string { return "" }
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		if len(args) > 0 && args[0] == "branch" {
			return errors.New("cannot delete")
		}
		return nil
	}

	err := m.destroySandbox(meta, false, true)
	if err == nil || !strings.Contains(err.Error(), "delete branch") {
		t.Fatalf("expected delete branch error, got %v", err)
	}
}

func TestDestroyAllSandboxesAggregatesFailures(t *testing.T) {
	origRun := runCommandFn
	origNoErr := commandOutputNoErrFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputNoErrFn = origNoErr
	})

	m := newTestManager(t)
	metaA := &sandboxMeta{Name: "a", Branch: "codex/a", BaseRef: "main", Worktree: filepath.Join(m.sandboxRoot, "a"), Container: "codex-sb-a"}
	metaB := &sandboxMeta{Name: "b", Branch: "codex/b", BaseRef: "main", Worktree: filepath.Join(m.sandboxRoot, "b"), Container: "codex-sb-b"}
	if err := m.saveSandbox(metaA); err != nil {
		t.Fatalf("saveSandbox a: %v", err)
	}
	if err := m.saveSandbox(metaB); err != nil {
		t.Fatalf("saveSandbox b: %v", err)
	}

	commandOutputNoErrFn = func(dir, name string, args ...string) string { return "" }
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		if len(args) >= 3 && args[0] == "worktree" && args[1] == "remove" && args[2] == metaB.Worktree {
			return errors.New("cannot remove")
		}
		return nil
	}

	count, err := m.destroyAllSandboxes(false, false)
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if err == nil || !strings.Contains(err.Error(), "failed to clean some sandboxes") || !strings.Contains(err.Error(), "b: ") {
		t.Fatalf("expected aggregated error, got %v", err)
	}
}

func newTestManager(t *testing.T) *manager {
	t.Helper()
	repoRoot := t.TempDir()
	sandboxRoot := filepath.Join(repoRoot, "sandboxes")
	metaDir := filepath.Join(sandboxRoot, "meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("mkdir meta dir: %v", err)
	}
	return &manager{repoRoot: repoRoot, sandboxRoot: sandboxRoot, metaDir: metaDir}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
