package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptDeleteCmd)
	promptDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

var promptDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a prompt",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		skipConfirm, _ := cmd.Flags().GetBool("yes")
		if !skipConfirm {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Are you sure you want to delete prompt '%s'? (y/N): ", id)
			response, _ := reader.ReadString('\n')
			response = strings.ToLower(strings.TrimSpace(response))

			if response != "y" && response != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		if err := app.GlobalContext.PromptStore.Delete(id); err != nil {
			return fmt.Errorf("cannot delete prompt: %w", err)
		}

		fmt.Printf("✓ Prompt '%s' deleted\n", id)
		return nil
	},
}
