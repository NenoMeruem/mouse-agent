//go:build darwin

package trigger

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"golang.design/x/hotkey"
)

// HotkeyBinding maps a parsed hotkey to a prompt ID
type HotkeyBinding struct {
	Raw      string // e.g. "alt+space"
	PromptID string
	hk       *hotkey.Hotkey
}

// HotkeyDaemon listens for OS-level global hotkeys and fires prompt-agent.
// On macOS this uses the Carbon RegisterEventHotKey API via golang.design/x/hotkey.
// Requires Accessibility permissions (System Preferences → Privacy → Accessibility).
type HotkeyDaemon struct {
	bindings []*HotkeyBinding
	binary   string // path to prompt-agent binary
	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
}

// NewHotkeyDaemon creates a daemon ready to register hotkeys.
// binary is the path to the prompt-agent executable (os.Executable()).
func NewHotkeyDaemon(binary string) *HotkeyDaemon {
	return &HotkeyDaemon{binary: binary}
}

// Add registers one hotkey → promptID binding (before Start).
func (d *HotkeyDaemon) Add(hotkeyStr, promptID string) error {
	mods, key, err := ParseHotkey(hotkeyStr)
	if err != nil {
		return err
	}
	hk := hotkey.New(mods, key)
	d.bindings = append(d.bindings, &HotkeyBinding{
		Raw:      hotkeyStr,
		PromptID: promptID,
		hk:       hk,
	})
	return nil
}

// Start registers all hotkeys with the OS and begins listening.
// Blocks until ctx is cancelled.
func (d *HotkeyDaemon) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	d.running = true
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		d.running = false
		d.mu.Unlock()
	}()

	// Register all hotkeys
	registered := make([]*HotkeyBinding, 0, len(d.bindings))
	for _, b := range d.bindings {
		if err := b.hk.Register(); err != nil {
			// Unregister already registered ones on failure
			for _, r := range registered {
				_ = r.hk.Unregister()
			}
			return fmt.Errorf("failed to register hotkey %q: %w\n(hint: grant Accessibility permission in System Preferences → Privacy & Security → Accessibility)", b.Raw, err)
		}
		registered = append(registered, b)
		fmt.Printf("  ⌨️  %s  →  %s\n", b.Raw, b.PromptID)
	}

	defer func() {
		for _, b := range registered {
			_ = b.hk.Unregister()
		}
	}()

	// Fan out: one goroutine per hotkey
	var wg sync.WaitGroup
	for _, b := range registered {
		wg.Add(1)
		go func(b *HotkeyBinding) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-b.hk.Keydown():
					d.fire(b)
				}
			}
		}(b)
	}

	<-ctx.Done()
	wg.Wait()
	return nil
}

// fire opens a terminal window and runs prompt-agent run <promptID>
func (d *HotkeyDaemon) fire(b *HotkeyBinding) {
	fmt.Printf("🔥 Hotkey %s → running prompt: %s\n", b.Raw, b.PromptID)

	binary := d.binary
	if binary == "" {
		var err error
		binary, err = os.Executable()
		if err != nil {
			binary = "prompt-agent"
		}
	}

	cmd := fmt.Sprintf("%s run %s", binary, b.PromptID)

	// Try to open in the user's preferred terminal
	for _, launcher := range terminalLaunchers(cmd) {
		if err := exec.Command(launcher[0], launcher[1:]...).Start(); err == nil {
			return
		}
	}

	// Last resort: run in background (no TUI)
	_ = exec.Command(binary, "run", b.PromptID, "--raw").Start()
}

// terminalLaunchers returns a prioritised list of ways to open a terminal
// window on macOS and run cmd inside it.
func terminalLaunchers(cmd string) [][]string {
	escaped := strings.ReplaceAll(cmd, `"`, `\"`)
	return [][]string{
		// iTerm2
		{"osascript", "-e",
			`tell application "iTerm2" to create window with default profile command "` + escaped + `"`},
		// Terminal.app (standard)
		{"osascript", "-e",
			`tell application "Terminal" to do script "` + escaped + `"`},
		// Warp
		{"open", "-a", "Warp", "--args", cmd},
		// Alacritty
		{"osascript", "-e",
			`tell application "Alacritty" to activate`},
	}
}
