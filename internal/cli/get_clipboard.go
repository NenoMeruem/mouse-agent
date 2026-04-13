package cli

import (
	"fmt"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getClipboardCmd)
}

// getClipboardCmd reads clipboard content via the OS selection provider
// (pbpaste on macOS, xclip/xdotool on Linux, PowerShell on Windows)
// and prints it to stdout. Exits silently if clipboard is empty or unavailable.
var getClipboardCmd = &cobra.Command{
	Use:    "get-clipboard",
	Short:  "Print clipboard contents to stdout",
	Hidden: true, // internal use by Tauri overlay
	RunE: func(cmd *cobra.Command, args []string) error {
		text, err := app.GlobalContext.SelectionManager.GetSelection()
		if err != nil {
			// Empty clipboard is not an error for GUI use — exit cleanly
			return nil
		}
		fmt.Print(text)
		return nil
	},
}
