package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize prompt-agent config",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get home directory: %w", err)
		}

		dir := filepath.Join(home, ".prompt-agent")

		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		configPath := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			fmt.Println("Config already exists:", configPath)
		} else {
			defaultConfig := `engines:
  openai:
    api_key: ""
    model: "gpt-4-mini"
ui:
  output: "stdout"
`
			if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
				return fmt.Errorf("cannot write config file: %w", err)
			}
			fmt.Println("Config created at:", configPath)
		}
		// include a minimal example entry so users know the format
		examplePromptsPath := filepath.Join(dir, "prompts.json")
		examplePrompts := `[
  {
	"id": "example",
	"name": "Example Prompt",
	"description": "This is a sample prompt. Edit or delete it.",
	"engine": "openai",
	"template": "Write a short description of {{.topic}}.",
	"variables": ["topic"],
	"created_at": "2026-01-01T00:00:00Z",
	"updated_at": "2026-01-01T00:00:00Z"
  }
]
`
		if err := os.WriteFile(examplePromptsPath, []byte(examplePrompts), 0644); err != nil {
			return fmt.Errorf("cannot write example prompts file: %w", err)
		}
		fmt.Println("Example prompts created at:", examplePromptsPath)
		return nil
	},
}
