package octocli

import (
	"fmt"

	"github.com/sanja/octocli_cg/internal/config"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage octocli_cg configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create a sample configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.EnsureSample(cfgPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created sample config: %s\n", path)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show loaded configuration summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}
			active, err := cfg.ResolveProfile(profileName)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "default_profile: %s\nactive_profile: %s\nbase_url: %s\nmodel: %s\n", cfg.DefaultProfile, active.Name, active.BaseURL, active.Model)
			return nil
		},
	})

	return cmd
}
