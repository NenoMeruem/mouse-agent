package cli

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(promptCmd)
}

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Manage prompts",
}
