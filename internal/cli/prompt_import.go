package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/meruem/promptly/internal/app"
	promptlib "github.com/meruem/promptly/internal/prompt"
	"github.com/meruem/promptly/pkg/models"
	"github.com/spf13/cobra"
)

func init() {
	promptCmd.AddCommand(promptImportCmd)
}

var promptImportCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import prompt recipes and engine configurations from a CSV file",
	Long:  "Import prompts into the SQLite database and engine configurations into config.yaml from a CSV file. If [file] is not provided or is '-', input is read from stdin.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var in io.Reader = cmd.InOrStdin()
		if len(args) > 0 && args[0] != "-" {
			file, err := os.Open(args[0])
			if err != nil {
				return fmt.Errorf("cannot open file %s: %w", args[0], err)
			}
			defer file.Close()
			in = file
		}

		reader := csv.NewReader(in)
		// Read header
		header, err := reader.Read()
		if err != nil {
			return fmt.Errorf("cannot read CSV header: %w", err)
		}

		// Map header column names to indexes
		colIndex := make(map[string]int)
		for idx, name := range header {
			colIndex[name] = idx
		}

		// Helper to safely get value from row
		getVal := func(row []string, colName string) string {
			idx, ok := colIndex[colName]
			if !ok || idx >= len(row) {
				return ""
			}
			return row[idx]
		}

		// Track imports
		promptsImported := 0
		enginesImported := 0

		// Read rows
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("error reading CSV row: %w", err)
			}

			rowType := getVal(row, "type")
			if rowType == "prompt" {
				id := getVal(row, "id")
				if id == "" {
					continue
				}

				name := getVal(row, "name")
				if name == "" {
					name = id
				}
				description := getVal(row, "description")
				engine := getVal(row, "engine")
				if engine == "" {
					engine = "openai"
				}
				template := getVal(row, "template")

				// Parse variables
				var variables []string
				varsStr := getVal(row, "variables")
				if varsStr != "" {
					_ = json.Unmarshal([]byte(varsStr), &variables)
				}
				if len(variables) == 0 && template != "" {
					variables = promptlib.ExtractVariables(template)
				}

				// Parse params
				var params []string
				paramsStr := getVal(row, "params")
				if paramsStr != "" {
					_ = json.Unmarshal([]byte(paramsStr), &params)
				}

				icon := getVal(row, "icon")

				sortOrder := 0
				sortOrderStr := getVal(row, "sort_order")
				if sortOrderStr != "" {
					if so, err := strconv.Atoi(sortOrderStr); err == nil {
						sortOrder = so
					}
				}

				p := &models.Prompt{
					ID:          id,
					Name:        name,
					Description: description,
					Engine:      engine,
					Template:    template,
					Variables:   variables,
					Params:      params,
					Icon:        icon,
					SortOrder:   sortOrder,
				}

				// Save prompt
				existing, err := app.GlobalContext.PromptStore.Get(p.ID)
				if err == nil && existing != nil {
					p.CreatedAt = existing.CreatedAt
					if p.SortOrder == 0 && existing.SortOrder != 0 {
						p.SortOrder = existing.SortOrder
					}
					if err := app.GlobalContext.PromptStore.Update(p); err != nil {
						return fmt.Errorf("cannot update prompt '%s': %w", p.ID, err)
					}
				} else {
					if err := app.GlobalContext.PromptStore.Create(p); err != nil {
						return fmt.Errorf("cannot create prompt '%s': %w", p.ID, err)
					}
				}
				promptsImported++

			} else if rowType == "engine" {
				engineID := getVal(row, "id")
				if engineID == "" {
					continue
				}

				name := getVal(row, "name")
				apiKey := getVal(row, "api_key")
				model := getVal(row, "model")

				// Update engine config
				rawConfig, err := readRawConfig()
				if err != nil {
					rawConfig = map[string]interface{}{}
				}

				engs, _ := rawConfig["engines"].(map[string]interface{})
				if engs == nil {
					engs = map[string]interface{}{}
				}

				entry, _ := engs[engineID].(map[string]interface{})
				if entry == nil {
					entry = map[string]interface{}{}
				}

				if name != "" {
					entry["name"] = name
				}
				if apiKey != "" {
					entry["api_key"] = apiKey
				}
				if model != "" {
					entry["model"] = model
				}
				if _, ok := entry["timeout"]; !ok {
					entry["timeout"] = "120s"
				}

				engs[engineID] = entry
				rawConfig["engines"] = engs

				if err := writeRawConfig(rawConfig); err != nil {
					return fmt.Errorf("cannot write config for engine '%s': %w", engineID, err)
				}
				enginesImported++
			}
		}

		fmt.Printf("✓ Successfully imported %d prompts and %d engine configurations\n", promptsImported, enginesImported)
		return nil
	},
}
