package cli

import (
	"github.com/spf13/cobra"
)

var (
	dryRun        bool
	rawOutput     bool
	noSelection   bool
	editPrompt    bool
	runTone       string
	runLength     string
	runComplexity string
	runEngine     string
	runSessionID  string // conversation session ID (set by Tauri, groups turns in history)
)

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview prompt without running")
	runCmd.Flags().BoolVar(&rawOutput, "raw", false, "output raw prompt without formatting")
	runCmd.Flags().BoolVar(&noSelection, "no-select", false, "skip selection, use defaults only")
	runCmd.Flags().BoolVar(&editPrompt, "edit", false, "Open prompt in editor before sending")
	runCmd.Flags().StringVar(&runTone, "tone", "", "Tone: professional, casual, concise")
	runCmd.Flags().StringVar(&runLength, "length", "", "Length: short, medium, long")
	runCmd.Flags().StringVar(&runComplexity, "complexity", "", "Complexity: simple, normal, technical")
	runCmd.Flags().StringVar(&runEngine, "engine", "", "Override engine (gemini, openai, claude, ...)")
	runCmd.Flags().StringVar(&runSessionID, "session-id", "", "Conversation session ID (Tauri)")
}

var runCmd = &cobra.Command{
	Use:   "run <prompt-id>",
	Short: "Run prompt with current selection",
	Long: `Run a prompt with clipboard selection.

Examples:
  promptly run explain_code              # Get selection from clipboard
  promptly run code_review --raw         # Raw output without formatting
  promptly run explain_code --dry-run    # Preview only, no execution`,
	Args: cobra.ExactArgs(1),
	RunE: runPromptCommand,
}
