package trigger

import (
	"fmt"
	"sync"
)

type Manager struct {
	triggers []Trigger
	mu       sync.RWMutex
	running  bool
}

func NewManager() *Manager {
	return &Manager{
		triggers: make([]Trigger, 0),
		running:  false,
	}
}

func (m *Manager) Register(t Trigger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.triggers = append(m.triggers, t)
}

func (m *Manager) StartAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("triggers already running")
	}

	for _, t := range m.triggers {
		if err := t.Start(); err != nil {
			// Stop already started triggers on failure
			for _, started := range m.triggers {
				if started.IsActive() {
					_ = started.Stop()
				}
			}
			return fmt.Errorf("cannot start trigger %s: %w", t.Name(), err)
		}
	}

	m.running = true
	return nil
}

func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("triggers not running")
	}

	var lastErr error
	for _, t := range m.triggers {
		if t.IsActive() {
			if err := t.Stop(); err != nil {
				lastErr = err
			}
		}
	}

	m.running = false
	return lastErr
}

func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, len(m.triggers))
	for i, t := range m.triggers {
		names[i] = t.Name()
	}
	return names
}
