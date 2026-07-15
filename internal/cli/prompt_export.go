package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/meruem/promptly/internal/app"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptExportCmd)
}

var promptExportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export prompt recipes and engine configurations to a CSV file",
	Long:  "Export all prompts from SQLite database and engine configurations from config.yaml to a CSV file. If [file] is not provided or is '-', output is written to stdout.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var out io.Writer = cmd.OutOrStdout()
		if len(args) > 0 && args[0] != "-" {
			file, err := os.Create(args[0])
			if err != nil {
				return fmt.Errorf("cannot create file %s: %w", args[0], err)
			}
			defer file.Close()
			out = file
		}

		// Fetch prompts
		prompts, err := app.GlobalContext.PromptStore.List()
		if err != nil {
			return fmt.Errorf("cannot list prompts: %w", err)
		}

		// Fetch engines config
		rawConfig, err := readRawConfig()
		var engines map[string]interface{}
		if err == nil {
			if engs, ok := rawConfig["engines"].(map[string]interface{}); ok {
				engines = engs
			}
		}

		writer := csv.NewWriter(out)
		defer writer.Flush()

		// Write header
		header := []string{
			"type", "id", "name", "description", "engine", "template",
			"variables", "params", "icon", "sort_order", "api_key", "model",
		}
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("cannot write CSV header: %w", err)
		}

		// Write prompts
		for _, p := range prompts {
			varsJSON, _ := json.Marshal(p.Variables)
			paramsJSON, _ := json.Marshal(p.Params)
			row := []string{
				"prompt",
				p.ID,
				p.Name,
				p.Description,
				p.Engine,
				p.Template,
				string(varsJSON),
				string(paramsJSON),
				p.Icon,
				strconv.Itoa(p.SortOrder),
				"", // api_key
				"", // model
			}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("cannot write prompt row: %w", err)
			}
		}

		// Write engines
		if engines != nil {
			for engineID, v := range engines {
				if m, ok := v.(map[string]interface{}); ok {
					name := ""
					if n, ok := m["name"].(string); ok {
						name = n
					}
					apiKey := ""
					if k, ok := m["api_key"].(string); ok {
						apiKey = k
					}
					model := ""
					if mod, ok := m["model"].(string); ok {
						model = mod
					}

					row := []string{
						"engine",
						engineID,
						name,
						"", // description
						"", // engine
						"", // template
						"", // variables
						"", // params
						"", // icon
						"", // sort_order
						apiKey,
						model,
					}
					if err := writer.Write(row); err != nil {
						return fmt.Errorf("cannot write engine row: %w", err)
					}
				}
			}
		}

		return nil
	},
}
