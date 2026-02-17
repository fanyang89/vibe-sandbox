package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestParseDevcontainerBuild(t *testing.T) {
	t.Run("no build", func(t *testing.T) {
		got, hasBuild, err := parseDevcontainerBuild(&devcontainerConfig{})
		if err != nil {
			t.Fatalf("parseDevcontainerBuild returned error: %v", err)
		}
		if hasBuild {
			t.Fatal("expected hasBuild=false")
		}
		if got.Dockerfile != "" || got.Context != "" || len(got.Args) != 0 {
			t.Fatalf("unexpected build: %+v", got)
		}
	})

	t.Run("dockerfile and context fields", func(t *testing.T) {
		cfg := &devcontainerConfig{DockerFile: "Dockerfile.dev", Context: ".."}
		got, hasBuild, err := parseDevcontainerBuild(cfg)
		if err != nil {
			t.Fatalf("parseDevcontainerBuild returned error: %v", err)
		}
		if !hasBuild {
			t.Fatal("expected hasBuild=true")
		}
		if got.Dockerfile != "Dockerfile.dev" || got.Context != ".." {
			t.Fatalf("unexpected build: %+v", got)
		}
	})

	t.Run("build string", func(t *testing.T) {
		cfg := &devcontainerConfig{Build: []byte(`"Dockerfile.alt"`)}
		got, hasBuild, err := parseDevcontainerBuild(cfg)
		if err != nil {
			t.Fatalf("parseDevcontainerBuild returned error: %v", err)
		}
		if !hasBuild {
			t.Fatal("expected hasBuild=true")
		}
		if got.Dockerfile != "Dockerfile.alt" {
			t.Fatalf("unexpected dockerfile: %q", got.Dockerfile)
		}
	})

	t.Run("build object", func(t *testing.T) {
		cfg := &devcontainerConfig{Build: []byte(`{"dockerfile":"Dockerfile.obj","context":"../ctx","args":{"B":"2","A":"1"}}`)}
		got, hasBuild, err := parseDevcontainerBuild(cfg)
		if err != nil {
			t.Fatalf("parseDevcontainerBuild returned error: %v", err)
		}
		if !hasBuild {
			t.Fatal("expected hasBuild=true")
		}
		if got.Dockerfile != "Dockerfile.obj" || got.Context != "../ctx" {
			t.Fatalf("unexpected build fields: %+v", got)
		}
		if !reflect.DeepEqual(got.Args, map[string]string{"A": "1", "B": "2"}) {
			t.Fatalf("unexpected build args: %+v", got.Args)
		}
	})

	t.Run("invalid build object", func(t *testing.T) {
		cfg := &devcontainerConfig{Build: []byte(`{"dockerfile":`)}
		_, _, err := parseDevcontainerBuild(cfg)
		if err == nil || !strings.Contains(err.Error(), "decode devcontainer build object") {
			t.Fatalf("expected decode error, got %v", err)
		}
	})
}

func TestResolveRuntimeSpecMissingDevcontainer(t *testing.T) {
	worktree := t.TempDir()
	spec, err := resolveRuntimeSpec(worktree, "", "", false)
	if err != nil {
		t.Fatalf("resolveRuntimeSpec returned error: %v", err)
	}
	if spec.Image != defaultImage {
		t.Fatalf("image = %q, want %q", spec.Image, defaultImage)
	}
}

func TestResolveRuntimeSpecMissingDevcontainerStrict(t *testing.T) {
	worktree := t.TempDir()
	_, err := resolveRuntimeSpec(worktree, "", "missing.json", true)
	if err == nil || !strings.Contains(err.Error(), "devcontainer config not found") {
		t.Fatalf("expected strict missing error, got %v", err)
	}
}

func TestResolveRuntimeSpecExplicitImageStillAppliesDevcontainerFields(t *testing.T) {
	worktree := t.TempDir()
	dcDir := filepath.Join(worktree, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dcPath := filepath.Join(dcDir, "devcontainer.json")
	content := `{
		"image": "from-devcontainer",
		"runArgs": ["--privileged"],
		"containerEnv": {"FOO": "bar"},
		"remoteUser": "1000:1000",
		"mounts": ["type=bind,source=${localWorkspaceFolder},target=/src"],
		"workspaceMount": "type=bind,source=${localWorkspaceFolder},target=/workspace",
		"workspaceFolder": "/workspace/${localWorkspaceFolderBasename}"
	}`
	if err := os.WriteFile(dcPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write devcontainer: %v", err)
	}

	spec, err := resolveRuntimeSpec(worktree, "explicit-image:latest", "", true)
	if err != nil {
		t.Fatalf("resolveRuntimeSpec returned error: %v", err)
	}
	if spec.Image != "explicit-image:latest" {
		t.Fatalf("image = %q, want explicit image", spec.Image)
	}
	if len(spec.RunArgs) != 1 || spec.RunArgs[0] != "--privileged" {
		t.Fatalf("unexpected runArgs: %+v", spec.RunArgs)
	}
	if spec.ContainerEnv["FOO"] != "bar" {
		t.Fatalf("container env not applied: %+v", spec.ContainerEnv)
	}
	if spec.RemoteUser != "1000:1000" {
		t.Fatalf("remote user = %q", spec.RemoteUser)
	}
	if len(spec.Mounts) != 1 {
		t.Fatalf("mounts = %+v", spec.Mounts)
	}
	if spec.WorkspaceMount == "" || spec.WorkspaceFolder == "" {
		t.Fatalf("workspace fields not applied: mount=%q folder=%q", spec.WorkspaceMount, spec.WorkspaceFolder)
	}
}

func TestResolveRuntimeSpecUsesConfigImage(t *testing.T) {
	worktree := t.TempDir()
	dcDir := filepath.Join(worktree, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dcPath := filepath.Join(dcDir, "devcontainer.json")
	if err := os.WriteFile(dcPath, []byte(`{"image":"ghcr.io/example/dev:latest"}`), 0o644); err != nil {
		t.Fatalf("write devcontainer: %v", err)
	}

	spec, err := resolveRuntimeSpec(worktree, "", "", true)
	if err != nil {
		t.Fatalf("resolveRuntimeSpec returned error: %v", err)
	}
	if spec.Image != "ghcr.io/example/dev:latest" {
		t.Fatalf("image = %q, want config image", spec.Image)
	}
}

func TestResolveRuntimeSpecBuildsImage(t *testing.T) {
	origRun := runCommandFn
	t.Cleanup(func() { runCommandFn = origRun })

	worktree := t.TempDir()
	dcDir := filepath.Join(worktree, ".devcontainer")
	if err := os.MkdirAll(dcDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dockerfilePath := filepath.Join(dcDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte("FROM alpine:3.20"), 0o644); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}

	dcPath := filepath.Join(dcDir, "devcontainer.json")
	dcContent := `{"build":{"dockerfile":"Dockerfile","context":".","args":{"B":"2","A":"1"}}}`
	if err := os.WriteFile(dcPath, []byte(dcContent), 0o644); err != nil {
		t.Fatalf("write devcontainer: %v", err)
	}

	var gotName string
	var gotDir string
	var gotArgs []string
	runCommandFn = func(dir string, stdout, stderr io.Writer, name string, args ...string) error {
		gotDir = dir
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	spec, err := resolveRuntimeSpec(worktree, "", "", true)
	if err != nil {
		t.Fatalf("resolveRuntimeSpec returned error: %v", err)
	}
	if !strings.HasPrefix(spec.Image, "vibe-devcontainer:") {
		t.Fatalf("image = %q, want generated tag", spec.Image)
	}
	if gotDir != "" || gotName != "docker" {
		t.Fatalf("unexpected docker build invocation: dir=%q name=%q", gotDir, gotName)
	}
	if len(gotArgs) == 0 || gotArgs[0] != "build" {
		t.Fatalf("unexpected docker args: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "-f", dockerfilePath) {
		t.Fatalf("docker args missing dockerfile path: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "--build-arg", "A=1") || !runtimeHasPair(gotArgs, "--build-arg", "B=2") {
		t.Fatalf("docker args missing build args: %+v", gotArgs)
	}
	if runtimePairIndex(gotArgs, "--build-arg", "A=1") > runtimePairIndex(gotArgs, "--build-arg", "B=2") {
		t.Fatalf("build args should be sorted: %+v", gotArgs)
	}
}

func TestBuildDevcontainerImageValidation(t *testing.T) {
	dcPath := filepath.Join(t.TempDir(), "devcontainer.json")
	if _, err := buildDevcontainerImage(dcPath, devcontainerBuild{Dockerfile: "missing", Context: "."}); err == nil || !strings.Contains(err.Error(), "dockerfile not found") {
		t.Fatalf("expected missing dockerfile error, got %v", err)
	}

	base := t.TempDir()
	dcPath = filepath.Join(base, "devcontainer.json")
	dockerfile := filepath.Join(base, "Dockerfile")
	if err := os.WriteFile(dockerfile, []byte("FROM scratch"), 0o644); err != nil {
		t.Fatalf("write dockerfile: %v", err)
	}
	ctxFile := filepath.Join(base, "ctx.txt")
	if err := os.WriteFile(ctxFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("write context file: %v", err)
	}
	if _, err := buildDevcontainerImage(dcPath, devcontainerBuild{Dockerfile: "Dockerfile", Context: "ctx.txt"}); err == nil || !strings.Contains(err.Error(), "context is not a directory") {
		t.Fatalf("expected invalid context error, got %v", err)
	}
}

func TestExpandWorkspaceVariables(t *testing.T) {
	worktree := filepath.Join("/tmp", "sandbox", "feat-x")
	got := expandWorkspaceVariables(
		"${localWorkspaceFolder}|${localWorkspaceFolderBasename}",
		worktree,
	)
	want := worktree + "|feat-x"
	if got != want {
		t.Fatalf("expandWorkspaceVariables = %q, want %q", got, want)
	}
}

func TestPassthroughEnvs(t *testing.T) {
	keys := []string{
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OPENAI_ORG_ID",
		"OPENAI_PROJECT",
		"GITHUB_TOKEN",
		"GH_TOKEN",
		"ANTHROPIC_API_KEY",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("GH_TOKEN", "gh-test")

	got := passthroughEnvs()
	envMap := runtimeEnvMap(got)

	want := map[string]string{
		"OPENAI_API_KEY": "sk-test",
		"GH_TOKEN":       "gh-test",
	}
	if !reflect.DeepEqual(envMap, want) {
		t.Fatalf("passthroughEnvs = %+v, want %+v", envMap, want)
	}
}

func TestDefaultMounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte("[user]"), 0o644); err != nil {
		t.Fatalf("write .gitconfig: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".git-credentials"), []byte("https://example.invalid\n"), 0o644); err != nil {
		t.Fatalf("write .git-credentials: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o755); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755); err != nil {
		t.Fatalf("mkdir .config/opencode: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".local", "share", "opencode"), 0o755); err != nil {
		t.Fatalf("mkdir .local/share/opencode: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".local", "state", "opencode"), 0o755); err != nil {
		t.Fatalf("mkdir .local/state/opencode: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".cache", "opencode"), 0o755); err != nil {
		t.Fatalf("mkdir .cache/opencode: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "gh"), 0o755); err != nil {
		t.Fatalf("mkdir .config/gh: %v", err)
	}

	got := defaultMounts()
	mounts := runtimeFlagValues(got, "-v")
	sort.Strings(mounts)

	want := []string{
		filepath.Join(home, ".gitconfig") + ":/root/.gitconfig:ro",
		filepath.Join(home, ".git-credentials") + ":/root/.git-credentials:ro",
		filepath.Join(home, ".ssh") + ":/root/.ssh:ro",
		filepath.Join(home, ".config", "opencode") + ":/root/.config/opencode",
		filepath.Join(home, ".local", "share", "opencode") + ":/root/.local/share/opencode",
		filepath.Join(home, ".local", "state", "opencode") + ":/root/.local/state/opencode",
		filepath.Join(home, ".cache", "opencode") + ":/root/.cache/opencode",
		filepath.Join(home, ".config", "gh") + ":/root/.config/gh:ro",
	}
	sort.Strings(want)

	if !reflect.DeepEqual(mounts, want) {
		t.Fatalf("defaultMounts mounts = %+v, want %+v", mounts, want)
	}
}

func TestRunOpenCodeContainerBuildsDockerArgs(t *testing.T) {
	origInteractive := interactiveCommandFn
	t.Cleanup(func() { interactiveCommandFn = origInteractive })

	keys := []string{
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OPENAI_ORG_ID",
		"OPENAI_PROJECT",
		"GITHUB_TOKEN",
		"GH_TOKEN",
		"ANTHROPIC_API_KEY",
	}
	for _, k := range keys {
		t.Setenv(k, "")
	}

	meta := &sandboxMeta{
		Name:      "feat",
		Worktree:  filepath.Join(t.TempDir(), "feat-worktree"),
		Container: "codex-sb-feat",
	}
	if err := os.MkdirAll(meta.Worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	runtime := &runtimeSpec{
		Image:           "ghcr.io/example/codex:latest",
		RunArgs:         []string{"--network", "host"},
		ContainerEnv:    map[string]string{"PROJECT_DIR": "${localWorkspaceFolderBasename}"},
		RemoteUser:      "1000:1000",
		Mounts:          []string{"type=bind,source=${localWorkspaceFolder},target=/src"},
		WorkspaceMount:  "type=bind,source=${localWorkspaceFolder},target=/workspace",
		WorkspaceFolder: "/workspace/${localWorkspaceFolderBasename}",
	}

	var gotName string
	var gotArgs []string
	interactiveCommandFn = func(name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := runOpenCodeContainer(meta, runtime, "echo hi"); err != nil {
		t.Fatalf("runOpenCodeContainer returned error: %v", err)
	}
	if gotName != "docker" {
		t.Fatalf("interactive name = %q, want docker", gotName)
	}
	if !runtimeHasSequence(gotArgs, []string{"run", "--rm", "-it", "--name", "codex-sb-feat"}) {
		t.Fatalf("missing docker run prelude: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "--mount", "type=bind,source="+meta.Worktree+",target=/workspace") {
		t.Fatalf("missing workspace mount: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "-w", "/workspace/"+filepath.Base(meta.Worktree)) {
		t.Fatalf("missing workspace folder: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "--mount", "type=bind,source="+meta.Worktree+",target=/src") {
		t.Fatalf("missing extra mount: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "--user", "1000:1000") {
		t.Fatalf("missing remote user: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "-e", "PROJECT_DIR="+filepath.Base(meta.Worktree)) {
		t.Fatalf("missing container env: %+v", gotArgs)
	}
	if !runtimeHasSequence(gotArgs, []string{"--network", "host"}) {
		t.Fatalf("missing run args: %+v", gotArgs)
	}
	if !runtimeHasSuffix(gotArgs, []string{"ghcr.io/example/codex:latest", "bash", "-lc", "echo hi"}) {
		t.Fatalf("missing image/command suffix: %+v", gotArgs)
	}
}

func TestRunOpenCodeContainerDefaultsRuntime(t *testing.T) {
	origInteractive := interactiveCommandFn
	t.Cleanup(func() { interactiveCommandFn = origInteractive })

	meta := &sandboxMeta{Worktree: filepath.Join(t.TempDir(), "wt"), Container: "codex-sb-default"}
	if err := os.MkdirAll(meta.Worktree, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	var gotArgs []string
	interactiveCommandFn = func(name string, args ...string) error {
		if name != "docker" {
			return fmt.Errorf("unexpected command %q", name)
		}
		gotArgs = append([]string(nil), args...)
		return nil
	}

	if err := runOpenCodeContainer(meta, nil, "pwd"); err != nil {
		t.Fatalf("runOpenCodeContainer returned error: %v", err)
	}
	if !runtimeHasPair(gotArgs, "-v", meta.Worktree+":/workspace") {
		t.Fatalf("missing default workspace volume: %+v", gotArgs)
	}
	if !runtimeHasPair(gotArgs, "-w", "/workspace") {
		t.Fatalf("missing default workspace folder: %+v", gotArgs)
	}
	if !runtimeHasSuffix(gotArgs, []string{defaultImage, "bash", "-lc", "pwd"}) {
		t.Fatalf("missing default image suffix: %+v", gotArgs)
	}
}

func runtimeHasPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func runtimePairIndex(args []string, flag, value string) int {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return i
		}
	}
	return -1
}

func runtimeHasSequence(args, seq []string) bool {
	if len(seq) == 0 || len(seq) > len(args) {
		return false
	}
	for i := 0; i <= len(args)-len(seq); i++ {
		ok := true
		for j := range seq {
			if args[i+j] != seq[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func runtimeHasSuffix(args, suffix []string) bool {
	if len(suffix) > len(args) {
		return false
	}
	offset := len(args) - len(suffix)
	for i := range suffix {
		if args[offset+i] != suffix[i] {
			return false
		}
	}
	return true
}

func runtimeFlagValues(args []string, flag string) []string {
	values := make([]string, 0)
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			values = append(values, args[i+1])
			i++
		}
	}
	return values
}

func runtimeEnvMap(args []string) map[string]string {
	result := map[string]string{}
	for _, pair := range runtimeFlagValues(args, "-e") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1]
	}
	return result
}
