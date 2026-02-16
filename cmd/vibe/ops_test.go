package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestResolveBaseRef(t *testing.T) {
	origGit := gitOutputFn
	t.Cleanup(func() { gitOutputFn = origGit })

	t.Run("explicit base", func(t *testing.T) {
		gitOutputFn = func(dir string, args ...string) (string, error) {
			t.Fatal("gitOutputFn should not be called when base is explicit")
			return "", nil
		}
		got, err := resolveBaseRef("/repo", "release")
		if err != nil {
			t.Fatalf("resolveBaseRef returned error: %v", err)
		}
		if got != "release" {
			t.Fatalf("baseRef = %q, want %q", got, "release")
		}
	})

	t.Run("current branch", func(t *testing.T) {
		gitOutputFn = func(dir string, args ...string) (string, error) {
			return "feature/a", nil
		}
		got, err := resolveBaseRef("/repo", "")
		if err != nil {
			t.Fatalf("resolveBaseRef returned error: %v", err)
		}
		if got != "feature/a" {
			t.Fatalf("baseRef = %q, want %q", got, "feature/a")
		}
	})

	t.Run("detached head", func(t *testing.T) {
		gitOutputFn = func(dir string, args ...string) (string, error) {
			return "HEAD", nil
		}
		_, err := resolveBaseRef("/repo", "")
		if err == nil || !strings.Contains(err.Error(), "detached HEAD") {
			t.Fatalf("expected detached HEAD error, got %v", err)
		}
	})

	t.Run("git error", func(t *testing.T) {
		gitOutputFn = func(dir string, args ...string) (string, error) {
			return "", errors.New("git failed")
		}
		_, err := resolveBaseRef("/repo", "")
		if err == nil || !strings.Contains(err.Error(), "detect current branch") {
			t.Fatalf("expected wrapped git error, got %v", err)
		}
	})
}

func TestCreatePRSuccess(t *testing.T) {
	origRun := runCommandFn
	origOut := commandOutputFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputFn = origOut
	})

	meta := &sandboxMeta{Worktree: "/repo/sb", Branch: "codex/feat-a", BaseRef: "main"}

	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		if dir != meta.Worktree {
			t.Fatalf("push dir = %q, want %q", dir, meta.Worktree)
		}
		if name != "git" {
			t.Fatalf("push command = %q, want git", name)
		}
		want := []string{"push", "-u", "origin", "codex/feat-a"}
		if !equalStrings(args, want) {
			t.Fatalf("push args = %+v, want %+v", args, want)
		}
		return nil
	}

	commandOutputFn = func(dir, name string, args ...string) (string, error) {
		if dir != meta.Worktree {
			t.Fatalf("gh dir = %q, want %q", dir, meta.Worktree)
		}
		if name != "gh" {
			t.Fatalf("gh command = %q, want gh", name)
		}
		if !containsArg(args, "--head") || !containsArg(args, meta.Branch) {
			t.Fatalf("gh args missing head branch: %+v", args)
		}
		if !containsPairArg(args, "--base", "develop") {
			t.Fatalf("gh args missing base override: %+v", args)
		}
		if !containsArg(args, "--fill") {
			t.Fatalf("gh args missing --fill: %+v", args)
		}
		if !containsArg(args, "--draft") {
			t.Fatalf("gh args missing --draft: %+v", args)
		}
		return "https://example.test/pull/1\n", nil
	}

	if err := createPR(meta, "develop", "", "", true); err != nil {
		t.Fatalf("createPR returned error: %v", err)
	}
}

func TestCreatePRPushError(t *testing.T) {
	origRun := runCommandFn
	origOut := commandOutputFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputFn = origOut
	})

	meta := &sandboxMeta{Worktree: "/repo/sb", Branch: "codex/feat-a", BaseRef: "main"}
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		return errors.New("push failed")
	}
	commandOutputFn = func(dir, name string, args ...string) (string, error) {
		t.Fatal("commandOutputFn should not be called on push failure")
		return "", nil
	}

	err := createPR(meta, "", "", "", false)
	if err == nil || !strings.Contains(err.Error(), "push branch") {
		t.Fatalf("expected push branch error, got %v", err)
	}
}

func TestCreatePRGhError(t *testing.T) {
	origRun := runCommandFn
	origOut := commandOutputFn
	t.Cleanup(func() {
		runCommandFn = origRun
		commandOutputFn = origOut
	})

	meta := &sandboxMeta{Worktree: "/repo/sb", Branch: "codex/feat-a", BaseRef: "main"}
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		return nil
	}
	commandOutputFn = func(dir, name string, args ...string) (string, error) {
		return "", errors.New("gh failed")
	}

	err := createPR(meta, "", "title", "body", false)
	if err == nil || !strings.Contains(err.Error(), "create pr") {
		t.Fatalf("expected create pr error, got %v", err)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func containsPairArg(args []string, key, val string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == val {
			return true
		}
	}
	return false
}
