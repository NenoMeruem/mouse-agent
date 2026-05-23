package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meruem/promptly/pkg/models"
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

// TestCreateWithParams verifies that Params and Icon are stored and retrieved correctly.
func TestCreateWithParams(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID:        "params-1",
		Name:      "Parameterised Prompt",
		Engine:    "gemini",
		Template:  "Explain {{selection}}",
		Variables: []string{"selection"},
		Params:    []string{"tone", "length"},
		Icon:      "📝",
	}

	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := store.Get("params-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(got.Params) != 2 || got.Params[0] != "tone" || got.Params[1] != "length" {
		t.Errorf("Params: want [tone length], got %v", got.Params)
	}
	if got.Icon != "📝" {
		t.Errorf("Icon: want %q, got %q", "📝", got.Icon)
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

// TestSQLiteList creates multiple prompts and verifies List returns them sorted by sort_order.
func TestSQLiteList(t *testing.T) {
	store := newTestSQLiteStore(t)

	ids := []string{"a", "b", "c"}
	for i, id := range ids {
		p := &models.Prompt{
			ID:       id,
			Name:     "Prompt " + id,
			Engine:   "openai",
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

	// Items are ordered by sort_order ASC (insertion order: a=0, b=1, c=2).
	if list[0].ID != "a" {
		t.Errorf("expected first item to be 'a' (sort_order=0), got %q", list[0].ID)
	}
	if list[0].SortOrder > list[1].SortOrder {
		t.Error("list is not sorted by sort_order ascending")
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

// --- HistoryStore tests -----------------------------------------------------

// newTestHistoryStore creates a SQLiteHistoryStore backed by a temporary DB,
// and inserts a seed prompt so foreign-key constraints can be satisfied.
func newTestHistoryStore(t *testing.T) (*SQLiteHistoryStore, *SQLiteStore) {
	t.Helper()
	store := newTestSQLiteStore(t)

	seed := &models.Prompt{
		ID: "seed-prompt", Name: "Seed", Engine: "openai", Template: "T",
	}
	if err := store.Create(seed); err != nil {
		t.Fatalf("seed Create failed: %v", err)
	}

	hs, err := NewSQLiteHistoryStore(store.DB())
	if err != nil {
		t.Fatalf("NewSQLiteHistoryStore failed: %v", err)
	}
	return hs, store
}

// TestHistoryAppendAndList appends a record and verifies it can be listed back.
func TestHistoryAppendAndList(t *testing.T) {
	hs, _ := newTestHistoryStore(t)

	rec := &models.RunRecord{
		ID:          "run-1",
		PromptID:    "seed-prompt",
		Engine:      "openai",
		InputText:   "hello",
		FinalPrompt: "explain hello",
		Response:    "Hello is a greeting.",
		DurationMs:  42,
	}
	if err := hs.Append(rec); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	records, err := hs.List(10, "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	got := records[0]
	if got.ID != rec.ID {
		t.Errorf("ID: want %q, got %q", rec.ID, got.ID)
	}
	if got.PromptID != rec.PromptID {
		t.Errorf("PromptID: want %q, got %q", rec.PromptID, got.PromptID)
	}
	if got.Response != rec.Response {
		t.Errorf("Response: want %q, got %q", rec.Response, got.Response)
	}
	if got.DurationMs != rec.DurationMs {
		t.Errorf("DurationMs: want %d, got %d", rec.DurationMs, got.DurationMs)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

// TestHistoryListByPromptID appends records for two different prompt IDs and
// verifies that filtering by promptID returns only matching records.
func TestHistoryListByPromptID(t *testing.T) {
	hs, store := newTestHistoryStore(t)

	// Create a second prompt for the foreign key.
	other := &models.Prompt{
		ID: "other-prompt", Name: "Other", Engine: "gemini", Template: "T2",
	}
	if err := store.Create(other); err != nil {
		t.Fatalf("Create other prompt failed: %v", err)
	}

	for _, r := range []models.RunRecord{
		{ID: "r1", PromptID: "seed-prompt", Engine: "openai", FinalPrompt: "fp1"},
		{ID: "r2", PromptID: "seed-prompt", Engine: "openai", FinalPrompt: "fp2"},
		{ID: "r3", PromptID: "other-prompt", Engine: "gemini", FinalPrompt: "fp3"},
	} {
		rc := r
		if err := hs.Append(&rc); err != nil {
			t.Fatalf("Append %q failed: %v", r.ID, err)
		}
	}

	got, err := hs.List(10, "seed-prompt")
	if err != nil {
		t.Fatalf("List by promptID failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 records for seed-prompt, got %d", len(got))
	}
	for _, r := range got {
		if r.PromptID != "seed-prompt" {
			t.Errorf("unexpected PromptID %q in filtered list", r.PromptID)
		}
	}
}

// TestHistorySearch appends records and verifies Search finds by final_prompt text.
func TestHistorySearch(t *testing.T) {
	hs, _ := newTestHistoryStore(t)

	records := []models.RunRecord{
		{ID: "s1", PromptID: "seed-prompt", Engine: "openai", FinalPrompt: "explain the pipeline failure"},
		{ID: "s2", PromptID: "seed-prompt", Engine: "openai", FinalPrompt: "summarize the meeting notes"},
		{ID: "s3", PromptID: "seed-prompt", Engine: "openai", FinalPrompt: "pipeline diagnostics"},
	}
	for i := range records {
		if err := hs.Append(&records[i]); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	got, err := hs.Search("pipeline")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results for 'pipeline', got %d", len(got))
	}
}

// TestHistoryClear appends records with different timestamps and verifies
// that Clear removes only records before the cutoff time.
func TestHistoryClear(t *testing.T) {
	hs, _ := newTestHistoryStore(t)

	base := time.Now().UTC().Add(-2 * time.Hour)
	cutoff := base.Add(time.Hour)

	for i, ts := range []time.Time{
		base,                       // before cutoff — should be cleared
		base.Add(30 * time.Minute), // before cutoff — should be cleared
		cutoff.Add(time.Minute),    // after cutoff — should remain
	} {
		r := &models.RunRecord{
			ID:          fmt.Sprintf("c%d", i),
			PromptID:    "seed-prompt",
			Engine:      "openai",
			FinalPrompt: "fp",
			CreatedAt:   ts,
		}
		if err := hs.Append(r); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if err := hs.Clear(cutoff); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	remaining, err := hs.List(0, "")
	if err != nil {
		t.Fatalf("List after Clear failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 record after Clear, got %d", len(remaining))
	}
}

// TestHistoryListNoLimit verifies that limit <= 0 returns all records.
func TestHistoryListNoLimit(t *testing.T) {
	hs, _ := newTestHistoryStore(t)

	for i := 0; i < 5; i++ {
		r := &models.RunRecord{
			ID:          fmt.Sprintf("nl%d", i),
			PromptID:    "seed-prompt",
			Engine:      "openai",
			FinalPrompt: "fp",
		}
		if err := hs.Append(r); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	all, err := hs.List(0, "")
	if err != nil {
		t.Fatalf("List(0) failed: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 records with limit=0, got %d", len(all))
	}
}

// TestUpdate creates a prompt, updates its name and description, and verifies Get returns updated values.
func TestUpdate(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID:          "upd-1",
		Name:        "Original Name",
		Description: "Original desc",
		Engine:      "openai",
		Template:    "Hello {{name}}",
		Variables:   []string{"name"},
	}
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	p.Name = "Updated Name"
	p.Description = "Updated desc"
	if err := store.Update(p); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Get("upd-1")
	if err != nil {
		t.Fatalf("Get after Update failed: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name: want %q, got %q", "Updated Name", got.Name)
	}
	if got.Description != "Updated desc" {
		t.Errorf("Description: want %q, got %q", "Updated desc", got.Description)
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero after update")
	}
}

// TestUpdateNotFound verifies that updating a non-existent ID returns an error.
func TestUpdateNotFound(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID:       "ghost",
		Name:     "Ghost",
		Engine:   "openai",
		Template: "T",
	}
	if err := store.Update(p); err == nil {
		t.Error("expected error for nonexistent ID, got nil")
	}
}

// TestUpdateParamsAndIcon verifies that Params and Icon can be updated.
func TestUpdateParamsAndIcon(t *testing.T) {
	store := newTestSQLiteStore(t)

	p := &models.Prompt{
		ID:       "upd-params",
		Name:     "Original",
		Engine:   "gemini",
		Template: "T",
		Params:   []string{"tone"},
		Icon:     "🔧",
	}
	if err := store.Create(p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	p.Params = []string{"tone", "length", "complexity"}
	p.Icon = "📝"
	if err := store.Update(p); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Get("upd-params")
	if err != nil {
		t.Fatalf("Get after Update failed: %v", err)
	}
	if len(got.Params) != 3 {
		t.Errorf("Params: want 3 items, got %v", got.Params)
	}
	if got.Icon != "📝" {
		t.Errorf("Icon: want %q, got %q", "📝", got.Icon)
	}
}

// TestSearch creates two prompts and verifies Search returns only the matching one.
func TestSearch(t *testing.T) {
	store := newTestSQLiteStore(t)

	p1 := &models.Prompt{
		ID:        "s1",
		Name:      "Explain Code",
		Engine:    "openai",
		Template:  "Explain this code snippet",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	p2 := &models.Prompt{
		ID:        "s2",
		Name:      "Summarize Text",
		Engine:    "gemini",
		Template:  "Summarize the following text",
		CreatedAt: time.Now().Add(-time.Minute),
		UpdatedAt: time.Now().Add(-time.Minute),
	}
	if err := store.Create(p1); err != nil {
		t.Fatalf("Create p1 failed: %v", err)
	}
	if err := store.Create(p2); err != nil {
		t.Fatalf("Create p2 failed: %v", err)
	}

	results, err := store.Search("Explain")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ID != "s1" {
		t.Errorf("expected result ID %q, got %q", "s1", results[0].ID)
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
