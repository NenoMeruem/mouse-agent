package cli

import (
	"fmt"
	"strings"

	"github.com/sl/prompt-builder-agent/internal/app"
	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	upName        string
	upDescription string
	upEngine      string
	upTemplate    string
	upParams      []string
	upIcon        string
	upSortOrder   int
)

func init() {
	promptCmd.AddCommand(promptUpdateCmd)

	promptUpdateCmd.Flags().StringVar(&upName, "name", "", "Prompt name")
	promptUpdateCmd.Flags().StringVar(&upDescription, "description", "", "Prompt description")
	promptUpdateCmd.Flags().StringVar(&upEngine, "engine", "", "LLM engine (openai or gemini)")
	promptUpdateCmd.Flags().StringVar(&upTemplate, "template", "", "Prompt template")
	promptUpdateCmd.Flags().StringSliceVar(&upParams, "params", nil, "Supported param keys, e.g. tone,length,complexity")
	promptUpdateCmd.Flags().StringVar(&upIcon, "icon", "", "Icon emoji or name")
	promptUpdateCmd.Flags().IntVar(&upSortOrder, "sort-order", 0, "Display sort order (lower = higher in list)")
}

var promptUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an existing prompt via flags",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		existing, err := app.GlobalContext.PromptStore.Get(id)
		if err != nil {
			return fmt.Errorf("prompt not found: %w", err)
		}

		// Apply only the flags that were explicitly set
		if cmd.Flags().Changed("name") {
			existing.Name = upName
		}
		if cmd.Flags().Changed("description") {
			existing.Description = upDescription
		}
		if cmd.Flags().Changed("engine") {
			existing.Engine = upEngine
		}
		if cmd.Flags().Changed("template") {
			existing.Template = upTemplate
			existing.Variables = promptlib.ExtractVariables(upTemplate)
		}
		if cmd.Flags().Changed("params") {
			// Filter out empty strings from comma-split
			var cleaned []string
			for _, p := range upParams {
				if t := strings.TrimSpace(p); t != "" {
					cleaned = append(cleaned, t)
				}
			}
			existing.Params = cleaned
		}
		if cmd.Flags().Changed("icon") {
			existing.Icon = upIcon
		}
		if cmd.Flags().Changed("sort-order") {
			existing.SortOrder = upSortOrder
		}

		if err := app.GlobalContext.PromptStore.Update(existing); err != nil {
			return fmt.Errorf("cannot update prompt: %w", err)
		}

		fmt.Printf("✓ Prompt '%s' updated\n", id)
		return nil
	},
}
