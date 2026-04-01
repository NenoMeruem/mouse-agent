//go:build darwin

package trigger

import (
	"fmt"
	"strings"

	"golang.design/x/hotkey"
)

// ParseHotkey parses "alt+space", "ctrl+shift+e", etc. into
// hotkey modifiers + key for macOS (Carbon API).
//
// Modifier aliases:
//
//	ctrl | control   → ModCtrl
//	alt | option     → ModOption
//	shift            → ModShift
//	cmd | command    → ModCmd
func ParseHotkey(s string) ([]hotkey.Modifier, hotkey.Key, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	if len(parts) < 2 {
		return nil, 0, fmt.Errorf("invalid hotkey %q: need at least one modifier+key", s)
	}

	var mods []hotkey.Modifier
	for _, p := range parts[:len(parts)-1] {
		m, err := parseMod(p)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid hotkey %q: %w", s, err)
		}
		mods = append(mods, m)
	}

	k, err := parseKey(parts[len(parts)-1])
	if err != nil {
		return nil, 0, fmt.Errorf("invalid hotkey %q: %w", s, err)
	}

	return mods, k, nil
}

func parseMod(s string) (hotkey.Modifier, error) {
	switch s {
	case "ctrl", "control":
		return hotkey.ModCtrl, nil
	case "alt", "option":
		return hotkey.ModOption, nil
	case "shift":
		return hotkey.ModShift, nil
	case "cmd", "command", "super", "win":
		return hotkey.ModCmd, nil
	default:
		return 0, fmt.Errorf("unknown modifier %q (use: ctrl, alt, shift, cmd)", s)
	}
}

func parseKey(s string) (hotkey.Key, error) {
	letters := map[string]hotkey.Key{
		"a": hotkey.KeyA, "b": hotkey.KeyB, "c": hotkey.KeyC, "d": hotkey.KeyD,
		"e": hotkey.KeyE, "f": hotkey.KeyF, "g": hotkey.KeyG, "h": hotkey.KeyH,
		"i": hotkey.KeyI, "j": hotkey.KeyJ, "k": hotkey.KeyK, "l": hotkey.KeyL,
		"m": hotkey.KeyM, "n": hotkey.KeyN, "o": hotkey.KeyO, "p": hotkey.KeyP,
		"q": hotkey.KeyQ, "r": hotkey.KeyR, "s": hotkey.KeyS, "t": hotkey.KeyT,
		"u": hotkey.KeyU, "v": hotkey.KeyV, "w": hotkey.KeyW, "x": hotkey.KeyX,
		"y": hotkey.KeyY, "z": hotkey.KeyZ,
	}
	if k, ok := letters[s]; ok {
		return k, nil
	}

	digits := map[string]hotkey.Key{
		"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3,
		"4": hotkey.Key4, "5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7,
		"8": hotkey.Key8, "9": hotkey.Key9,
	}
	if k, ok := digits[s]; ok {
		return k, nil
	}

	special := map[string]hotkey.Key{
		"space":     hotkey.KeySpace,
		"return":    hotkey.KeyReturn,
		"enter":     hotkey.KeyReturn,
		"escape":    hotkey.KeyEscape,
		"esc":       hotkey.KeyEscape,
		"tab":       hotkey.KeyTab,
		"backspace": hotkey.KeyDelete,
		"delete":    hotkey.KeyDelete,
		"up":        hotkey.KeyUp,
		"down":      hotkey.KeyDown,
		"left":      hotkey.KeyLeft,
		"right":     hotkey.KeyRight,
		"f1":        hotkey.KeyF1,
		"f2":        hotkey.KeyF2,
		"f3":        hotkey.KeyF3,
		"f4":        hotkey.KeyF4,
		"f5":        hotkey.KeyF5,
		"f6":        hotkey.KeyF6,
		"f7":        hotkey.KeyF7,
		"f8":        hotkey.KeyF8,
		"f9":        hotkey.KeyF9,
		"f10":       hotkey.KeyF10,
		"f11":       hotkey.KeyF11,
		"f12":       hotkey.KeyF12,
	}
	if k, ok := special[s]; ok {
		return k, nil
	}

	return 0, fmt.Errorf("unknown key %q", s)
}
