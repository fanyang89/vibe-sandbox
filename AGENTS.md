# AGENTS.md

Guidance for agentic coding assistants working in this repository.

## Project Snapshot
- Language: Go
- Module: `github.com/fanyang89/vibe-docker`
- Go version: `1.25.5` (from `go.mod`)
- CLI entrypoint package: `./cmd/vibe`
- CLI framework: `github.com/spf13/cobra`
- Runtime integrations: `git`, `docker`, `gh`

## Repository Layout
- `cmd/vibe/main.go`: process entrypoint.
- `cmd/vibe/root_cmd.go`: root command and subcommand registration.
- `cmd/vibe/cmd_*.go`: command-specific flags + `RunE` handlers.
- `cmd/vibe/manager.go`: sandbox metadata and worktree lifecycle.
- `cmd/vibe/runtime.go`: devcontainer parsing and Docker runtime resolution.
- `cmd/vibe/ops.go`: base-ref resolution and PR creation helpers.
- `cmd/vibe/exec.go`: shell execution wrappers.
- `cmd/vibe/naming.go`: name normalization/validation/hash helpers.
- `cmd/vibe/types.go`: shared constants and option/data structs.
- `cmd/vibe/main_test.go`: current unit tests.

## Build Commands
- Build binary: `go build -o bin/vibe ./cmd/vibe`
- Build all packages: `go build ./...`
- Run without prebuild: `go run ./cmd/vibe --help`
- Optional default sandbox image:
  `docker build -t codex-sandbox:latest -f docker/Dockerfile.codex-sandbox .`

## Test Commands
- Run all tests: `go test ./...`
- Run CLI package tests: `go test ./cmd/vibe`
- Run one exact test: `go test ./cmd/vibe -run '^TestNormalizeName$'`
- Run selected tests by regex:
  `go test ./cmd/vibe -run 'TestNormalizeName|TestValidName'`
- Verbose mode: `go test -v ./cmd/vibe`
- Disable cache while iterating:
  `go test ./cmd/vibe -run '^TestNormalizeName$' -count=1`
- Quick coverage check: `go test ./cmd/vibe -cover`

## Lint, Vet, and Formatting

This repo has no dedicated lint config. Use Go-native checks as baseline.

- Format all packages: `go fmt ./...`
- Check non-gofmt files: `gofmt -l cmd/vibe/*.go`
- Apply gofmt directly: `gofmt -w cmd/vibe/*.go`
- Run vet: `go vet ./...`
- Optional extra static analysis (if installed): `staticcheck ./...`

## Code Style Guidelines

### General
- Keep changes scoped to requested behavior.
- Prefer small focused functions over multi-responsibility blocks.
- Avoid speculative abstractions.
- Do not add dependencies without clear justification.

### Formatting
- Keep all Go code `gofmt`-clean.
- Trust standard Go formatting; do not hand-align for style.
- Add comments only when behavior is non-obvious.

### Imports
- Use standard Go grouping order:
  1. standard library
  2. blank line
  3. third-party packages
- Let `gofmt` handle sorting.
- Remove unused imports immediately.

### Types and Data Modeling
- Prefer concrete types until an interface is truly required.
- Keep broadly reused constants/types in `cmd/vibe/types.go`.
- Use explicit JSON tags for serialized metadata.
- Follow existing JSON tag convention (`snake_case`).

### Naming
- Follow idiomatic Go naming:
  - exported identifiers: `PascalCase`
  - unexported identifiers: `camelCase`
- Keep command constructor pattern consistent (`newGoCmd`, `newDoneCmd`).
- Use clear verb-based names (`resolveRuntimeSpec`, `createPR`).

### Error Handling
- Return errors; do not panic for expected runtime/validation failures.
- Wrap low-level errors with context using `%w`.
- Use `errors.New` for fixed validation messages.
- Keep error text concise, lowercase, and actionable.
- Include operation context in wrapped errors for debugging clarity.
- Best-effort cleanup may ignore errors when explicitly intentional.

### Command Execution and Side Effects
- Reuse helpers from `cmd/vibe/exec.go`.
- Avoid duplicating command-shell invocation logic.
- Include command context when propagating failures.
- Prefer deterministic ordering when generating args/output.

### File and Path Handling
- Use `filepath` helpers (`Join`, `IsAbs`, `Clean`, `Base`).
- Validate files/directories before relying on them.
- Preserve existing atomic write style (tmp file + rename).

### CLI Behavior and UX
- Validate flags early at the start of `RunE`.
- Initialize manager once per command and pass through helpers.
- Keep success output concise and user-facing.
- Preserve hidden compatibility commands unless explicitly asked to change.

## Testing Guidelines
- Prefer table-driven tests for pure logic.
- Keep tests deterministic and fast.
- Use precise failure messages with expected vs actual values.
- Add/update tests when changing:
  - naming normalization/validation
  - base-ref or branch logic
  - runtime spec resolution behavior

## Agent Workflow Checklist

Before finalizing changes, run:
- `go fmt ./...`
- `go test ./...`
- `go vet ./...`
- `go build ./...`

If CLI behavior changed, also smoke test:
- `go run ./cmd/vibe --help`

## Cursor/Copilot Rules

Checked locations:
- `.cursor/rules/`
- `.cursorrules`
- `.github/copilot-instructions.md`

Current repository status:
- No Cursor rules were found.
- No Copilot instruction file was found.

If these files are added later, treat them as authoritative project policy and
update this document accordingly.
