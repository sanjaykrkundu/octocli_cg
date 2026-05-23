package octocli

import (
	"fmt"
	"os"

	"github.com/sanja/octocli_cg/internal/config"
	"github.com/spf13/cobra"
)

var cfgPath string
var profileName string

var rootCmd = &cobra.Command{
	Use:   "octocli_cg",
	Short: "A fast CLI agent with configurable OpenAI-compatible LLM connectivity",
	Long: `octocli_cg is a CLI agent foundation focused on speed, asynchronous execution,
and unified local state. Step 1 provides the Cobra command shell, configuration
loading, and an OpenAI-compatible streaming LLM harness.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to config.yaml (defaults to ~/.octocli_cg/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&profileName, "profile", "", "LLM profile name to use")
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(chatCmd())
}

func loadConfig() (*config.Config, error) {
	return config.Load(cfgPath)
}
