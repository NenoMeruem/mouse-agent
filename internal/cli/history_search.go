package cli

import (
	"fmt"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

var historySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search run history by keyword",
	Args:  cobra.ExactArgs(1),
	RunE:  historySearchCommand,
}

func init() {
	historyCmd.AddCommand(historySearchCmd)
}

func historySearchCommand(cmd *cobra.Command, args []string) error {
	query := args[0]

	records, err := app.GlobalContext.HistoryStore.Search(query)
	if err != nil {
		return fmt.Errorf("cannot search history: %w", err)
	}

	return renderHistoryTable(cmd.OutOrStdout(), records)
}
