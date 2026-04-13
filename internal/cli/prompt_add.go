package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/meruem/prompt-builder-agent/pkg/models"
	"github.com/spf13/cobra"

	promptlib "github.com/meruem/prompt-builder-agent/internal/prompt"
)

var (
	addID          string
	addName        string
	addDescription string
	addEngine      string
	addTemplate    string
	addParams      []string
	addIcon        string
)

func init() {
	promptCmd.AddCommand(promptAddCmd)

	// Add flags
	promptAddCmd.Flags().StringVar(&addID, "id", "", "Prompt ID (required)")
	promptAddCmd.Flags().StringVar(&addName, "name", "", "Prompt name")
	promptAddCmd.Flags().StringVar(&addDescription, "description", "", "Prompt description")
	promptAddCmd.Flags().StringVar(&addEngine, "engine", "openai", "LLM engine (openai or gemini)")
	promptAddCmd.Flags().StringVar(&addTemplate, "template", "", "Prompt template")
	promptAddCmd.Flags().StringSliceVar(&addParams, "params", nil, "Supported param keys, e.g. tone,length,complexity")
	promptAddCmd.Flags().StringVar(&addIcon, "icon", "", "Icon emoji or name for this prompt")
}

var promptAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new prompt",
	Long:  "Add a new prompt with flags or interactive mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		// If flags provided, use them; otherwise use interactive mode
		if addID != "" || addTemplate != "" {
			return addPromptFromFlags()
		}
		return addPromptInteractive()
	},
}

func addPromptFromFlags() error {
	// Validate required fields
	if addID == "" {
		return fmt.Errorf("--id is required")
	}
	if addTemplate == "" {
		return fmt.Errorf("--template is required")
	}

	// Set defaults
	if addName == "" {
		addName = addID
	}
	if addEngine == "" {
		addEngine = "openai"
	}

	// Extract variables
	variables := promptlib.ExtractVariables(addTemplate)

	prompt := &models.Prompt{
		ID:          addID,
		Name:        addName,
		Description: addDescription,
		Engine:      addEngine,
		Template:    addTemplate,
		Variables:   variables,
		Params:      addParams,
		Icon:        addIcon,
	}

	if err := app.GlobalContext.PromptStore.Create(prompt); err != nil {
		return fmt.Errorf("cannot create prompt: %w", err)
	}

	fmt.Printf("✓ Prompt '%s' created\n", addID)
	if len(variables) > 0 {
		fmt.Printf("  Variables: %v\n", variables)
	}
	if len(addParams) > 0 {
		fmt.Printf("  Params: %v\n", addParams)
	}
	return nil
}

func addPromptInteractive() error {
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
}
