package selection

import (
	"fmt"
)

type Manager struct {
	provider Provider
}

func NewManager(provider Provider) *Manager {
	return &Manager{
		provider: provider,
	}
}

func (m *Manager) GetSelection() (string, error) {
	if m.provider == nil {
		return "", fmt.Errorf("selection provider not initialized")
	}

	text, err := m.provider.Get()
	if err != nil {
		return "", fmt.Errorf("cannot get selection from %s: %w", m.provider.Name(), err)
	}

	if text == "" {
		return "", fmt.Errorf("selection is empty")
	}

	return text, nil
}

func (m *Manager) GetProvider() Provider {
	return m.provider
}
