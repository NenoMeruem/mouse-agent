// Package storage defines interfaces and implementations for persisting prompts.
package storage

import (
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

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

// HistoryStore defines operations for storing and retrieving run history records.
type HistoryStore interface {
	// Append adds a new run record to storage.
	Append(record *models.RunRecord) error

	// List returns run records. If promptID is non-empty, filters by prompt.
	// Returns at most limit records, ordered newest first.
	List(limit int, promptID string) ([]models.RunRecord, error)

	// Search finds records where final_prompt or response match the query (LIKE).
	Search(query string) ([]models.RunRecord, error)

	// Clear deletes records created before the given time.
	Clear(before time.Time) error
}
