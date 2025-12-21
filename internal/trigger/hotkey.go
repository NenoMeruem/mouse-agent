package trigger

import (
	"fmt"
)

// HotkeyTrigger represents a hotkey-based trigger
// In MVP, this is just a placeholder that explains the architecture
// Real implementation uses external hotkey daemon (skhd on macOS, sxhkd on Linux, etc.)
type HotkeyTrigger struct {
	name     string
	hotkey   string // e.g., "alt+space"
	promptID string
	active   bool
}

func NewHotkeyTrigger(hotkey, promptID string) *HotkeyTrigger {
	return &HotkeyTrigger{
		name:     fmt.Sprintf("hotkey:%s", hotkey),
		hotkey:   hotkey,
		promptID: promptID,
		active:   false,
	}
}

func (h *HotkeyTrigger) Name() string {
	return h.name
}

func (h *HotkeyTrigger) Start() error {
	if h.active {
		return fmt.Errorf("hotkey trigger already running")
	}

	// MVP: Hotkey is configured in external daemon (skhd/sxhkd/autohotkey)
	// This just marks the trigger as active for management purposes
	h.active = true
	return nil
}

func (h *HotkeyTrigger) Stop() error {
	if !h.active {
		return fmt.Errorf("hotkey trigger not running")
	}

	h.active = false
	return nil
}

func (h *HotkeyTrigger) IsActive() bool {
	return h.active
}

func (h *HotkeyTrigger) GetPromptID() string {
	return h.promptID
}

func (h *HotkeyTrigger) GetHotkey() string {
	return h.hotkey
}
