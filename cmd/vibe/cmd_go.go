package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGoCmd(rootOpts *rootOptions) *cobra.Command {
	opts := goOptions{}
	cmd := &cobra.Command{
		Use:   "go",
		Short: "Create worktree, start docker, and run codex --yolo",
		RunE: func(cmd *cobra.Command, _ []string) error {
			mgr, err := newManager(rootOpts.sandboxRoot)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}

			name := normalizeName(opts.name)
			if name == "" {
				name = generateName()
			}

			baseRef, err := resolveBaseRef(mgr.repoRoot, opts.base)
			if err != nil {
				return err
			}

			meta, err := mgr.createSandbox(name, baseRef, opts.branchPrefix)
			if err != nil {
				return err
			}

			fmt.Printf("created sandbox %s\n", meta.Name)
			fmt.Printf("worktree: %s\n", meta.Worktree)
			fmt.Printf("branch:   %s\n", meta.Branch)

			runtime, err := resolveRuntimeSpec(meta.Worktree, opts.image, opts.devcontainer, cmd.Flags().Changed("devcontainer"))
			if err != nil {
				return fmt.Errorf("resolve runtime failed; sandbox is preserved, use `vibe done --name %s` to cleanup: %w", meta.Name, err)
			}

			if err := runCodexContainer(meta, runtime, opts.command); err != nil {
				return fmt.Errorf("run codex failed; sandbox is preserved, use `vibe done --name %s` to cleanup: %w", meta.Name, err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name (auto-generated if omitted)")
	cmd.Flags().StringVar(&opts.base, "base", "", "base branch/ref (defaults to current branch)")
	cmd.Flags().StringVar(&opts.branchPrefix, "branch-prefix", defaultBranchPrefix, "sandbox branch prefix")
	cmd.Flags().StringVar(&opts.image, "image", "", "docker image to run (overrides devcontainer image/build)")
	cmd.Flags().StringVar(&opts.command, "cmd", defaultRunCommand, "command executed in container")
	cmd.Flags().StringVar(&opts.devcontainer, "devcontainer", ".devcontainer/devcontainer.json", "devcontainer.json path relative to worktree")
	return cmd
}
