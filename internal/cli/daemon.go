package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sl/prompt-builder-agent/internal/app"
	"github.com/sl/prompt-builder-agent/internal/trigger"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Start trigger daemon (for hotkey support)",
	Long: `Start the trigger daemon to listen for configured hotkeys.
	
This daemon enables quick prompt execution via hotkeys without needing to open terminal.
Configure hotkeys in ~/.prompt-agent/config.yaml:

triggers:
  enabled: true
  hotkeys:
    alt+space: explain_code
    alt+shift+space: code_review
    
Note: On macOS, configure hotkeys in ~/.skhdrc instead (skhd is more reliable).`,
	RunE: runDaemon,
}

func runDaemon(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Starting trigger daemon...")

	// Create trigger manager
	mgr := trigger.NewManager()

	// Register hotkey triggers from config
	if app.GlobalContext.Config != nil && app.GlobalContext.Config.Triggers.Enabled {
		for hotkey, promptID := range app.GlobalContext.Config.Triggers.Hotkeys {
			fmt.Printf("  📍 Registered: %s -> %s\n", hotkey, promptID)
			mgr.Register(trigger.NewHotkeyTrigger(hotkey, promptID))
		}
	}

	if len(mgr.List()) == 0 {
		fmt.Println("⚠️  No triggers configured. Check ~/.prompt-agent/config.yaml")
		return nil
	}

	// Start all triggers
	if err := mgr.StartAll(); err != nil {
		return fmt.Errorf("cannot start triggers: %w", err)
	}

	fmt.Println("✓ Daemon running. Press Ctrl+C to stop.")

	// Wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Stopping daemon...")
	if err := mgr.StopAll(); err != nil {
		return fmt.Errorf("error stopping triggers: %w", err)
	}

	fmt.Println("✓ Daemon stopped")
	return nil
}
