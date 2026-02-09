# vibe

`vibe` is a sandbox orchestrator for Codex workflows.
It creates isolated `git worktree` environments, launches Docker, runs `codex --yolo`, and then cleans everything up.

## Workflow

- `vibe go`: create worktree + start container + run Codex
- `vibe done`: optionally create PR, then destroy resources
- `vibe done --all`: one-click destroy all sandboxes

## Features

- High-concurrency sandbox model: each sandbox has its own worktree, branch, metadata, and container name
- Docker image customization via `--image`
- Devcontainer-compatible runtime resolution via `--devcontainer`
- Optional PR creation with `gh` before cleanup

## Build

```bash
go build -o bin/vibe ./cmd/vibe
```

## Quick Start

1. Build a default image (optional, used when no devcontainer/image is provided):

```bash
docker build -t codex-sandbox:latest -f docker/Dockerfile.codex-sandbox .
```

2. Start a sandbox and run Codex interactively:

```bash
./bin/vibe go --name feat-login --base main
```

3. Finish work, open PR, and clean resources:

```bash
./bin/vibe done --name feat-login --pr --base main
```

## Command Reference

```bash
# Start a sandbox (auto-name if --name is omitted)
./bin/vibe go --name feat-login --base main

# Use a custom image
./bin/vibe go --name feat-login --image ghcr.io/acme/codex:latest

# Use a custom devcontainer config path
./bin/vibe go --name feat-login --devcontainer .devcontainer/devcontainer.json

# Cleanup one sandbox
./bin/vibe done --name feat-login

# Force cleanup dirty worktree and delete branch
./bin/vibe done --name feat-login --force --delete-branch

# One-click cleanup for all sandboxes
./bin/vibe done --all

# One-click cleanup for all sandboxes and delete local branches
./bin/vibe done --all --delete-branch

# Inspect current sandbox state
./bin/vibe list
```

## Devcontainer Compatibility

When `--devcontainer` points to a valid `devcontainer.json`, `vibe` supports these fields:

- `image`
- `build` (string or object, with `dockerfile`, `context`, `args`)
- `dockerFile`
- `context`
- `runArgs`
- `containerEnv`
- `remoteUser`
- `mounts`
- `workspaceMount`
- `workspaceFolder`

Resolution order:

1. `--image` (highest priority)
2. `devcontainer.image`
3. Build from devcontainer Dockerfile/context
4. Fallback to `codex-sandbox:latest`

## Host Mounts and Env Passthrough

When available, `vibe` mounts:

- `~/.gitconfig`
- `~/.git-credentials`
- `~/.ssh`
- `~/.codex`
- `~/.config/gh`

And forwards these env vars when present:

- `OPENAI_API_KEY`
- `OPENAI_BASE_URL`
- `OPENAI_ORG_ID`
- `OPENAI_PROJECT`
- `GITHUB_TOKEN`
- `GH_TOKEN`
- `ANTHROPIC_API_KEY`

## Notes

- `vibe done --all` does not create PRs. Use per-sandbox `vibe done --name <name> --pr` if you need PR creation.
- `vibe pr` remains available for explicit PR creation.
- Hidden compatibility commands still exist: `create`, `run`, `destroy`.
