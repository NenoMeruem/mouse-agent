package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptListCmd)
}

var promptListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all prompts",
	RunE: func(cmd *cobra.Command, args []string) error {
		prompts, err := app.GlobalContext.PromptStore.List()
		if err != nil {
			return fmt.Errorf("cannot list prompts: %w", err)
		}

		if len(prompts) == 0 {
			fmt.Println("No prompts found. Create one with 'prompt-agent prompt add'")
			return nil
		}

		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tENGINE\tNAME")
		fmt.Fprintln(w, "-\t-\t-")

		for _, p := range prompts {
			fmt.Fprintf(w, "%s\t%s\t%s\n", p.ID, p.Engine, p.Name)
		}

		w.Flush()
		return nil
	},
}
