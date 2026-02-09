package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newDoneCmd(rootOpts *rootOptions) *cobra.Command {
	opts := doneOptions{}
	cmd := &cobra.Command{
		Use:   "done",
		Short: "Cleanup sandbox resources (optionally create PR first)",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := newManager(rootOpts.sandboxRoot)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}

			if opts.all {
				if opts.name != "" {
					return errors.New("--name cannot be used with --all")
				}
				if opts.createPR {
					return errors.New("--pr cannot be used with --all")
				}
				count, err := mgr.destroyAllSandboxes(opts.force, opts.deleteBranch)
				if err != nil {
					return err
				}
				fmt.Printf("done: cleaned %d sandbox(es)\n", count)
				return nil
			}

			if opts.name == "" {
				return errors.New("either --name or --all is required")
			}
			meta, err := mgr.loadSandbox(opts.name)
			if err != nil {
				return err
			}
			if opts.createPR {
				if err := createPR(meta, opts.base, opts.title, opts.body, opts.draft); err != nil {
					return err
				}
			}

			if err := mgr.destroySandbox(meta, opts.force, opts.deleteBranch); err != nil {
				return err
			}
			fmt.Printf("done: cleaned sandbox %s\n", meta.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name")
	cmd.Flags().BoolVar(&opts.all, "all", false, "cleanup all sandboxes")
	cmd.Flags().BoolVar(&opts.force, "force", false, "force remove dirty worktree")
	cmd.Flags().BoolVar(&opts.deleteBranch, "delete-branch", true, "delete local branch after worktree removal")
	cmd.Flags().BoolVar(&opts.createPR, "pr", false, "create PR before cleanup")
	cmd.Flags().StringVar(&opts.base, "base", "", "target base branch (used with --pr)")
	cmd.Flags().StringVar(&opts.title, "title", "", "PR title (used with --pr)")
	cmd.Flags().StringVar(&opts.body, "body", "", "PR body (used with --pr)")
	cmd.Flags().BoolVar(&opts.draft, "draft", false, "create draft PR (used with --pr)")
	return cmd
}
