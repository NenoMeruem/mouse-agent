package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sl/prompt-builder-agent/pkg/models"
)

const createHistoryTable = `
CREATE TABLE IF NOT EXISTS run_history (
    id           TEXT PRIMARY KEY,
    prompt_id    TEXT NOT NULL,
    engine       TEXT NOT NULL,
    input_text   TEXT DEFAULT '',
    final_prompt TEXT NOT NULL,
    response     TEXT DEFAULT '',
    duration_ms  INTEGER DEFAULT 0,
    error        TEXT DEFAULT '',
    created_at   DATETIME NOT NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id)
);

CREATE INDEX IF NOT EXISTS idx_history_prompt_id ON run_history(prompt_id);
CREATE INDEX IF NOT EXISTS idx_history_created_at ON run_history(created_at DESC);
`

// SQLiteHistoryStore is a SQLite-backed implementation of HistoryStore.
type SQLiteHistoryStore struct {
	db *sql.DB
}

// NewSQLiteHistoryStore creates a new history store sharing the given DB connection.
// It initialises the run_history table and indices if they don't exist.
func NewSQLiteHistoryStore(db *sql.DB) (*SQLiteHistoryStore, error) {
	if _, err := db.Exec(createHistoryTable); err != nil {
		return nil, fmt.Errorf("cannot create history table: %w", err)
	}
	return &SQLiteHistoryStore{db: db}, nil
}

// Append inserts a new run record into the history table.
func (h *SQLiteHistoryStore) Append(record *models.RunRecord) error {
	if record.ID == "" {
		return fmt.Errorf("record ID cannot be empty")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	_, err := h.db.Exec(
		`INSERT INTO run_history
		 (id, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.PromptID,
		record.Engine,
		record.InputText,
		record.FinalPrompt,
		record.Response,
		record.DurationMs,
		record.Error,
		record.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("cannot append run record: %w", err)
	}
	return nil
}

// List returns run records ordered by created_at DESC. If promptID is non-empty,
// only records for that prompt are returned. At most limit records are returned.
func (h *SQLiteHistoryStore) List(limit int, promptID string) ([]models.RunRecord, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if promptID != "" {
		rows, err = h.db.Query(
			`SELECT id, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
			 FROM run_history WHERE prompt_id = ?
			 ORDER BY created_at DESC LIMIT ?`,
			promptID, limit,
		)
	} else {
		rows, err = h.db.Query(
			`SELECT id, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
			 FROM run_history ORDER BY created_at DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot list run history: %w", err)
	}
	defer rows.Close()

	return scanRecords(rows)
}

// Search returns records where final_prompt or response match the LIKE query.
func (h *SQLiteHistoryStore) Search(query string) ([]models.RunRecord, error) {
	pattern := "%" + query + "%"
	rows, err := h.db.Query(
		`SELECT id, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
		 FROM run_history
		 WHERE final_prompt LIKE ? OR response LIKE ?
		 ORDER BY created_at DESC`,
		pattern, pattern,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot search run history: %w", err)
	}
	defer rows.Close()

	return scanRecords(rows)
}

// Clear deletes all records created before the given time.
func (h *SQLiteHistoryStore) Clear(before time.Time) error {
	_, err := h.db.Exec(
		`DELETE FROM run_history WHERE created_at < ?`,
		before.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("cannot clear run history: %w", err)
	}
	return nil
}

// --- helpers ----------------------------------------------------------------

func scanRecord(rows *sql.Rows) (*models.RunRecord, error) {
	var r models.RunRecord
	var createdStr string

	if err := rows.Scan(
		&r.ID, &r.PromptID, &r.Engine,
		&r.InputText, &r.FinalPrompt, &r.Response,
		&r.DurationMs, &r.Error, &createdStr,
	); err != nil {
		return nil, err
	}

	var err error
	r.CreatedAt, err = time.Parse(time.RFC3339Nano, createdStr)
	if err != nil {
		r.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	}

	return &r, nil
}

func scanRecords(rows *sql.Rows) ([]models.RunRecord, error) {
	var records []models.RunRecord
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("cannot scan run record: %w", err)
		}
		records = append(records, *r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if records == nil {
		records = []models.RunRecord{}
	}
	return records, nil
}
