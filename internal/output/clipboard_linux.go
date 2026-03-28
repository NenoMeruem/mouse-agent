//go:build linux

package output

import (
	"fmt"
	"os/exec"
	"strings"
)

func copyToClipboard(text string) error {
	for _, args := range [][]string{
		{"xclip", "-selection", "clipboard"},
		{"xsel", "--clipboard", "--input"},
		{"wl-copy"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no clipboard tool found (tried xclip, xsel, wl-copy)")
}
