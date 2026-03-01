// Package storage defines interfaces and implementations for persisting prompts.
package storage

import "github.com/sl/prompt-builder-agent/pkg/models"

// PromptStore defines operations for storing and retrieving prompts.
// Implementations must be thread-safe for concurrent access.
type PromptStore interface {
	// Create adds a new prompt to storage. Returns error if ID already exists.
	Create(prompt *models.Prompt) error

	// List returns all prompts, sorted by creation date (newest first).
	List() ([]models.Prompt, error)

	// Get retrieves a prompt by ID. Returns error if not found.
	Get(id string) (*models.Prompt, error)

	// Delete removes a prompt by ID. Returns error if not found.
	Delete(id string) error
}
