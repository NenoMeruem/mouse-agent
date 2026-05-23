package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/meruem/promptly/internal/app"
	"github.com/meruem/promptly/internal/trigger"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
	daemonCmd.AddCommand(daemonSetupCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage hotkey trigger daemon",
	Long: `Start or configure the hotkey trigger daemon.

The daemon listens for system-wide keyboard shortcuts and runs the
corresponding prompt without needing to open a terminal manually.

Configure hotkeys in ~/.promptly/config.yaml:

  triggers:
    enabled: true
    hotkeys:
      alt+space: explain_code
      alt+shift+r: code_review

Supported modifiers: ctrl, alt/option, shift, cmd
Supported keys:      a-z, 0-9, space, return, escape, tab, f1-f12, arrows

macOS note: Grant Accessibility permission in
  System Preferences → Privacy & Security → Accessibility`,
	RunE: runDaemon,
}

var daemonSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate hotkey config for external tools (skhd / xbindkeys)",
	Long: `Generate hotkey configuration files for external tools.

macOS: writes ~/.skhdrc  (requires: brew install skhd)
Linux: writes ~/.xbindkeysrc  (requires: apt install xbindkeys)`,
	RunE: runDaemonSetup,
}

func runDaemon(_ *cobra.Command, _ []string) error {
	if app.GlobalContext.Config == nil || !app.GlobalContext.Config.Triggers.Enabled {
		return fmt.Errorf("triggers not enabled — add to ~/.promptly/config.yaml:\n" +
			"  triggers:\n    enabled: true\n    hotkeys:\n      alt+space: explain_code")
	}

	hotkeys := app.GlobalContext.Config.Triggers.Hotkeys
	if len(hotkeys) == 0 {
		return fmt.Errorf("no hotkeys configured in ~/.promptly/config.yaml")
	}

	// Resolve binary path so the daemon can launch itself
	binary, err := os.Executable()
	if err != nil {
		binary = "promptly"
	}

	daemon := trigger.NewHotkeyDaemon(binary)
	var parseErrors []string
	for hotkeyStr, promptID := range hotkeys {
		if err := daemon.Add(hotkeyStr, promptID); err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("  ✗ %s: %v", hotkeyStr, err))
		}
	}

	if len(parseErrors) > 0 {
		for _, e := range parseErrors {
			fmt.Fprintln(os.Stderr, e)
		}
		return fmt.Errorf("some hotkeys could not be parsed")
	}

	fmt.Println("🚀 Starting hotkey daemon...")
	fmt.Println("   Registered hotkeys:")

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	fmt.Println("\n✓ Daemon running — press Ctrl+C to stop")

	if err := daemon.Start(ctx); err != nil {
		return fmt.Errorf("daemon error: %w", err)
	}

	fmt.Println("\n🛑 Daemon stopped")
	return nil
}

func runDaemonSetup(_ *cobra.Command, _ []string) error {
	if app.GlobalContext.Config == nil {
		return fmt.Errorf("config not loaded")
	}
	hotkeys := app.GlobalContext.Config.Triggers.Hotkeys
	if len(hotkeys) == 0 {
		return fmt.Errorf("no hotkeys configured in ~/.promptly/config.yaml")
	}

	binary, _ := os.Executable()
	if binary == "" {
		binary = "promptly"
	}

	return trigger.GenerateExternalConfig(hotkeys, binary)
}
