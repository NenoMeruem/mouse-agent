//go:build !darwin

package trigger

import "fmt"

// ParseHotkey is not implemented on non-darwin platforms.
// Use daemon setup to generate config for xbindkeys / AutoHotkey.
func ParseHotkey(s string) (interface{}, interface{}, error) {
	return nil, nil, fmt.Errorf("ParseHotkey not supported on this platform")
}
