package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
	_ "modernc.org/sqlite"
)

const createPromptsTable = `
CREATE TABLE IF NOT EXISTS prompts (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    engine      TEXT NOT NULL,
    template    TEXT NOT NULL,
    variables   TEXT DEFAULT '[]',
    params      TEXT DEFAULT '[]',
    icon        TEXT DEFAULT '',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);`

// SQLiteStore is a SQLite-backed implementation of PromptStore.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) the SQLite database at dbPath and
// initialises the schema. The directory is created if it does not exist.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createPromptsTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("cannot create prompts table: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB so it can be shared with other stores.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// GetDefaultDBPath returns the default file path for the SQLite database.
func GetDefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prompt-agent", "prompts.db")
}

// Create inserts a new prompt. Returns an error if the ID already exists.
func (s *SQLiteStore) Create(prompt *models.Prompt) error {
	if prompt.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	if prompt.CreatedAt.IsZero() {
		now := time.Now()
		prompt.CreatedAt = now
		prompt.UpdatedAt = now
	}
	if prompt.UpdatedAt.IsZero() {
		prompt.UpdatedAt = time.Now()
	}

	varsJSON, err := json.Marshal(prompt.Variables)
	if err != nil {
		return fmt.Errorf("cannot marshal variables: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO prompts (id, name, description, engine, template, variables, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		prompt.Engine,
		prompt.Template,
		string(varsJSON),
		prompt.CreatedAt.UTC().Format(time.RFC3339Nano),
		prompt.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("cannot insert prompt: %w", err)
	}
	return nil
}

// List returns all prompts sorted by created_at descending.
func (s *SQLiteStore) List() ([]models.Prompt, error) {
	rows, err := s.db.Query(
		`SELECT id, name, description, engine, template, variables, created_at, updated_at
		 FROM prompts ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot list prompts: %w", err)
	}
	defer rows.Close()

	return scanPrompts(rows)
}

// Get retrieves a single prompt by ID. Returns an error if not found.
func (s *SQLiteStore) Get(id string) (*models.Prompt, error) {
	row := s.db.QueryRow(
		`SELECT id, name, description, engine, template, variables, created_at, updated_at
		 FROM prompts WHERE id = ?`, id,
	)

	p, err := scanPrompt(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get prompt: %w", err)
	}
	return p, nil
}

// Delete removes a prompt by ID. Returns an error if the prompt does not exist.
func (s *SQLiteStore) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM prompts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("cannot delete prompt: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("prompt not found: %s", id)
	}
	return nil
}

// Update modifies an existing prompt. Returns error if not found.
func (s *SQLiteStore) Update(prompt *models.Prompt) error {
	prompt.UpdatedAt = time.Now()

	varsJSON, err := json.Marshal(prompt.Variables)
	if err != nil {
		return fmt.Errorf("cannot marshal variables: %w", err)
	}

	result, err := s.db.Exec(
		`UPDATE prompts SET name=?, description=?, engine=?, template=?, variables=?, updated_at=? WHERE id=?`,
		prompt.Name, prompt.Description, prompt.Engine, prompt.Template,
		string(varsJSON), prompt.UpdatedAt.UTC().Format(time.RFC3339Nano),
		prompt.ID,
	)
	if err != nil {
		return fmt.Errorf("cannot update prompt: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("prompt not found: %s", prompt.ID)
	}
	return nil
}

// Search returns prompts whose name or template contain the query string (case-insensitive LIKE).
func (s *SQLiteStore) Search(query string) ([]models.Prompt, error) {
	if query == "" {
		return []models.Prompt{}, nil
	}
	pattern := "%" + query + "%"
	rows, err := s.db.Query(
		`SELECT id, name, description, engine, template, variables, created_at, updated_at
		 FROM prompts WHERE name LIKE ? OR template LIKE ?
		 ORDER BY created_at DESC`,
		pattern, pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot search prompts: %w", err)
	}
	defer rows.Close()

	return scanPrompts(rows)
}

// --- helpers ----------------------------------------------------------------

type scanner interface {
	Scan(dest ...any) error
}

func scanPrompt(row scanner) (*models.Prompt, error) {
	var p models.Prompt
	var varsJSON, createdStr, updatedStr string

	if err := row.Scan(
		&p.ID, &p.Name, &p.Description, &p.Engine, &p.Template,
		&varsJSON, &createdStr, &updatedStr,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(varsJSON), &p.Variables); err != nil {
		return nil, fmt.Errorf("cannot unmarshal variables: %w", err)
	}

	var err error
	p.CreatedAt, err = time.Parse(time.RFC3339Nano, createdStr)
	if err != nil {
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	}
	p.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedStr)
	if err != nil {
		p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	}

	return &p, nil
}

func scanPrompts(rows *sql.Rows) ([]models.Prompt, error) {
	var prompts []models.Prompt
	for rows.Next() {
		p, err := scanPrompt(rows)
		if err != nil {
			return nil, fmt.Errorf("cannot scan prompt: %w", err)
		}
		prompts = append(prompts, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if prompts == nil {
		prompts = []models.Prompt{}
	}
	return prompts, nil
}
