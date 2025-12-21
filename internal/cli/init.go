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
			return nil
		}

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
		return nil
	},
}
