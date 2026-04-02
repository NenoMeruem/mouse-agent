package cli

import (
	"fmt"
	"os"
	"path/filepath"

	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/sl/prompt-builder-agent/internal/storage"
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

		// Seed default recipes into SQLite store
		dbPath := storage.GetDefaultDBPath()
		store, err := storage.NewSQLiteStore(dbPath)
		if err != nil {
			return fmt.Errorf("cannot open store: %w", err)
		}

		existing, err := store.List()
		if err != nil {
			return fmt.Errorf("cannot list prompts: %w", err)
		}

		if len(existing) == 0 {
			for _, recipe := range promptlib.DefaultRecipes() {
				r := recipe // copy
				if err := store.Create(&r); err != nil {
					// skip if already exists (e.g. concurrent init)
					fmt.Printf("warning: could not seed recipe %s: %v\n", recipe.ID, err)
				}
			}
			fmt.Printf("Seeded %d default recipes.\n", len(promptlib.DefaultRecipes()))
		} else {
			fmt.Printf("Store already has %d prompts, skipping seed.\n", len(existing))
		}

		return nil
	},
}
