package octocli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	appcontext "github.com/sanja/octocli_cg/internal/context"
	"github.com/spf13/cobra"
)

func workflowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Inspect saved slash-command workflows",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List discovered workflows",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			bundle, err := appcontext.Load(workspaceRoot)
			if err != nil {
				return err
			}
			if len(bundle.Workflows) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "no workflows found")
				return err
			}
			names := make([]string, 0, len(bundle.Workflows))
			for name := range bundle.Workflows {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				wf := bundle.Workflows[name]
				fmt.Fprintf(cmd.OutOrStdout(), "/%s (%s) -> %s\n", wf.Name, wf.Source, wf.Path)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show [name]",
		Short: "Show a workflow template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			bundle, err := appcontext.Load(workspaceRoot)
			if err != nil {
				return err
			}
			wf, ok := bundle.Workflows[strings.TrimPrefix(args[0], "/")]
			if !ok {
				return fmt.Errorf("workflow %q not found", args[0])
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "# /%s\n# source: %s\n%s\n", wf.Name, wf.Source, wf.Content)
			return err
		},
	})

	return cmd
}
