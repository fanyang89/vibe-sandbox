package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newCreateCmd(rootOpts *rootOptions) *cobra.Command {
	opts := createOptions{}
	cmd := &cobra.Command{
		Use:    "create",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
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
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name (auto-generated if omitted)")
	cmd.Flags().StringVar(&opts.base, "base", "", "base branch/ref (defaults to current branch)")
	cmd.Flags().StringVar(&opts.branchPrefix, "branch-prefix", defaultBranchPrefix, "sandbox branch prefix")
	return cmd
}

func newRunCmd(rootOpts *rootOptions) *cobra.Command {
	opts := runOptions{}
	cmd := &cobra.Command{
		Use:    "run",
		Hidden: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if opts.name == "" {
				return errors.New("--name is required")
			}
			mgr, err := newManager(rootOpts.sandboxRoot)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}
			meta, err := mgr.loadSandbox(opts.name)
			if err != nil {
				return err
			}
			runtime, err := resolveRuntimeSpec(meta.Worktree, opts.image, opts.devcontainer, cmd.Flags().Changed("devcontainer"))
			if err != nil {
				return err
			}
			return runCodexContainer(meta, runtime, opts.command)
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name")
	cmd.Flags().StringVar(&opts.image, "image", "", "docker image to run (overrides devcontainer image/build)")
	cmd.Flags().StringVar(&opts.command, "cmd", defaultRunCommand, "command executed in container")
	cmd.Flags().StringVar(&opts.devcontainer, "devcontainer", ".devcontainer/devcontainer.json", "devcontainer.json path relative to worktree")
	return cmd
}

func newDestroyCmd(rootOpts *rootOptions) *cobra.Command {
	opts := destroyOptions{}
	cmd := &cobra.Command{
		Use:    "destroy",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := newManager(rootOpts.sandboxRoot)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}

			if opts.all {
				if opts.name != "" {
					return errors.New("--name cannot be used with --all")
				}
				count, err := mgr.destroyAllSandboxes(opts.force, opts.deleteBranch)
				if err != nil {
					return err
				}
				fmt.Printf("destroyed %d sandbox(es)\n", count)
				return nil
			}
			if opts.name == "" {
				return errors.New("either --name or --all is required")
			}
			meta, err := mgr.loadSandbox(opts.name)
			if err != nil {
				return err
			}
			if err := mgr.destroySandbox(meta, opts.force, opts.deleteBranch); err != nil {
				return err
			}
			fmt.Printf("destroyed sandbox %s\n", meta.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name")
	cmd.Flags().BoolVar(&opts.all, "all", false, "destroy all sandboxes")
	cmd.Flags().BoolVar(&opts.force, "force", false, "force remove dirty worktree")
	cmd.Flags().BoolVar(&opts.deleteBranch, "delete-branch", false, "delete sandbox branch after worktree removal")
	return cmd
}
