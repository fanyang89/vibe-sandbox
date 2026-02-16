package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "Feature A", want: "feature-a"},
		{in: "a/b/c", want: "a-b-c"},
		{in: "_X-1.", want: "_x-1"},
		{in: "%%%", want: ""},
	}

	for _, tc := range cases {
		if got := normalizeName(tc.in); got != tc.want {
			t.Fatalf("normalizeName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidName(t *testing.T) {
	if !validName("ok-name_1.0") {
		t.Fatal("expected valid name")
	}
	if validName("bad/name") {
		t.Fatal("slash should be invalid")
	}
}

func TestGenerateName(t *testing.T) {
	name := generateName()
	if len(name) < len("sb-20060102-150405") {
		t.Fatalf("generated name too short: %q", name)
	}
}

func TestShortHashDeterministic(t *testing.T) {
	a := shortHash("same-input")
	b := shortHash("same-input")
	if a != b {
		t.Fatalf("shortHash should be deterministic, got %q vs %q", a, b)
	}
	if len(a) != 12 {
		t.Fatalf("shortHash length mismatch, got %d", len(a))
	}
}

func TestResolveSandboxRootUsesDefaultWhenNoExistingDirs(t *testing.T) {
	repoRoot := t.TempDir()

	got := resolveSandboxRoot(repoRoot, "")
	want := filepath.Join(repoRoot, defaultSandboxDir)
	if got != want {
		t.Fatalf("resolveSandboxRoot() = %q, want %q", got, want)
	}
}

func TestResolveSandboxRootFallsBackToLegacy(t *testing.T) {
	repoRoot := t.TempDir()
	legacyRoot := filepath.Join(repoRoot, legacySandboxDir)
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}

	got := resolveSandboxRoot(repoRoot, "")
	if got != legacyRoot {
		t.Fatalf("resolveSandboxRoot() = %q, want %q", got, legacyRoot)
	}
}

func TestResolveSandboxRootPrefersDefaultOverLegacy(t *testing.T) {
	repoRoot := t.TempDir()
	defaultRoot := filepath.Join(repoRoot, defaultSandboxDir)
	legacyRoot := filepath.Join(repoRoot, legacySandboxDir)
	if err := os.MkdirAll(defaultRoot, 0o755); err != nil {
		t.Fatalf("create default dir: %v", err)
	}
	if err := os.MkdirAll(legacyRoot, 0o755); err != nil {
		t.Fatalf("create legacy dir: %v", err)
	}

	got := resolveSandboxRoot(repoRoot, "")
	if got != defaultRoot {
		t.Fatalf("resolveSandboxRoot() = %q, want %q", got, defaultRoot)
	}
}

func TestResolveSandboxRootUsesExplicitValue(t *testing.T) {
	repoRoot := t.TempDir()
	explicit := "custom-sandboxes"

	got := resolveSandboxRoot(repoRoot, explicit)
	if got != explicit {
		t.Fatalf("resolveSandboxRoot() = %q, want %q", got, explicit)
	}
}
