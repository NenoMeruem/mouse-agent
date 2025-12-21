package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("prompt-agent version", version)
	},
}
