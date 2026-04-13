package cli

import (
	"fmt"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptReorderCmd)
}

var promptReorderCmd = &cobra.Command{
	Use:   "reorder <id1> <id2> ...",
	Short: "Set display order by listing prompt IDs in desired order",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := app.GlobalContext.PromptStore
		for i, id := range args {
			p, err := store.Get(id)
			if err != nil {
				return fmt.Errorf("prompt not found: %s", id)
			}
			p.SortOrder = i
			if err := store.Update(p); err != nil {
				return fmt.Errorf("cannot update sort order for %s: %w", id, err)
			}
		}
		fmt.Printf("✓ Reordered %d prompts\n", len(args))
		return nil
	},
}
