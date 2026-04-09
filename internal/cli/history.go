package cli

import "github.com/spf13/cobra"

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View and manage run history",
	RunE:  historyListCommand, // default to list when no subcommand
}

// flags for the default list action
var historyLimit int
var historyPromptFilter string
var historyOutputJSON bool

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().IntVar(&historyLimit, "limit", 50, "Number of records to show")
	historyCmd.Flags().StringVar(&historyPromptFilter, "prompt", "", "Filter by prompt ID")
	historyCmd.Flags().BoolVar(&historyOutputJSON, "output-json", false, "Output as JSON (for Tauri)")
}
