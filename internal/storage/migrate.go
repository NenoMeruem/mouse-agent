package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

// MigrateFromJSON checks if jsonPath exists and no prompts are in SQLite yet.
// If so, it reads the JSON file, inserts each prompt into SQLite (preserving
// original timestamps), and renames the JSON file to jsonPath+".bak".
// This runs silently and only once.
func MigrateFromJSON(store *SQLiteStore, jsonPath string) error {
	// 1. Check if JSON file exists.
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return nil // nothing to migrate
	}

	// 2. Check if SQLite already has prompts.
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM prompts`).Scan(&count); err != nil {
		return fmt.Errorf("cannot count prompts: %w", err)
	}
	if count > 0 {
		return nil // already migrated
	}

	// 3. Read and unmarshal the JSON file.
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("cannot read JSON file: %w", err)
	}

	var prompts []models.Prompt
	if err := json.Unmarshal(data, &prompts); err != nil {
		return fmt.Errorf("cannot unmarshal JSON: %w", err)
	}

	// 4. Insert each prompt into SQLite, preserving original timestamps.
	for i := range prompts {
		if err := store.Create(&prompts[i]); err != nil {
			return fmt.Errorf("cannot insert prompt %q: %w", prompts[i].ID, err)
		}
	}

	// 5. Rename JSON file to .bak.
	if err := os.Rename(jsonPath, jsonPath+".bak"); err != nil {
		return fmt.Errorf("cannot rename JSON file: %w", err)
	}

	return nil
}
