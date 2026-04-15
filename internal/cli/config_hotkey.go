package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const defaultHotkey = "Alt+Space"

func init() {
	configCmd.AddCommand(getHotkeyCmd)
	configCmd.AddCommand(setHotkeyCmd)
}

var getHotkeyCmd = &cobra.Command{
	Use:   "get-hotkey",
	Short: "Print the current app trigger hotkey",
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readRawConfig()
		if err != nil {
			fmt.Println(defaultHotkey)
			return nil
		}
		if app, ok := raw["app"].(map[string]interface{}); ok {
			if hk, ok := app["hotkey"].(string); ok && hk != "" {
				fmt.Println(hk)
				return nil
			}
		}
		fmt.Println(defaultHotkey)
		return nil
	},
}

var setHotkeyCmd = &cobra.Command{
	Use:   "set-hotkey <hotkey>",
	Short: "Set the app trigger hotkey (e.g. Alt+Space, Ctrl+Shift+Space)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hotkey := strings.TrimSpace(args[0])
		if hotkey == "" {
			return fmt.Errorf("hotkey cannot be empty")
		}

		raw, err := readRawConfig()
		if err != nil {
			raw = map[string]interface{}{}
		}

		app, _ := raw["app"].(map[string]interface{})
		if app == nil {
			app = map[string]interface{}{}
		}
		app["hotkey"] = hotkey
		raw["app"] = app

		if err := writeRawConfig(raw); err != nil {
			return fmt.Errorf("cannot write config: %w", err)
		}
		fmt.Printf("✓ Hotkey set to '%s'\n", hotkey)
		return nil
	},
}
