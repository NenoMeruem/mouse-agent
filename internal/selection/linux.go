//go:build linux

package selection

import "errors"

type LinuxProvider struct{}

func (p *LinuxProvider) Name() string {
	return "Linux"
}

func (p *LinuxProvider) Get() (string, error) {
	return "", errors.New("selection provider not implemented for Linux yet")
}

func getOSProvider() Provider {
	return &LinuxProvider{}
}
