// Package llm provides a plugin architecture for registering and using different
// LLM backends (OpenAI, Gemini, etc.) with a consistent interface.
package llm

import (
	"fmt"
)

// Manager is a registry for different LLM provider clients.
// It allows dynamic registration and retrieval of LLM providers.
type Manager struct {
	clients map[string]Client
}

// NewManager creates a new LLM manager with no providers registered.
// Providers must be registered via Register() before use.
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]Client),
	}
}

// Register registers an LLM provider client by name.
// Returns an error if a client with the same name is already registered.
func (m *Manager) Register(name string, client Client) error {
	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("client %q already registered", name)
	}
	m.clients[name] = client
	return nil
}

// Get retrieves a registered LLM provider by name.
// Returns an error if the provider is not found.
func (m *Manager) Get(name string) (Client, error) {
	client, exists := m.clients[name]
	if !exists {
		return nil, fmt.Errorf("LLM provider %q not found (registered: %v)", name, m.List())
	}
	return client, nil
}

// List returns all registered provider names.
func (m *Manager) List() []string {
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}
