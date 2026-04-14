// Package cli implements the command-line interface for the prompt agent.
package cli

import (
	"fmt"
	"os"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/meruem/prompt-builder-agent/internal/config"
	"github.com/meruem/prompt-builder-agent/internal/logger"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prompt-agent",
	Short: "Prompt Agent - build & run AI prompts fast",
	Long: `A CLI tool for quickly building and running AI prompts with support for
multiple LLM backends (OpenAI, Google Gemini) and reusable templates.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Init file logger — non-fatal if directory is unwritable.
		if err := logger.Init(logger.DefaultLogDir()); err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot init log file: %v\n", err)
		}

		// Initialize application context once at startup.
		if err := app.InitializeContext(); err != nil {
			logger.Error("context initialization failed", "error", err)
			return err
		}
		if err := config.Load(); err != nil {
			logger.Error("config load failed", "error", err)
			return err
		}
		return nil
	},
}

// Execute runs the root command and handles errors.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("command failed", "error", err)
		fmt.Println(err)
		os.Exit(1)
	}
}
