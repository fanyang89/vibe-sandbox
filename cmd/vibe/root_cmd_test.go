package main

import "testing"

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd()
	if root.Use != "vibe" {
		t.Fatalf("root use = %q, want %q", root.Use, "vibe")
	}

	flag := root.PersistentFlags().Lookup("sandbox-root")
	if flag == nil {
		t.Fatal("missing --sandbox-root persistent flag")
	}

	expected := []string{"go", "done", "list", "pr", "create", "run", "destroy"}
	for _, name := range expected {
		if root.CommandPath() == "" {
			t.Fatal("unexpected empty command path")
		}
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("subcommand %q not found: %v", name, err)
		}
		if cmd.Name() != name {
			t.Fatalf("resolved command = %q, want %q", cmd.Name(), name)
		}
	}

	if cmd, _, err := root.Find([]string{"create"}); err != nil || !cmd.Hidden {
		t.Fatalf("create should be hidden, err=%v hidden=%v", err, cmd != nil && cmd.Hidden)
	}
	if cmd, _, err := root.Find([]string{"run"}); err != nil || !cmd.Hidden {
		t.Fatalf("run should be hidden, err=%v hidden=%v", err, cmd != nil && cmd.Hidden)
	}
	if cmd, _, err := root.Find([]string{"destroy"}); err != nil || !cmd.Hidden {
		t.Fatalf("destroy should be hidden, err=%v hidden=%v", err, cmd != nil && cmd.Hidden)
	}
}
