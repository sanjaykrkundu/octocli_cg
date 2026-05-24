package octocli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sanja/octocli_cg/internal/brain"
	"github.com/spf13/cobra"
)

func taskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage artifact-tracked tasks in .agents/brain",
	}

	cmd.AddCommand(taskCreateCmd())
	cmd.AddCommand(taskListCmd())
	cmd.AddCommand(taskShowCmd())
	cmd.AddCommand(taskCheckCmd())
	return cmd
}

func taskCreateCmd() *cobra.Command {
	var checklist []string
	var plan string

	cmd := &cobra.Command{
		Use:   "create [goal]",
		Short: "Create a tracked task with checklist and implementation plan",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			store := brain.Store{WorkspaceRoot: workspaceRoot}
			task, err := store.Create(strings.Join(args, " "), checklist, plan)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "created task %s at %s\n", task.Metadata.ID, task.Dir)
			return err
		},
	}

	cmd.Flags().StringArrayVar(&checklist, "item", nil, "checklist item (repeatable)")
	cmd.Flags().StringVar(&plan, "plan", "", "initial implementation plan markdown")
	return cmd
}

func taskListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tracked tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			store := brain.Store{WorkspaceRoot: workspaceRoot}
			items, err := store.List()
			if err != nil {
				return err
			}
			if len(items) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "no tasks found")
				return err
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s [%s] %s\n", item.ID, item.Status, item.Goal)
			}
			return nil
		},
	}
}

func taskShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [id]",
		Short: "Show task files and metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			store := brain.Store{WorkspaceRoot: workspaceRoot}
			task, err := store.Get(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"ID: %s\nStatus: %s\nDir: %s\n\n%s\n\n%s\n",
				task.Metadata.ID,
				task.Metadata.Status,
				task.Dir,
				task.TaskMarkdown,
				task.ImplementationPlan,
			)
			return err
		},
	}
}

func taskCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check [id] [index] [true|false]",
		Short: "Mark a checklist item complete or incomplete",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			index, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("parse checklist index: %w", err)
			}
			completed, err := strconv.ParseBool(args[2])
			if err != nil {
				return fmt.Errorf("parse completed value: %w", err)
			}
			store := brain.Store{WorkspaceRoot: workspaceRoot}
			task, err := store.SetChecklistItem(args[0], index, completed)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "updated task %s status=%s\n", task.Metadata.ID, task.Metadata.Status)
			return err
		},
	}
}
