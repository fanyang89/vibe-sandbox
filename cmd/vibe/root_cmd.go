package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	rootOpts := rootOptions{}

	root := &cobra.Command{
		Use:          "vibe",
		Short:        "Manage opencode worktree sandboxes",
		SilenceUsage: true,
		Long:         "vibe creates git-worktree + docker sandboxes and runs opencode.",
	}
	root.PersistentFlags().StringVar(&rootOpts.sandboxRoot, "sandbox-root", "", "sandbox root path (default: <repo>/.opencode-sandboxes)")

	root.AddCommand(newGoCmd(&rootOpts))
	root.AddCommand(newDoneCmd(&rootOpts))
	root.AddCommand(newListCmd(&rootOpts))
	root.AddCommand(newPRCmd(&rootOpts))

	// Compatibility subcommands.
	root.AddCommand(newCreateCmd(&rootOpts))
	root.AddCommand(newRunCmd(&rootOpts))
	root.AddCommand(newDestroyCmd(&rootOpts))

	return root
}
