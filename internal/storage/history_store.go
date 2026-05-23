package storage

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/meruem/promptly/pkg/models"
)

const createHistoryTable = `
CREATE TABLE IF NOT EXISTS run_history (
    id           TEXT PRIMARY KEY,
    session_id   TEXT DEFAULT '',
    turn_index   INTEGER DEFAULT 0,
    prompt_id    TEXT NOT NULL,
    engine       TEXT NOT NULL,
    input_text   TEXT DEFAULT '',
    final_prompt TEXT NOT NULL,
    response     TEXT DEFAULT '',
    duration_ms  INTEGER DEFAULT 0,
    error        TEXT DEFAULT '',
    created_at   DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_history_prompt_id  ON run_history(prompt_id);
CREATE INDEX IF NOT EXISTS idx_history_session_id ON run_history(session_id);
CREATE INDEX IF NOT EXISTS idx_history_created_at ON run_history(created_at DESC);
`

// SQLiteHistoryStore is a SQLite-backed implementation of HistoryStore.
type SQLiteHistoryStore struct {
	db *sql.DB
}

// NewSQLiteHistoryStore creates a new history store sharing the given DB connection.
// It migrates away any existing FOREIGN KEY constraint then ensures the table exists.
func NewSQLiteHistoryStore(db *sql.DB) (*SQLiteHistoryStore, error) {
	if err := migrateDropHistoryFK(db); err != nil {
		return nil, fmt.Errorf("cannot migrate history table: %w", err)
	}
	// Add session_id / turn_index to existing DBs BEFORE creating indices that reference them.
	// These are no-ops when the columns already exist or the table doesn't exist yet.
	db.Exec(`ALTER TABLE run_history ADD COLUMN session_id TEXT DEFAULT ''`)   //nolint:errcheck
	db.Exec(`ALTER TABLE run_history ADD COLUMN turn_index INTEGER DEFAULT 0`) //nolint:errcheck
	if _, err := db.Exec(createHistoryTable); err != nil {
		return nil, fmt.Errorf("cannot create history table: %w", err)
	}
	return &SQLiteHistoryStore{db: db}, nil
}

// migrateDropHistoryFK removes the FOREIGN KEY constraint from run_history if
// it still exists. SQLite requires a table-recreation to drop constraints.
func migrateDropHistoryFK(db *sql.DB) error {
	var tableDef string
	err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='run_history'`,
	).Scan(&tableDef)
	if err == sql.ErrNoRows {
		return nil // table doesn't exist yet — nothing to migrate
	}
	if err != nil {
		return err
	}
	if !strings.Contains(strings.ToUpper(tableDef), "FOREIGN KEY") {
		return nil // already clean
	}

	// Disable FK enforcement during migration (must be outside a transaction).
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if tx != nil {
			tx.Rollback() //nolint:errcheck
		}
		db.Exec(`PRAGMA foreign_keys = ON`) //nolint:errcheck
	}()

	steps := []string{
		`CREATE TABLE run_history_v2 (
			id           TEXT PRIMARY KEY,
			prompt_id    TEXT NOT NULL,
			engine       TEXT NOT NULL,
			input_text   TEXT DEFAULT '',
			final_prompt TEXT NOT NULL,
			response     TEXT DEFAULT '',
			duration_ms  INTEGER DEFAULT 0,
			error        TEXT DEFAULT '',
			created_at   DATETIME NOT NULL
		)`,
		`INSERT INTO run_history_v2 SELECT * FROM run_history`,
		`DROP TABLE run_history`,
		`ALTER TABLE run_history_v2 RENAME TO run_history`,
	}
	for _, q := range steps {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("migration step failed: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	tx = nil // prevent rollback in defer
	return nil
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
		 (id, session_id, turn_index, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.SessionID,
		record.TurnIndex,
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
// If limit <= 0, all records are returned.
func (h *SQLiteHistoryStore) List(limit int, promptID string) ([]models.RunRecord, error) {
	var (
		rows *sql.Rows
		err  error
	)

	// SQLite treats LIMIT -1 as unlimited; LIMIT 0 returns zero rows.
	sqlLimit := limit
	if limit <= 0 {
		sqlLimit = -1
	}

	if promptID != "" {
		rows, err = h.db.Query(
			`SELECT id, session_id, turn_index, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
			 FROM run_history WHERE prompt_id = ?
			 ORDER BY created_at DESC LIMIT ?`,
			promptID, sqlLimit,
		)
	} else {
		rows, err = h.db.Query(
			`SELECT id, session_id, turn_index, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
			 FROM run_history ORDER BY created_at DESC LIMIT ?`,
			sqlLimit,
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
		`SELECT id, session_id, turn_index, prompt_id, engine, input_text, final_prompt, response, duration_ms, error, created_at
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

// ClearAll deletes every run record unconditionally.
func (h *SQLiteHistoryStore) ClearAll() error {
	_, err := h.db.Exec(`DELETE FROM run_history`)
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
		&r.ID, &r.SessionID, &r.TurnIndex, &r.PromptID, &r.Engine,
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
