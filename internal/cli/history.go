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

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "Number of records to show")
	historyCmd.Flags().StringVar(&historyPromptFilter, "prompt", "", "Filter by prompt ID")
}
