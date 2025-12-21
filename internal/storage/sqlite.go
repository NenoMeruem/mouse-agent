package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sl/prompt-builder-agent/pkg/models"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS prompts (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	engine TEXT NOT NULL,
	template TEXT NOT NULL,
	variables TEXT NOT NULL,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
`

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("cannot ping database: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init() error {
	_, err := s.db.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("cannot initialize schema: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Create(prompt *models.Prompt) error {
	if prompt.ID == "" {
		return fmt.Errorf("prompt ID cannot be empty")
	}

	variablesJSON, err := json.Marshal(prompt.Variables)
	if err != nil {
		return fmt.Errorf("cannot marshal variables: %w", err)
	}

	now := time.Now()
	prompt.CreatedAt = now
	prompt.UpdatedAt = now

	_, err = s.db.Exec(
		`INSERT INTO prompts (id, name, description, engine, template, variables, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		prompt.ID,
		prompt.Name,
		prompt.Description,
		prompt.Engine,
		prompt.Template,
		string(variablesJSON),
		prompt.CreatedAt,
		prompt.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("cannot insert prompt: %w", err)
	}

	return nil
}

func (s *SQLiteStore) List() ([]models.Prompt, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, engine, template, variables, created_at, updated_at
		FROM prompts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("cannot query prompts: %w", err)
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		var variablesJSON string

		err := rows.Scan(
			&prompt.ID,
			&prompt.Name,
			&prompt.Description,
			&prompt.Engine,
			&prompt.Template,
			&variablesJSON,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("cannot scan row: %w", err)
		}

		err = json.Unmarshal([]byte(variablesJSON), &prompt.Variables)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal variables: %w", err)
		}

		prompts = append(prompts, prompt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return prompts, nil
}

func (s *SQLiteStore) Get(id string) (*models.Prompt, error) {
	var prompt models.Prompt
	var variablesJSON string

	err := s.db.QueryRow(`
		SELECT id, name, description, engine, template, variables, created_at, updated_at
		FROM prompts
		WHERE id = ?
	`, id).Scan(
		&prompt.ID,
		&prompt.Name,
		&prompt.Description,
		&prompt.Engine,
		&prompt.Template,
		&variablesJSON,
		&prompt.CreatedAt,
		&prompt.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("prompt not found: %s", id)
		}
		return nil, fmt.Errorf("cannot query prompt: %w", err)
	}

	err = json.Unmarshal([]byte(variablesJSON), &prompt.Variables)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal variables: %w", err)
	}

	return &prompt, nil
}

func (s *SQLiteStore) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM prompts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("cannot delete prompt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("cannot get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt not found: %s", id)
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func GetDefaultDBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".prompt-agent", "prompts.db")
}
