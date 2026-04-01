package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

// newTestSQLiteStore creates a temporary SQLiteStore for testing.
func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	t.Cleanup(func() { store.DB().Close() })
	return store
}

// TestSQLiteCreate creates a prompt and verifies it can be retrieved.
func TestSQLiteCreate(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID:          "test-1",
		Name:        "Test Prompt",
		Description: "A test",
		Engine:      "openai",
		Template:    "Hello {{name}}",
		Variables:   []string{"name"},
	}

	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Get("test-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.ID != p.ID {
		t.Errorf("ID: want %q, got %q", p.ID, got.ID)
	}
	if got.Name != p.Name {
		t.Errorf("Name: want %q, got %q", p.Name, got.Name)
	}
	if got.Engine != p.Engine {
		t.Errorf("Engine: want %q, got %q", p.Engine, got.Engine)
	}
	if len(got.Variables) != 1 || got.Variables[0] != "name" {
		t.Errorf("Variables: want [name], got %v", got.Variables)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

// TestSQLiteCreateDuplicate verifies that inserting the same ID twice fails.
func TestSQLiteCreateDuplicate(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID: "dup", Name: "Dup", Engine: "openai", Template: "T",
	}
	if err := store.Create(p); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}
	if err := store.Create(p); err == nil {
		t.Error("expected error for duplicate ID, got nil")
	}
}

// TestSQLiteList creates multiple prompts and verifies List returns them newest first.
func TestSQLiteList(t *testing.T) {
	store := newTestSQLiteStore(t)

	ids := []string{"a", "b", "c"}
	for i, id := range ids {
		p := &models.Prompt{
			ID:     id,
			Name:   "Prompt " + id,
			Engine: "openai",
			Template: "T",
			// Stagger timestamps so ordering is deterministic.
			CreatedAt: time.Now().Add(time.Duration(i) * time.Millisecond),
			UpdatedAt: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
		if err := store.Create(p); err != nil {
			t.Fatalf("Create %q failed: %v", id, err)
		}
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(list))
	}

	// Newest first: "c" was inserted last with the largest offset.
	if list[0].ID != "c" {
		t.Errorf("expected first item to be 'c' (newest), got %q", list[0].ID)
	}
	if !list[0].CreatedAt.After(list[1].CreatedAt) {
		t.Error("list is not sorted newest first")
	}
}

// TestSQLiteGetNotFound verifies that Get returns an error for missing IDs.
func TestSQLiteGetNotFound(t *testing.T) {
	store := newTestSQLiteStore(t)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent ID, got nil")
	}
}

// TestSQLiteDelete verifies that a deleted prompt is no longer retrievable.
func TestSQLiteDelete(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID: "del-me", Name: "Delete Me", Engine: "openai", Template: "T",
	}
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := store.Delete("del-me"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := store.Get("del-me"); err == nil {
		t.Error("expected error after delete, got nil")
	}
}

// TestSQLiteDeleteNotFound verifies that deleting a missing ID returns an error.
func TestSQLiteDeleteNotFound(t *testing.T) {
	store := newTestSQLiteStore(t)
	if err := store.Delete("ghost"); err == nil {
		t.Error("expected error for nonexistent ID, got nil")
	}
}

// TestMigrateFromJSON verifies that prompts from a JSON file are imported into SQLite.
func TestMigrateFromJSON(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "prompts.json")
	dbPath := filepath.Join(dir, "prompts.db")

	// Build a small JSON file.
	now := time.Now().UTC().Truncate(time.Second)
	prompts := []models.Prompt{
		{
			ID: "m1", Name: "Migrated 1", Engine: "openai",
			Template: "T1", Variables: []string{"x"},
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "m2", Name: "Migrated 2", Engine: "gemini",
			Template: "T2", Variables: nil,
			CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
		},
	}
	data, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Create an empty SQLiteStore and run migration.
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore failed: %v", err)
	}
	defer store.DB().Close()

	if err := MigrateFromJSON(store, jsonPath); err != nil {
		t.Fatalf("MigrateFromJSON failed: %v", err)
	}

	// Verify prompts are in SQLite.
	list, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 migrated prompts, got %d", len(list))
	}

	// Verify the JSON file was renamed to .bak.
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Error("expected original JSON file to be gone after migration")
	}
	if _, err := os.Stat(jsonPath + ".bak"); os.IsNotExist(err) {
		t.Error("expected .bak file to exist after migration")
	}
}

// TestMigrateFromJSONNoFile verifies that migration is a no-op when JSON is absent.
func TestMigrateFromJSONNoFile(t *testing.T) {
	store := newTestSQLiteStore(t)
	err := MigrateFromJSON(store, "/nonexistent/path/prompts.json")
	if err != nil {
		t.Errorf("expected nil error when JSON missing, got %v", err)
	}
}

// TestMigrateFromJSONAlreadyMigrated verifies migration is skipped when SQLite has data.
func TestMigrateFromJSONAlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "prompts.json")

	// Write minimal JSON.
	if err := os.WriteFile(jsonPath, []byte(`[{"id":"x","name":"X","engine":"openai","template":"T","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}]`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := newTestSQLiteStore(t)

	// Pre-populate SQLite.
	if err := store.Create(&models.Prompt{ID: "existing", Name: "E", Engine: "openai", Template: "T"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := MigrateFromJSON(store, jsonPath); err != nil {
		t.Fatalf("MigrateFromJSON: %v", err)
	}

	// The JSON file should NOT have been touched.
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON file should not be renamed when SQLite already has data")
	}

	// Count: only the pre-existing prompt.
	list, _ := store.List()
	if len(list) != 1 {
		t.Errorf("expected 1 prompt (no migration), got %d", len(list))
	}
}
