package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tailscale/hujson"
)

func resolveRuntimeSpec(worktree, explicitImage, devcontainerPath string, strictDevcontainer bool) (*runtimeSpec, error) {
	spec := &runtimeSpec{
		Image:        defaultImage,
		ContainerEnv: map[string]string{},
	}
	if explicitImage != "" {
		spec.Image = explicitImage
	}

	dcPath := devcontainerPath
	if dcPath == "" {
		dcPath = ".devcontainer/devcontainer.json"
	}
	if !filepath.IsAbs(dcPath) {
		dcPath = filepath.Join(worktree, dcPath)
	}

	_, statErr := os.Stat(dcPath)
	if statErr != nil {
		if strictDevcontainer {
			return nil, fmt.Errorf("devcontainer config not found: %s", dcPath)
		}
		if explicitImage == "" {
			spec.Image = defaultImage
		}
		return spec, nil
	}

	cfg, err := readDevcontainerConfig(dcPath)
	if err != nil {
		return nil, err
	}
	spec.RunArgs = append(spec.RunArgs, cfg.RunArgs...)
	for k, v := range cfg.ContainerEnv {
		spec.ContainerEnv[k] = v
	}
	spec.RemoteUser = cfg.RemoteUser
	spec.Mounts = append(spec.Mounts, cfg.Mounts...)
	spec.WorkspaceMount = cfg.WorkspaceMount
	spec.WorkspaceFolder = cfg.WorkspaceFolder

	if explicitImage != "" {
		return spec, nil
	}
	if cfg.Image != "" {
		spec.Image = cfg.Image
		return spec, nil
	}

	build, hasBuild, err := parseDevcontainerBuild(cfg)
	if err != nil {
		return nil, err
	}
	if !hasBuild {
		return spec, nil
	}

	image, err := buildDevcontainerImage(dcPath, build)
	if err != nil {
		return nil, err
	}
	spec.Image = image
	return spec, nil
}

func readDevcontainerConfig(path string) (*devcontainerConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read devcontainer config: %w", err)
	}
	standard, err := hujson.Standardize(raw)
	if err != nil {
		return nil, fmt.Errorf("parse devcontainer config: %w", err)
	}
	var cfg devcontainerConfig
	if err := json.Unmarshal(standard, &cfg); err != nil {
		return nil, fmt.Errorf("decode devcontainer config: %w", err)
	}
	return &cfg, nil
}

func parseDevcontainerBuild(cfg *devcontainerConfig) (devcontainerBuild, bool, error) {
	build := devcontainerBuild{Args: map[string]string{}}
	if cfg.DockerFile != "" {
		build.Dockerfile = cfg.DockerFile
	}
	if cfg.Context != "" {
		build.Context = cfg.Context
	}
	if len(cfg.Build) == 0 {
		if build.Dockerfile == "" && build.Context == "" {
			return build, false, nil
		}
		return build, true, nil
	}

	trimmed := strings.TrimSpace(string(cfg.Build))
	if strings.HasPrefix(trimmed, "\"") {
		var dockerfile string
		if err := json.Unmarshal(cfg.Build, &dockerfile); err != nil {
			return build, false, fmt.Errorf("decode devcontainer build string: %w", err)
		}
		if dockerfile != "" {
			build.Dockerfile = dockerfile
		}
		return build, true, nil
	}

	var fromObject devcontainerBuild
	if err := json.Unmarshal(cfg.Build, &fromObject); err != nil {
		return build, false, fmt.Errorf("decode devcontainer build object: %w", err)
	}
	if fromObject.Dockerfile != "" {
		build.Dockerfile = fromObject.Dockerfile
	}
	if fromObject.Context != "" {
		build.Context = fromObject.Context
	}
	if len(fromObject.Args) > 0 {
		build.Args = fromObject.Args
	}
	return build, true, nil
}

func buildDevcontainerImage(devcontainerPath string, build devcontainerBuild) (string, error) {
	baseDir := filepath.Dir(devcontainerPath)
	dockerfile := build.Dockerfile
	if dockerfile == "" {
		dockerfile = "Dockerfile"
	}
	contextPath := build.Context
	if contextPath == "" {
		contextPath = "."
	}

	if !filepath.IsAbs(dockerfile) {
		dockerfile = filepath.Join(baseDir, dockerfile)
	}
	if !filepath.IsAbs(contextPath) {
		contextPath = filepath.Join(baseDir, contextPath)
	}

	if _, err := os.Stat(dockerfile); err != nil {
		return "", fmt.Errorf("devcontainer dockerfile not found: %s", dockerfile)
	}
	if stat, err := os.Stat(contextPath); err != nil || !stat.IsDir() {
		return "", fmt.Errorf("devcontainer context is not a directory: %s", contextPath)
	}

	tag := "vibe-devcontainer:" + shortHash(devcontainerPath+"|"+dockerfile+"|"+contextPath)
	args := []string{"build", "-t", tag, "-f", dockerfile}
	if len(build.Args) > 0 {
		keys := make([]string, 0, len(build.Args))
		for k := range build.Args {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			args = append(args, "--build-arg", k+"="+build.Args[k])
		}
	}
	args = append(args, contextPath)

	if err := runCommand("", os.Stdout, os.Stderr, "docker", args...); err != nil {
		return "", fmt.Errorf("build devcontainer image: %w", err)
	}
	return tag, nil
}

func runCodexContainer(meta *sandboxMeta, runtime *runtimeSpec, command string) error {
	if runtime == nil {
		runtime = &runtimeSpec{Image: defaultImage}
	}
	if runtime.Image == "" {
		runtime.Image = defaultImage
	}

	workspaceFolder := "/workspace"
	if runtime.WorkspaceFolder != "" {
		workspaceFolder = expandWorkspaceVariables(runtime.WorkspaceFolder, meta.Worktree)
	}

	dockerArgs := []string{"run", "--rm", "-it", "--name", meta.Container}
	if runtime.WorkspaceMount != "" {
		dockerArgs = append(dockerArgs, "--mount", expandWorkspaceVariables(runtime.WorkspaceMount, meta.Worktree))
	} else {
		dockerArgs = append(dockerArgs, "-v", meta.Worktree+":"+workspaceFolder)
	}
	dockerArgs = append(dockerArgs, "-w", workspaceFolder)

	for _, mount := range runtime.Mounts {
		dockerArgs = append(dockerArgs, "--mount", expandWorkspaceVariables(mount, meta.Worktree))
	}
	if runtime.RemoteUser != "" {
		dockerArgs = append(dockerArgs, "--user", runtime.RemoteUser)
	}

	dockerArgs = append(dockerArgs, defaultMounts()...)
	dockerArgs = append(dockerArgs, passthroughEnvs()...)
	for k, v := range runtime.ContainerEnv {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", k, expandWorkspaceVariables(v, meta.Worktree)))
	}
	if len(runtime.RunArgs) > 0 {
		dockerArgs = append(dockerArgs, runtime.RunArgs...)
	}

	dockerArgs = append(dockerArgs, runtime.Image, "bash", "-lc", command)

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	return nil
}

func defaultMounts() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	mounts := []struct {
		host string
		ctr  string
		ro   bool
	}{
		{host: filepath.Join(home, ".gitconfig"), ctr: "/root/.gitconfig", ro: true},
		{host: filepath.Join(home, ".git-credentials"), ctr: "/root/.git-credentials", ro: true},
		{host: filepath.Join(home, ".ssh"), ctr: "/root/.ssh", ro: true},
		{host: filepath.Join(home, ".codex"), ctr: "/root/.codex", ro: false},
		{host: filepath.Join(home, ".config", "gh"), ctr: "/root/.config/gh", ro: true},
	}
	args := make([]string, 0, len(mounts)*2)
	for _, mount := range mounts {
		if _, err := os.Stat(mount.host); err != nil {
			continue
		}
		value := mount.host + ":" + mount.ctr
		if mount.ro {
			value += ":ro"
		}
		args = append(args, "-v", value)
	}
	return args
}

func passthroughEnvs() []string {
	keys := []string{
		"OPENAI_API_KEY",
		"OPENAI_BASE_URL",
		"OPENAI_ORG_ID",
		"OPENAI_PROJECT",
		"GITHUB_TOKEN",
		"GH_TOKEN",
		"ANTHROPIC_API_KEY",
	}
	args := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if !ok || value == "" {
			continue
		}
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}
	return args
}

func expandWorkspaceVariables(value, worktree string) string {
	result := value
	result = strings.ReplaceAll(result, "${localWorkspaceFolder}", worktree)
	result = strings.ReplaceAll(result, "${localWorkspaceFolderBasename}", filepath.Base(worktree))
	return result
}

func runningContainers() map[string]bool {
	result := map[string]bool{}
	out, err := commandOutput("", "docker", "ps", "--format", "{{.Names}}")
	if err != nil {
		return result
	}
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		result[name] = true
	}
	return result
}
