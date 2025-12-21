package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sl/prompt-builder-agent/internal/app"
	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
	"github.com/spf13/cobra"
)

func runPromptCommand(cmd *cobra.Command, args []string) error {
	id := args[0]

	// Get prompt definition
	promptDef, err := app.GlobalContext.PromptStore.Get(id)
	if err != nil {
		return fmt.Errorf("❌ prompt not found: %s", id)
	}

	// Build data map for template
	data := map[string]string{}

	// Get selection from clipboard (unless --no-select is set)
	if !noSelection {
		selectionText, err := app.GlobalContext.SelectionManager.GetSelection()
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}
		data["selection"] = selectionText
	}

	// Collect missing variables from environment or user input
	reader := bufio.NewReader(os.Stdin)
	for _, varName := range promptDef.Variables {
		if _, ok := data[varName]; !ok {
			// Try environment variable first
			envKey := "PROMPT_" + strings.ToUpper(varName)
			if envValue := os.Getenv(envKey); envValue != "" {
				data[varName] = envValue
			} else if noSelection && varName == "selection" {
				// Skip if --no-select and variable is selection
				continue
			} else {
				// Ask user interactively
				fmt.Printf("Enter %s: ", varName)
				value, _ := reader.ReadString('\n')
				value = strings.TrimSpace(value)
				if value == "" {
					return fmt.Errorf("❌ missing required variable: %s", varName)
				}
				data[varName] = value
			}
		}
	}

	// Validate all required variables are present
	if err := promptlib.ValidateVariables(promptDef.Variables, data); err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	// Build final prompt
	finalPrompt, err := app.GlobalContext.Builder.Build(
		promptDef.Template,
		data,
	)
	if err != nil {
		return fmt.Errorf("❌ cannot build prompt: %w", err)
	}

	// Render final prompt
	title := fmt.Sprintf("PROMPT: %s", promptDef.Name)
	if err := app.GlobalContext.Renderer.Render(title, finalPrompt); err != nil {
		return fmt.Errorf("❌ cannot render output: %w", err)
	}

	// Show metadata if in formatted mode
	if !rawOutput {
		fmt.Printf("\n📋 Prompt ID: %s | Engine: %s | Variables: %v\n",
			promptDef.ID, promptDef.Engine, promptDef.Variables)
	}

	return nil
}
