package octocli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sanja/octocli_cg/internal/agent"
	appcontext "github.com/sanja/octocli_cg/internal/context"
	"github.com/sanja/octocli_cg/internal/llm"
	"github.com/sanja/octocli_cg/internal/tools"
	"github.com/spf13/cobra"
)

func agentCmd() *cobra.Command {
	var force bool
	var maxSteps int

	cmd := &cobra.Command{
		Use:   "agent [prompt]",
		Short: "Run the JSON-based ReAct agent or start an interactive REPL",
		Long:  "Runs a minimal ReAct loop that can call read_file, write_file, and execute_shell before returning a final answer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			profile, err := cfg.ResolveProfile(profileName)
			if err != nil {
				return err
			}

			workspaceRoot, err := os.Getwd()
			if err != nil {
				return err
			}
			bundle, err := appcontext.Load(workspaceRoot)
			if err != nil {
				return err
			}

			runner := agent.Loop{
				Client: llm.NewOpenAICompatibleClient(profile),
				Tools: tools.Runtime{
					WorkspaceRoot: workspaceRoot,
					ForceShell:    force,
					In:            os.Stdin,
					Out:           os.Stdout,
					Err:           os.Stderr,
				},
				MaxIterations: maxSteps,
				Rules:         bundle.Rules,
			}

			if len(args) > 0 {
				prompt, err := resolvePrompt(bundle, strings.Join(args, " "))
				if err != nil {
					return err
				}
				answer, err := runner.Run(context.Background(), prompt)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), answer)
				return err
			}

			return runAgentREPL(cmd, runner, bundle)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "allow execute_shell without interactive confirmation")
	cmd.Flags().IntVar(&maxSteps, "max-steps", agent.DefaultMaxIterations, "maximum ReAct iterations before aborting")
	return cmd
}

func runAgentREPL(cmd *cobra.Command, runner agent.Loop, bundle appcontext.Bundle) error {
	reader := bufio.NewScanner(os.Stdin)
	fmt.Fprintln(cmd.OutOrStdout(), "octocli_cg agent REPL. Type 'exit' or 'quit' to stop.")
	for {
		fmt.Fprint(cmd.OutOrStdout(), "> ")
		if !reader.Scan() {
			if err := reader.Err(); err != nil {
				return err
			}
			return nil
		}
		prompt := strings.TrimSpace(reader.Text())
		if prompt == "" {
			continue
		}
		if prompt == "exit" || prompt == "quit" {
			return nil
		}
		resolvedPrompt, err := resolvePrompt(bundle, prompt)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			continue
		}

		answer, err := runner.Run(context.Background(), resolvedPrompt)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", err)
			continue
		}
		fmt.Fprintln(cmd.OutOrStdout(), answer)
	}
}

func resolvePrompt(bundle appcontext.Bundle, input string) (string, error) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return input, nil
	}

	parts := strings.Fields(input)
	name := strings.TrimPrefix(parts[0], "/")
	wf, ok := bundle.Workflows[name]
	if !ok {
		return "", fmt.Errorf("workflow /%s not found", name)
	}
	if len(parts) == 1 {
		return wf.Content, nil
	}
	return wf.Content + "\n\nUser arguments:\n" + strings.Join(parts[1:], " "), nil
}
