package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd(rootOpts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all sandboxes",
		RunE: func(_ *cobra.Command, _ []string) error {
			mgr, err := newManager(rootOpts.sandboxRoot)
			if err != nil {
				return fmt.Errorf("init failed: %w", err)
			}

			metas, err := mgr.listSandboxes()
			if err != nil {
				return err
			}
			sort.Slice(metas, func(i, j int) bool { return metas[i].Name < metas[j].Name })

			running := runningContainers()
			w := tabwriter.NewWriter(os.Stdout, 4, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tBRANCH\tBASE\tWORKTREE\tRUNNING")
			for _, meta := range metas {
				_, err := os.Stat(meta.Worktree)
				exists := err == nil
				status := "no"
				if running[meta.Container] {
					status = "yes"
				}
				if !exists {
					status = "missing-worktree"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", meta.Name, meta.Branch, meta.BaseRef, meta.Worktree, status)
			}
			w.Flush()
			return nil
		},
	}
}
