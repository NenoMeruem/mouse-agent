package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/pkg/models"
	"github.com/spf13/cobra"

	promptlib "github.com/sl/prompt-builder-agent/internal/prompt"
)

func init() {
	promptCmd.AddCommand(promptAddCmd)
}

var promptAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new prompt",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// ID
		fmt.Print("ID (e.g., explain_code): ")
		id, _ := reader.ReadString('\n')
		id = strings.TrimSpace(id)
		if id == "" {
			return fmt.Errorf("ID cannot be empty")
		}

		// Name
		fmt.Print("Name: ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)

		// Description
		fmt.Print("Description: ")
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		// Engine
		fmt.Print("Engine [openai]: ")
		engine, _ := reader.ReadString('\n')
		engine = strings.TrimSpace(engine)
		if engine == "" {
			engine = "openai"
		}

		// Template
		fmt.Println("Template (type 'EOF' on new line to finish):")
		var templateLines []string
		for {
			line, _ := reader.ReadString('\n')
			line = strings.TrimSuffix(line, "\n")
			if line == "EOF" {
				break
			}
			templateLines = append(templateLines, line)
		}
		template := strings.Join(templateLines, "\n")

		// Extract variables
		variables := promptlib.ExtractVariables(template)

		prompt := &models.Prompt{
			ID:          id,
			Name:        name,
			Description: description,
			Engine:      engine,
			Template:    template,
			Variables:   variables,
		}

		if err := app.GlobalContext.PromptStore.Create(prompt); err != nil {
			return fmt.Errorf("cannot create prompt: %w", err)
		}

		fmt.Printf("✓ Prompt '%s' created\n", id)
		if len(variables) > 0 {
			fmt.Printf("  Variables: %v\n", variables)
		}
		return nil
	},
}
