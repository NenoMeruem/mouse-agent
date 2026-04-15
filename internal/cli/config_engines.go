package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var (
	setEngineAPIKey string
	setEngineModel  string
	setEngineName   string
)

func init() {
	configCmd.AddCommand(getEnginesCmd)
	configCmd.AddCommand(setEngineCmd)
	configCmd.AddCommand(deleteEngineCmd)

	setEngineCmd.Flags().StringVar(&setEngineAPIKey, "api-key", "", "API key (plain text or env:VAR_NAME)")
	setEngineCmd.Flags().StringVar(&setEngineModel, "model", "", "Model name")
	setEngineCmd.Flags().StringVar(&setEngineName, "name", "", "Display name for this engine")
}

// EngineConfigJSON is the JSON representation returned by get-engines.
type EngineConfigJSON struct {
	Name   string `json:"name"`
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// getEnginesCmd reads and returns all engine configs as JSON (raw, not resolved).
var getEnginesCmd = &cobra.Command{
	Use:   "get-engines",
	Short: "Return engine configs as JSON (raw api_key values, not resolved)",
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readRawConfig()
		if err != nil {
			// Return empty object if no config yet
			fmt.Println("{}")
			return nil
		}

		engines := map[string]EngineConfigJSON{}
		if engs, ok := raw["engines"].(map[string]interface{}); ok {
			for name, v := range engs {
				if m, ok := v.(map[string]interface{}); ok {
					ec := EngineConfigJSON{}
					if n, ok := m["name"].(string); ok {
						ec.Name = n
					}
					if k, ok := m["api_key"].(string); ok {
						ec.APIKey = k
					}
					if mod, ok := m["model"].(string); ok {
						ec.Model = mod
					}
					engines[name] = ec
				}
			}
		}

		b, _ := json.Marshal(engines)
		fmt.Println(string(b))
		return nil
	},
}

// setEngineCmd writes api_key and/or model for one engine into config.yaml.
var setEngineCmd = &cobra.Command{
	Use:   "set-engine <engine>",
	Short: "Set api_key and/or model for an engine in config.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := args[0]

		raw, err := readRawConfig()
		if err != nil {
			raw = map[string]interface{}{}
		}

		// Ensure engines map exists
		engs, _ := raw["engines"].(map[string]interface{})
		if engs == nil {
			engs = map[string]interface{}{}
		}

		// Get or create engine entry
		entry, _ := engs[engine].(map[string]interface{})
		if entry == nil {
			entry = map[string]interface{}{}
		}

		if cmd.Flags().Changed("name") {
			entry["name"] = setEngineName
		}
		if cmd.Flags().Changed("api-key") {
			entry["api_key"] = setEngineAPIKey
		}
		if cmd.Flags().Changed("model") {
			entry["model"] = setEngineModel
		}

		// Always set a default timeout if missing
		if _, ok := entry["timeout"]; !ok {
			entry["timeout"] = "120s"
		}

		engs[engine] = entry
		raw["engines"] = engs

		if err := writeRawConfig(raw); err != nil {
			return fmt.Errorf("cannot write config: %w", err)
		}

		fmt.Printf("✓ Engine '%s' updated\n", engine)
		return nil
	},
}

// deleteEngineCmd removes an engine entry from config.yaml.
var deleteEngineCmd = &cobra.Command{
	Use:   "delete-engine <engine>",
	Short: "Remove an engine from config.yaml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := args[0]
		raw, err := readRawConfig()
		if err != nil {
			return fmt.Errorf("cannot read config: %w", err)
		}
		engs, _ := raw["engines"].(map[string]interface{})
		if engs != nil {
			delete(engs, engine)
		}
		raw["engines"] = engs
		if err := writeRawConfig(raw); err != nil {
			return fmt.Errorf("cannot write config: %w", err)
		}
		fmt.Printf("✓ Engine '%s' deleted\n", engine)
		return nil
	},
}

// readRawConfig reads ~/.prompt-agent/config.yaml as a plain map.
func readRawConfig() (map[string]interface{}, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

// writeRawConfig writes a plain map back to ~/.prompt-agent/config.yaml.
func writeRawConfig(raw map[string]interface{}) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prompt-agent", "config.yaml")
}
