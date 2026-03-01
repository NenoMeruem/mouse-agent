//go:build linux

package selection

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type LinuxProvider struct{}

func (p *LinuxProvider) Name() string {
	return "Linux"
}

func (p *LinuxProvider) Get() (string, error) {
	// Try xclip first (most common X11 clipboard tool)
	text, err := tryGetClipboard("xclip", "-selection", "clipboard", "-o")
	if err == nil {
		return text, nil
	}

	// Fallback to xsel (alternative X11 clipboard tool)
	text, err = tryGetClipboard("xsel", "--clipboard", "--output")
	if err == nil {
		return text, nil
	}

	// Fallback to wl-paste (Wayland clipboard tool)
	text, err = tryGetClipboard("wl-paste")
	if err == nil {
		return text, nil
	}

	return "", fmt.Errorf("cannot get clipboard: no clipboard tool found (tried xclip, xsel, wl-paste). Please install xclip or xsel")
}

// tryGetClipboard attempts to get clipboard content using the specified command
func tryGetClipboard(cmdName string, args ...string) (string, error) {
	cmd := exec.Command(cmdName, args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", fmt.Errorf("clipboard is empty")
	}

	return text, nil
}

func getOSProvider() Provider {
	return &LinuxProvider{}
}
