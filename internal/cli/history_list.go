package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/meruem/prompt-builder-agent/pkg/models"
	"github.com/spf13/cobra"
)

func historyListCommand(cmd *cobra.Command, args []string) error {
	records, err := app.GlobalContext.HistoryStore.List(historyLimit, historyPromptFilter)
	if err != nil {
		return fmt.Errorf("cannot list history: %w", err)
	}

	if historyOutputJSON {
		b, _ := json.Marshal(records)
		fmt.Println(string(b))
		return nil
	}

	return renderHistoryTable(cmd.OutOrStdout(), records)
}

// renderHistoryTable writes records as a tab-separated table to w.
func renderHistoryTable(w io.Writer, records []models.RunRecord) error {
	if len(records) == 0 {
		fmt.Fprintln(w, "No history found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tPROMPT\tENGINE\tDURATION\tCREATED")
	fmt.Fprintln(tw, "-\t-\t-\t-\t-")

	for _, r := range records {
		id := r.ID
		if len(id) > 8 {
			id = id[:8]
		}
		duration := fmt.Sprintf("%.2fs", float64(r.DurationMs)/1000.0)
		created := r.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", id, r.PromptID, r.Engine, duration, created)
	}

	return tw.Flush()
}
