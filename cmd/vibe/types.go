package main

import "encoding/json"

const (
	defaultSandboxDir   = ".codex-sandboxes"
	defaultBranchPrefix = "codex"
	defaultImage        = "codex-sandbox:latest"
	defaultRunCommand   = "codex --yolo"
)

type manager struct {
	repoRoot    string
	sandboxRoot string
	metaDir     string
}

type sandboxMeta struct {
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	BaseRef   string `json:"base_ref"`
	Worktree  string `json:"worktree"`
	Container string `json:"container"`
	CreatedAt string `json:"created_at"`
}

type rootOptions struct {
	sandboxRoot string
}

type goOptions struct {
	name         string
	base         string
	branchPrefix string
	image        string
	command      string
	devcontainer string
}

type doneOptions struct {
	name         string
	all          bool
	force        bool
	deleteBranch bool
	createPR     bool
	base         string
	title        string
	body         string
	draft        bool
}

type prOptions struct {
	name  string
	base  string
	title string
	body  string
	draft bool
}

type createOptions struct {
	name         string
	base         string
	branchPrefix string
}

type runOptions struct {
	name         string
	image        string
	command      string
	devcontainer string
}

type destroyOptions struct {
	name         string
	all          bool
	force        bool
	deleteBranch bool
}

type runtimeSpec struct {
	Image           string
	RunArgs         []string
	ContainerEnv    map[string]string
	RemoteUser      string
	Mounts          []string
	WorkspaceMount  string
	WorkspaceFolder string
}

type devcontainerConfig struct {
	Image           string            `json:"image"`
	Build           json.RawMessage   `json:"build"`
	DockerFile      string            `json:"dockerFile"`
	Context         string            `json:"context"`
	RunArgs         []string          `json:"runArgs"`
	ContainerEnv    map[string]string `json:"containerEnv"`
	RemoteUser      string            `json:"remoteUser"`
	Mounts          []string          `json:"mounts"`
	WorkspaceMount  string            `json:"workspaceMount"`
	WorkspaceFolder string            `json:"workspaceFolder"`
}

type devcontainerBuild struct {
	Dockerfile string            `json:"dockerfile"`
	Context    string            `json:"context"`
	Args       map[string]string `json:"args"`
}
