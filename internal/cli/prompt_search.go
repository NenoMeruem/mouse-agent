package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptSearchCmd)
}

var promptSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search prompts by name or template content",
	Args:  cobra.ExactArgs(1),
	RunE:  promptSearchCommand,
}

func promptSearchCommand(cmd *cobra.Command, args []string) error {
	query := args[0]

	prompts, err := app.GlobalContext.PromptStore.Search(query)
	if err != nil {
		return fmt.Errorf("cannot search prompts: %w", err)
	}

	if len(prompts) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No prompts found matching %q.\n", query)
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
}
