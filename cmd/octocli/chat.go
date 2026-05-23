package octocli

import (
	"context"
	"fmt"
	"strings"

	"github.com/sanja/octocli_cg/internal/llm"
	"github.com/spf13/cobra"
)

func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat [prompt]",
		Short: "Send a prompt to the configured LLM and stream the response",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			profile, err := cfg.ResolveProfile(profileName)
			if err != nil {
				return err
			}

			client := llm.NewOpenAICompatibleClient(profile)
			request := llm.ChatRequest{
				Messages: []llm.Message{{Role: "user", Content: strings.Join(args, " ")}},
			}

			return client.StreamChat(context.Background(), request, func(delta string) error {
				_, err := fmt.Fprint(cmd.OutOrStdout(), delta)
				return err
			})
		},
	}
}
