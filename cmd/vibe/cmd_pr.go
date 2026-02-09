package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func newPRCmd(rootOpts *rootOptions) *cobra.Command {
	opts := prOptions{}
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Push branch and create PR for sandbox",
		RunE: func(_ *cobra.Command, _ []string) error {
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
			return createPR(meta, opts.base, opts.title, opts.body, opts.draft)
		},
	}
	cmd.Flags().StringVar(&opts.name, "name", "", "sandbox name")
	cmd.Flags().StringVar(&opts.base, "base", "", "target base branch")
	cmd.Flags().StringVar(&opts.title, "title", "", "PR title")
	cmd.Flags().StringVar(&opts.body, "body", "", "PR body")
	cmd.Flags().BoolVar(&opts.draft, "draft", false, "create draft PR")
	return cmd
}
