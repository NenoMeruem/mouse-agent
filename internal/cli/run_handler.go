package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

func runPromptCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Get prompt definition
	promptDef, err := app.GlobalContext.PromptStore.Get(id)
	if err != nil {
		return fmt.Errorf("cannot get prompt: %w", err)
	}

	// Get selection from clipboard
	selectionText, err := app.GlobalContext.Selection.Get()
	if err != nil {
		return fmt.Errorf("cannot get selection: %w", err)
	}

	// Build data map for template
	data := map[string]string{
		"selection": selectionText,
	}

	// Collect missing variables from environment or user input
	reader := bufio.NewReader(os.Stdin)
	for _, varName := range promptDef.Variables {
		if _, ok := data[varName]; !ok {
			// Try environment variable first
			envKey := "PROMPT_" + strings.ToUpper(varName)
			if envValue := os.Getenv(envKey); envValue != "" {
				data[varName] = envValue
			} else {
				// Ask user interactively
				fmt.Printf("Enter %s: ", varName)
				value, _ := reader.ReadString('\n')
				value = strings.TrimSpace(value)
				if value == "" {
					return fmt.Errorf("missing required variable: %s", varName)
				}
				data[varName] = value
			}
		}
	}

	// Build final prompt
	finalPrompt, err := app.GlobalContext.Builder.Build(
		promptDef.Template,
		data,
	)
	if err != nil {
		return fmt.Errorf("cannot build prompt: %w", err)
	}

	// Display preview
	if rawOutput {
		fmt.Println(finalPrompt)
	} else {
		fmt.Println("=== PROMPT PREVIEW ===")
		fmt.Println(finalPrompt)
		fmt.Println(strings.Repeat("=", 22))
	}

	return nil
}
