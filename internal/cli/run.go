package cli

import (
	"github.com/spf13/cobra"
)

var (
	dryRun    bool
	rawOutput bool
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview prompt without running")
	runCmd.Flags().BoolVar(&rawOutput, "raw", false, "output raw prompt without formatting")
}

var runCmd = &cobra.Command{
	Use:   "run <prompt-id>",
	Short: "Run prompt with current selection",
	Args:  cobra.ExactArgs(1),
	RunE:  runPromptCommand,
}
