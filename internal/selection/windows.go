//go:build windows

package selection

import "errors"

type WindowsProvider struct{}

func (p *WindowsProvider) Get() (string, error) {
	return "", errors.New("selection provider not implemented for Windows yet")
}

func getOSProvider() Provider {
	return &WindowsProvider{}
}
