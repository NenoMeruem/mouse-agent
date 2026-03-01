package storage

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

// TestNewJSONStore tests the creation of a new JSONStore
func TestNewJSONStore(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "prompts.json")

	store, err := NewJSONStore(filePath)
	if err != nil {
		t.Fatalf("NewJSONStore failed: %v", err)
	}

	if store == nil {
		t.Error("Expected non-nil store")
	}

	if store.filePath != filePath {
		t.Errorf("Expected filePath %q, got %q", filePath, store.filePath)
	}
}

// TestCreate tests creating a new prompt
func TestCreate(t *testing.T) {
	store := newTestStore(t)

	prompt := &models.Prompt{
		ID:        "test-1",
		Name:      "Test Prompt",
		Template:  "Hello {{name}}",
		Engine:    "openai",
		Variables: []string{"name"},
	}

	err := store.Create(prompt)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify prompt was created
	retrieved, err := store.Get("test-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != prompt.ID {
		t.Errorf("Expected ID %q, got %q", prompt.ID, retrieved.ID)
	}
}

// TestCreateDuplicate tests that creating duplicate IDs fails
func TestCreateDuplicate(t *testing.T) {
	store := newTestStore(t)

	prompt := &models.Prompt{
		ID:       "test-1",
		Name:     "Test Prompt",
		Template: "Hello {{name}}",
		Engine:   "openai",
	}

	err := store.Create(prompt)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// Try creating again with same ID
	err = store.Create(prompt)
	if err == nil {
		t.Error("Expected error for duplicate ID, got nil")
	}
}

// TestList tests listing all prompts
func TestList(t *testing.T) {
	store := newTestStore(t)

	prompts := []models.Prompt{
		{ID: "test-1", Name: "Prompt 1", Template: "Template 1", Engine: "openai"},
		{ID: "test-2", Name: "Prompt 2", Template: "Template 2", Engine: "gemini"},
		{ID: "test-3", Name: "Prompt 3", Template: "Template 3", Engine: "openai"},
	}

	for i := range prompts {
		err := store.Create(&prompts[i])
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		// Add small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("Expected 3 prompts, got %d", len(list))
	}

	// Verify they are sorted by creation date (newest first)
	if len(list) >= 2 && !list[0].CreatedAt.After(list[1].CreatedAt) {
		t.Error("Prompts are not sorted by creation date (newest first)")
	}
}

// TestDelete tests deleting a prompt
func TestDelete(t *testing.T) {
	store := newTestStore(t)

	prompt := &models.Prompt{
		ID:       "test-1",
		Name:     "Test Prompt",
		Template: "Hello {{name}}",
		Engine:   "openai",
	}

	err := store.Create(prompt)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.Delete("test-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err = store.Get("test-1")
	if err == nil {
		t.Error("Expected error for deleted prompt, got nil")
	}
}

// TestConcurrentCreates tests thread-safe concurrent Create operations
func TestConcurrentCreates(t *testing.T) {
	store := newTestStore(t)
	numGoroutines := 10
	promptsPerGoroutine := 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*promptsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for p := 0; p < promptsPerGoroutine; p++ {
				id := fmt.Sprintf("prompt-%d-%d", goroutineID, p)
				prompt := &models.Prompt{
					ID:       id,
					Name:     fmt.Sprintf("Prompt %s", id),
					Template: "Template",
					Engine:   "openai",
				}
				if err := store.Create(prompt); err != nil {
					errors <- err
				}
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent create error: %v", err)
	}

	// Verify all prompts were created
	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	expected := numGoroutines * promptsPerGoroutine
	if len(list) != expected {
		t.Errorf("Expected %d prompts, got %d", expected, len(list))
	}
}

// TestConcurrentReads tests thread-safe concurrent Get operations
func TestConcurrentReads(t *testing.T) {
	store := newTestStore(t)

	// Create some prompts first
	for i := 0; i < 5; i++ {
		prompt := &models.Prompt{
			ID:       fmt.Sprintf("prompt-%d", i),
			Name:     fmt.Sprintf("Prompt %d", i),
			Template: "Template",
			Engine:   "openai",
		}
		store.Create(prompt)
	}

	// Concurrently read
	var wg sync.WaitGroup
	numReaders := 10
	readsPerReader := 20

	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < readsPerReader; i++ {
				_, _ = store.Get("prompt-0")
			}
		}()
	}

	wg.Wait()
	// If we got here without deadlock or panic, concurrent reads are safe
}

// TestEmptyID tests that empty ID is rejected
func TestEmptyID(t *testing.T) {
	store := newTestStore(t)

	prompt := &models.Prompt{
		ID:       "",
		Name:     "Test",
		Template: "Template",
		Engine:   "openai",
	}

	err := store.Create(prompt)
	if err == nil {
		t.Error("Expected error for empty ID, got nil")
	}
}

// Helper function to create a test store
func newTestStore(t *testing.T) *JSONStore {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test-prompts.json")

	store, err := NewJSONStore(filePath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	return store
}
