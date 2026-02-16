package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func resolveBaseRef(repoRoot, base string) (string, error) {
	if base != "" {
		return base, nil
	}
	current, err := gitOutputFn(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("detect current branch: %w", err)
	}
	if current == "HEAD" {
		return "", errors.New("detached HEAD; pass --base explicitly")
	}
	return current, nil
}

func createPR(meta *sandboxMeta, base, title, body string, draft bool) error {
	prBase := meta.BaseRef
	if base != "" {
		prBase = base
	}

	if err := runCommandFn(meta.Worktree, os.Stdout, os.Stderr, "git", "push", "-u", "origin", meta.Branch); err != nil {
		return fmt.Errorf("push branch: %w", err)
	}

	ghArgs := []string{"pr", "create", "--head", meta.Branch, "--base", prBase}
	if title != "" {
		ghArgs = append(ghArgs, "--title", title)
	}
	if body != "" {
		ghArgs = append(ghArgs, "--body", body)
	}
	if title == "" && body == "" {
		ghArgs = append(ghArgs, "--fill")
	}
	if draft {
		ghArgs = append(ghArgs, "--draft")
	}

	out, err := commandOutputFn(meta.Worktree, "gh", ghArgs...)
	if err != nil {
		return fmt.Errorf("create pr: %w", err)
	}
	fmt.Println(strings.TrimSpace(out))
	return nil
}
