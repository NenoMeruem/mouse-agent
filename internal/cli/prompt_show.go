package cli

import (
	"fmt"
	"strings"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptShowCmd)
}

var promptShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a prompt details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		prompt, err := app.GlobalContext.PromptStore.Get(id)
		if err != nil {
			return fmt.Errorf("cannot get prompt: %w", err)
		}

		fmt.Printf("ID: %s\n", prompt.ID)
		if prompt.Icon != "" {
			fmt.Printf("Icon: %s\n", prompt.Icon)
		}
		fmt.Printf("Name: %s\n", prompt.Name)
		fmt.Printf("Engine: %s\n", prompt.Engine)
		if prompt.Description != "" {
			fmt.Printf("Description: %s\n", prompt.Description)
		}
		if len(prompt.Variables) > 0 {
			fmt.Printf("Variables: %s\n", strings.Join(prompt.Variables, ", "))
		}
		if len(prompt.Params) > 0 {
			fmt.Printf("Params: %s\n", strings.Join(prompt.Params, ", "))
		}
		fmt.Println("\nTemplate:")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println(prompt.Template)
		fmt.Println(strings.Repeat("-", 40))

		return nil
	},
}
