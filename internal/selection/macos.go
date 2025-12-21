//go:build darwin

package selection

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type MacOSProvider struct{}

func (p *MacOSProvider) Name() string {
	return "macOS"
}

func (p *MacOSProvider) Get() (string, error) {
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cannot get clipboard: %w", err)
	}

	text := strings.TrimSpace(out.String())
	if text == "" {
		return "", fmt.Errorf("clipboard is empty")
	}

	return text, nil
}

func getOSProvider() Provider {
	return &MacOSProvider{}
}
