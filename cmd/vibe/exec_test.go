package main

import (
	"io"
	"strings"
	"testing"
)

func TestRunCommand(t *testing.T) {
	if err := runCommand("", io.Discard, io.Discard, "sh", "-c", "exit 0"); err != nil {
		t.Fatalf("runCommand success returned error: %v", err)
	}

	err := runCommand("", io.Discard, io.Discard, "sh", "-c", "exit 7")
	if err == nil {
		t.Fatal("expected runCommand error")
	}
	if !strings.Contains(err.Error(), "sh -c exit 7") {
		t.Fatalf("error missing command context: %v", err)
	}
}

func TestCommandOutput(t *testing.T) {
	out, err := commandOutput("", "sh", "-c", "printf ok")
	if err != nil {
		t.Fatalf("commandOutput returned error: %v", err)
	}
	if out != "ok" {
		t.Fatalf("output = %q, want %q", out, "ok")
	}

	_, err = commandOutput("", "sh", "-c", "echo boom >&2; exit 4")
	if err == nil {
		t.Fatal("expected commandOutput error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error missing stderr output: %v", err)
	}
}

func TestCommandOutputNoErr(t *testing.T) {
	out := commandOutputNoErr("", "sh", "-c", "echo x; exit 0")
	if strings.TrimSpace(out) != "x" {
		t.Fatalf("output = %q, want %q", out, "x")
	}

	out = commandOutputNoErr("", "sh", "-c", "exit 1")
	if out != "" {
		t.Fatalf("output = %q, want empty on error", out)
	}
}

func TestGitOutput(t *testing.T) {
	out, err := gitOutput("", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("gitOutput returned error: %v", err)
	}
	if out != "true" {
		t.Fatalf("gitOutput = %q, want %q", out, "true")
	}
}
