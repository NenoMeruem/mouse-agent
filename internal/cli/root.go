// Package cli implements the command-line interface for the prompt agent.
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
	Long: `A CLI tool for quickly building and running AI prompts with support for
multiple LLM backends (OpenAI, Google Gemini) and reusable templates.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize application context once at startup
		if err := app.InitializeContext(); err != nil {
			return err
		}
		return config.Load()
	},
}

// Execute runs the root command and handles errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
