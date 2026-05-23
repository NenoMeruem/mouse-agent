package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/meruem/promptly/pkg/models"
)

// JSONStore provides thread-safe storage for prompts using JSON files.
// All operations are protected by RWMutex to ensure safe concurrent access.
// Files are written atomically to prevent corruption.
type JSONStore struct {
	filePath string
	prompts  map[string]*models.Prompt
	mu       sync.RWMutex // Protects prompts map and file access
}

// NewJSONStore creates a new JSON-backed prompt store.
// Loads prompts from file if it exists, otherwise initializes with hardcoded prompts.
func NewJSONStore(filePath string) (*JSONStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create directory: %w", err)
	}

	store := &JSONStore{
		filePath: filePath,
		prompts:  make(map[string]*models.Prompt),
	}

	// Try to load existing prompts from file
	if _, err := os.Stat(filePath); err == nil {
		// File exists, load from it
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("cannot load prompts: %w", err)
		}
	} else {
		// File doesn't exist, initialize with hardcoded example prompts
		now := time.Now()
		examplePrompts := []models.Prompt{
			{
				ID:          "example",
				Name:        "Example Prompt",
				Description: "This is a sample prompt. Edit or delete it.",
				Engine:      "openai",
				Template:    "Write a short description of {{.topic}}.",
				Variables:   []string{"topic"},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          "summarize",
				Name:        "Summarize Text",
				Description: "Summarize the provided text in concise bullet points.",
				Engine:      "openai",
				Template:    "Please summarize the following text:\n{{.text}}",
				Variables:   []string{"text"},
				CreatedAt:   now.Add(-1 * time.Hour),
				UpdatedAt:   now.Add(-1 * time.Hour),
			},
			{
				ID:          "translate",
				Name:        "Translate Text",
				Description: "Translate text from one language to another.",
				Engine:      "openai",
				Template:    "Translate the following {{.source_lang}} text to {{.target_lang}}:\n{{.text}}",
				Variables:   []string{"source_lang", "target_lang", "text"},
				CreatedAt:   now.Add(-2 * time.Hour),
				UpdatedAt:   now.Add(-2 * time.Hour),
			},
		}

		// Add hardcoded prompts to store
		for i := range examplePrompts {
			store.prompts[examplePrompts[i].ID] = &examplePrompts[i]
		}
	}

	return store, nil
}

func (s *JSONStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's fine
		}
		return fmt.Errorf("cannot read file: %w", err)
	}

	if len(data) == 0 {
		return nil // File is empty, that's fine
	}

	var prompts []models.Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	for i := range prompts {
		s.prompts[prompts[i].ID] = &prompts[i]
	}

	return nil
}

// save writes prompts to file atomically using temp file + rename pattern.
// This ensures the file is never left in a corrupted state.
func (s *JSONStore) save() error {
	// Convert map to sorted slice
	prompts := make([]models.Prompt, 0, len(s.prompts))
	for _, p := range s.prompts {
		prompts = append(prompts, *p)
	}

	// Sort by CreatedAt descending
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].CreatedAt.After(prompts[j].CreatedAt)
	})

	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal JSON: %w", err)
	}

	// Atomic write: write to temp file first, then rename
	// This prevents corruption if process crashes mid-write
	dir := filepath.Dir(s.filePath)
	tmpFile, err := os.CreateTemp(dir, ".prompt-agent.tmp.")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("cannot write temp file: %w", err)
	}

	// Ensure data is written to disk before rename
	if err := tmpFile.Sync(); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("cannot sync temp file: %w", err)
	}

	// Atomic rename (POSIX) - on Windows this may not be atomic but is best-effort
	if err := os.Rename(tmpFile.Name(), s.filePath); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("cannot write file: %w", err)
	}

	return nil
}

func (s *JSONStore) Create(prompt *models.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if prompt.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	if _, exists := s.prompts[prompt.ID]; exists {
		return fmt.Errorf("prompt with ID %s already exists", prompt.ID)
	}

	now := time.Now()
	prompt.CreatedAt = now
	prompt.UpdatedAt = now

	s.prompts[prompt.ID] = prompt

	return s.save()
}

func (s *JSONStore) List() ([]models.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]models.Prompt, 0, len(s.prompts))
	for _, p := range s.prompts {
		prompts = append(prompts, *p)
	}

	// Sort by CreatedAt descending
	sort.Slice(prompts, func(i, j int) bool {
		return prompts[i].CreatedAt.After(prompts[j].CreatedAt)
	})

	return prompts, nil
}

func (s *JSONStore) Get(id string) (*models.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompt, exists := s.prompts[id]
	if !exists {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	return prompt, nil
}

func (s *JSONStore) Update(prompt *models.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.prompts[prompt.ID]; !exists {
		return fmt.Errorf("prompt not found: %s", prompt.ID)
	}
	prompt.UpdatedAt = time.Now()
	s.prompts[prompt.ID] = prompt
	return s.save()
}

func (s *JSONStore) Search(query string) ([]models.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query = strings.ToLower(query)
	var results []models.Prompt
	for _, p := range s.prompts {
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Template), query) {
			results = append(results, *p)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results, nil
}

func (s *JSONStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.prompts[id]; !exists {
		return fmt.Errorf("prompt not found: %s", id)
	}

	delete(s.prompts, id)
	return s.save()
}

// GetDefaultJSONPath returns the default file path for storing prompts.
// It uses ~/.prompt-agent/prompts.json
func GetDefaultJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prompt-agent", "prompts.json")
}
