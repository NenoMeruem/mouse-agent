package cli

import (
	"fmt"
	"os"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prompt-agent",
	Short: "Prompt Agent - build & run AI prompts fast",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize app context (storage, config)
		if err := app.InitializeContext(); err != nil {
			return err
		}
		return config.Load()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func loadConfig(cmd *cobra.Command, args []string) error {
	return config.Load()
}
