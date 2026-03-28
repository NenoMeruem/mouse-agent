//go:build windows

package output

import (
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "Set-Clipboard -Value $input")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
