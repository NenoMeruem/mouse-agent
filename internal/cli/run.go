package cli

import (
	"github.com/spf13/cobra"
)

var (
	dryRun      bool
	rawOutput   bool
	noSelection bool
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview prompt without running")
	runCmd.Flags().BoolVar(&rawOutput, "raw", false, "output raw prompt without formatting")
	runCmd.Flags().BoolVar(&noSelection, "no-select", false, "skip selection, use defaults only")
}

var runCmd = &cobra.Command{
	Use:   "run <prompt-id>",
	Short: "Run prompt with current selection",
	Long: `Run a prompt with clipboard selection.

Examples:
  prompt-agent run explain_code              # Get selection from clipboard
  prompt-agent run code_review --raw         # Raw output without formatting
  prompt-agent run explain_code --dry-run    # Preview only, no execution`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptCommand,
}
