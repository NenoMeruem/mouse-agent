package cli

import (
	"fmt"
	"strings"

	"github.com/meruem/promptly/internal/app"
	"github.com/spf13/cobra"
)

var historyShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show full details of a history record",
	Args:  cobra.ExactArgs(1),
	RunE:  historyShowCommand,
}

func init() {
	historyCmd.AddCommand(historyShowCmd)
}

func historyShowCommand(cmd *cobra.Command, args []string) error {
	prefix := args[0]

	records, err := app.GlobalContext.HistoryStore.List(0, "")
	if err != nil {
		return fmt.Errorf("cannot load history: %w", err)
	}

	for _, r := range records {
		shortID := r.ID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		if r.ID == prefix || shortID == prefix || strings.HasPrefix(r.ID, prefix) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "ID:         %s\n", r.ID)
			fmt.Fprintf(out, "Prompt ID:  %s\n", r.PromptID)
			fmt.Fprintf(out, "Engine:     %s\n", r.Engine)
			fmt.Fprintf(out, "Duration:   %.2fs\n", float64(r.DurationMs)/1000.0)
			fmt.Fprintf(out, "Created At: %s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
			if r.Error != "" {
				fmt.Fprintf(out, "Error:      %s\n", r.Error)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "--- Input Text ---")
			fmt.Fprintln(out, r.InputText)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "--- Final Prompt ---")
			fmt.Fprintln(out, r.FinalPrompt)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "--- Response ---")
			fmt.Fprintln(out, r.Response)
			return nil
		}
	}

	return fmt.Errorf("no history record found for id: %s", prefix)
}
