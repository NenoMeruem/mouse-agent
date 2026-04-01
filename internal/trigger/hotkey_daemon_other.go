//go:build !darwin

package trigger

import (
	"context"
	"fmt"
)

// HotkeyBinding maps a hotkey string to a prompt ID.
type HotkeyBinding struct {
	Raw      string
	PromptID string
}

// HotkeyDaemon is a stub for non-macOS platforms.
// On Linux use xbindkeys or sxhkd; on Windows use AutoHotkey.
// Run `prompt-agent daemon setup` to generate the config for your platform.
type HotkeyDaemon struct {
	bindings []*HotkeyBinding
	binary   string
}

func NewHotkeyDaemon(binary string) *HotkeyDaemon {
	return &HotkeyDaemon{binary: binary}
}

func (d *HotkeyDaemon) Add(hotkeyStr, promptID string) error {
	d.bindings = append(d.bindings, &HotkeyBinding{
		Raw:      hotkeyStr,
		PromptID: promptID,
	})
	return nil
}

func (d *HotkeyDaemon) Start(_ context.Context) error {
	return fmt.Errorf(
		"built-in hotkey daemon is macOS-only.\n" +
			"Run `prompt-agent daemon setup` to generate xbindkeys/sxhkd config for Linux.",
	)
}
