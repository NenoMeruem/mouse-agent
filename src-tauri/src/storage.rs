// storage.rs — SQLite database layer.
// Manages DB connection, schema migrations, and CRUD for Prompt + RunRecord.
// Mirrors Go's internal/storage/ package (sqlite.go + history_store.go).

use crate::models::{Prompt, RunRecord};
use crate::prompt::extract_variables;
use chrono::{DateTime, Utc};
use rusqlite::{params, Connection, Result as SqlResult};
use std::path::Path;

// ---------------------------------------------------------------------------
// Database initialisation & migrations
// ---------------------------------------------------------------------------

/// Opens (or creates) the SQLite database at `db_path` and runs all required
/// schema migrations before returning the connection.
pub fn open_db(db_path: &Path) -> SqlResult<Connection> {
    if let Some(parent) = db_path.parent() {
        std::fs::create_dir_all(parent).ok();
    }
    let conn = Connection::open(db_path)?;
    conn.execute_batch("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=OFF;")?;
    run_migrations(&conn)?;
    Ok(conn)
}

/// Applies all schema migrations idempotently.
fn run_migrations(conn: &Connection) -> SqlResult<()> {
    // ── prompts table ──────────────────────────────────────────────────────
    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS prompts (
            id          TEXT PRIMARY KEY,
            name        TEXT NOT NULL,
            description TEXT DEFAULT '',
            engine      TEXT NOT NULL DEFAULT '',
            template    TEXT NOT NULL,
            variables   TEXT DEFAULT '[]',
            params      TEXT DEFAULT '[]',
            icon        TEXT DEFAULT '',
            sort_order  INTEGER NOT NULL DEFAULT 0,
            created_at  TEXT NOT NULL,
            updated_at  TEXT NOT NULL
        );",
    )?;
    // Add sort_order if it's missing (existing DBs from Go that pre-date the column)
    conn.execute_batch(
        "ALTER TABLE prompts ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;",
    )
    .ok();

    // ── run_history table ──────────────────────────────────────────────────
    // Recreate without FK constraint if the old version (with FK) exists.
    migrate_drop_history_fk(conn)?;

    conn.execute_batch(
        "CREATE TABLE IF NOT EXISTS run_history (
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
            created_at   TEXT NOT NULL
        );
        CREATE INDEX IF NOT EXISTS idx_history_prompt_id  ON run_history(prompt_id);
        CREATE INDEX IF NOT EXISTS idx_history_session_id ON run_history(session_id);
        CREATE INDEX IF NOT EXISTS idx_history_created_at ON run_history(created_at);",
    )?;
    // Add session/turn columns to existing DBs
    conn.execute_batch(
        "ALTER TABLE run_history ADD COLUMN session_id TEXT DEFAULT '';",
    )
    .ok();
    conn.execute_batch(
        "ALTER TABLE run_history ADD COLUMN turn_index INTEGER DEFAULT 0;",
    )
    .ok();

    Ok(())
}

/// Drops the FOREIGN KEY constraint from run_history if it still exists by
/// recreating the table — mirrors Go's `migrateDropHistoryFK`.
fn migrate_drop_history_fk(conn: &Connection) -> SqlResult<()> {
    let mut stmt = conn.prepare(
        "SELECT sql FROM sqlite_master WHERE type='table' AND name='run_history'",
    )?;
    let table_def: Option<String> = stmt
        .query_map([], |row| row.get(0))?
        .filter_map(|r| r.ok())
        .next();

    match table_def {
        None => return Ok(()), // table doesn't exist yet
        Some(def) if !def.to_uppercase().contains("FOREIGN KEY") => return Ok(()),
        _ => {}
    }

    conn.execute_batch(
        "CREATE TABLE run_history_v2 (
            id           TEXT PRIMARY KEY,
            prompt_id    TEXT NOT NULL,
            engine       TEXT NOT NULL,
            input_text   TEXT DEFAULT '',
            final_prompt TEXT NOT NULL,
            response     TEXT DEFAULT '',
            duration_ms  INTEGER DEFAULT 0,
            error        TEXT DEFAULT '',
            created_at   TEXT NOT NULL
        );
        INSERT INTO run_history_v2 SELECT
            id, prompt_id, engine, input_text, final_prompt, response,
            duration_ms, error, created_at
        FROM run_history;
        DROP TABLE run_history;
        ALTER TABLE run_history_v2 RENAME TO run_history;",
    )?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Date-time helpers
// ---------------------------------------------------------------------------

fn fmt_dt(dt: &DateTime<Utc>) -> String {
    dt.format("%Y-%m-%dT%H:%M:%S%.9fZ").to_string()
}

fn parse_dt(s: &str) -> DateTime<Utc> {
    DateTime::parse_from_rfc3339(s)
        .map(|d| d.with_timezone(&Utc))
        .unwrap_or_else(|_| Utc::now())
}

// ---------------------------------------------------------------------------
// Prompt Store
// ---------------------------------------------------------------------------

/// Inserts a new prompt. Returns an error if the ID already exists.
pub fn prompt_create(conn: &Connection, p: &Prompt) -> SqlResult<()> {
    let vars = serde_json::to_string(&p.variables).unwrap_or_else(|_| "[]".into());
    let params_json = serde_json::to_string(&p.params).unwrap_or_else(|_| "[]".into());
    conn.execute(
        "INSERT INTO prompts
            (id, name, description, engine, template, variables, params, icon, sort_order, created_at, updated_at)
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11)",
        params![
            p.id, p.name, p.description, p.engine, p.template,
            vars, params_json, p.icon, p.sort_order,
            fmt_dt(&p.created_at), fmt_dt(&p.updated_at),
        ],
    )?;
    Ok(())
}

/// Returns all prompts ordered by `sort_order ASC, created_at DESC`.
pub fn prompt_list(conn: &Connection) -> SqlResult<Vec<Prompt>> {
    let mut stmt = conn.prepare(
        "SELECT id,name,description,engine,template,variables,params,icon,sort_order,created_at,updated_at
         FROM prompts ORDER BY sort_order ASC, created_at DESC",
    )?;
    collect_prompts(&mut stmt, [])
}

/// Returns a single prompt by ID, or an error if not found.
pub fn prompt_get(conn: &Connection, id: &str) -> SqlResult<Prompt> {
    let mut stmt = conn.prepare(
        "SELECT id,name,description,engine,template,variables,params,icon,sort_order,created_at,updated_at
         FROM prompts WHERE id=?1",
    )?;
    let mut rows = collect_prompts(&mut stmt, [id])?;
    rows.pop().ok_or_else(|| {
        rusqlite::Error::QueryReturnedNoRows
    })
}

/// Updates an existing prompt's mutable fields. `updated_at` is set to now.
pub fn prompt_update(conn: &Connection, p: &Prompt) -> SqlResult<()> {
    let vars = serde_json::to_string(&p.variables).unwrap_or_else(|_| "[]".into());
    let params_json = serde_json::to_string(&p.params).unwrap_or_else(|_| "[]".into());
    let updated_at = fmt_dt(&Utc::now());
    let n = conn.execute(
        "UPDATE prompts SET name=?1,description=?2,engine=?3,template=?4,
            variables=?5,params=?6,icon=?7,sort_order=?8,updated_at=?9
         WHERE id=?10",
        params![
            p.name, p.description, p.engine, p.template,
            vars, params_json, p.icon, p.sort_order, updated_at, p.id,
        ],
    )?;
    if n == 0 {
        return Err(rusqlite::Error::QueryReturnedNoRows);
    }
    Ok(())
}

/// Deletes a prompt by ID. Returns an error if it doesn't exist.
pub fn prompt_delete(conn: &Connection, id: &str) -> SqlResult<()> {
    let n = conn.execute("DELETE FROM prompts WHERE id=?1", params![id])?;
    if n == 0 {
        return Err(rusqlite::Error::QueryReturnedNoRows);
    }
    Ok(())
}

/// LIKE search on `name` and `template` fields.
pub fn prompt_search(conn: &Connection, query: &str) -> SqlResult<Vec<Prompt>> {
    if query.is_empty() {
        return Ok(vec![]);
    }
    let pattern = format!("%{}%", query);
    let mut stmt = conn.prepare(
        "SELECT id,name,description,engine,template,variables,params,icon,sort_order,created_at,updated_at
         FROM prompts WHERE name LIKE ?1 OR template LIKE ?1
         ORDER BY sort_order ASC, created_at DESC",
    )?;
    collect_prompts(&mut stmt, [pattern])
}

/// Sets `sort_order = 0, 1, 2, …` for the given IDs in provided order.
pub fn prompt_reorder(conn: &Connection, ids: &[String]) -> SqlResult<()> {
    for (i, id) in ids.iter().enumerate() {
        conn.execute(
            "UPDATE prompts SET sort_order=?1 WHERE id=?2",
            params![i as i64, id],
        )?;
    }
    Ok(())
}

/// Returns the next sort_order value (max + 1) for auto-assignment.
pub fn prompt_next_sort_order(conn: &Connection) -> i64 {
    conn.query_row(
        "SELECT COALESCE(MAX(sort_order), -1) FROM prompts",
        [],
        |row| row.get::<_, i64>(0),
    )
    .unwrap_or(-1)
        + 1
}

// ---------------------------------------------------------------------------
// Helpers — query execution
// ---------------------------------------------------------------------------

fn collect_prompts(
    stmt: &mut rusqlite::Statement<'_>,
    params: impl rusqlite::Params,
) -> SqlResult<Vec<Prompt>> {
    let rows = stmt.query_map(params, |row| {
        Ok((
            row.get::<_, String>(0)?,  // id
            row.get::<_, String>(1)?,  // name
            row.get::<_, String>(2)?,  // description
            row.get::<_, String>(3)?,  // engine
            row.get::<_, String>(4)?,  // template
            row.get::<_, String>(5)?,  // variables JSON
            row.get::<_, String>(6)?,  // params JSON
            row.get::<_, String>(7)?,  // icon
            row.get::<_, i64>(8)?,     // sort_order
            row.get::<_, String>(9)?,  // created_at
            row.get::<_, String>(10)?, // updated_at
        ))
    })?;

    let mut prompts = Vec::new();
    for row in rows {
        let (id, name, description, engine, template, vars_json, params_json, icon, sort_order, created_str, updated_str) = row?;
        let variables: Vec<String> = serde_json::from_str(&vars_json).unwrap_or_default();
        let params_vec: Vec<String> = serde_json::from_str(&params_json).unwrap_or_default();
        prompts.push(Prompt {
            id,
            name,
            description,
            engine,
            template,
            variables,
            params: params_vec,
            icon,
            sort_order,
            created_at: parse_dt(&created_str),
            updated_at: parse_dt(&updated_str),
        });
    }
    Ok(prompts)
}

// ---------------------------------------------------------------------------
// History Store
// ---------------------------------------------------------------------------

/// Inserts a new run record.
pub fn history_append(conn: &Connection, r: &RunRecord) -> SqlResult<()> {
    conn.execute(
        "INSERT INTO run_history
            (id,session_id,turn_index,prompt_id,engine,input_text,final_prompt,response,duration_ms,error,created_at)
         VALUES (?1,?2,?3,?4,?5,?6,?7,?8,?9,?10,?11)",
        params![
            r.id, r.session_id, r.turn_index, r.prompt_id, r.engine,
            r.input_text, r.final_prompt, r.response, r.duration_ms, r.error,
            fmt_dt(&r.created_at),
        ],
    )?;
    Ok(())
}

/// Lists run records, optionally filtered by `prompt_id`, limited by `limit`
/// (0 = unlimited), ordered newest first.
pub fn history_list(
    conn: &Connection,
    limit: i64,
    prompt_id: &str,
) -> SqlResult<Vec<RunRecord>> {
    let sql_limit = if limit <= 0 { -1 } else { limit };
    let mut stmt = if prompt_id.is_empty() {
        conn.prepare(
            "SELECT id,session_id,turn_index,prompt_id,engine,input_text,
                    final_prompt,response,duration_ms,error,created_at
             FROM run_history ORDER BY created_at DESC LIMIT ?1",
        )?
    } else {
        // SQLite positional params: re-use pattern using same ?1
        conn.prepare(
            "SELECT id,session_id,turn_index,prompt_id,engine,input_text,
                    final_prompt,response,duration_ms,error,created_at
             FROM run_history WHERE prompt_id=?2
             ORDER BY created_at DESC LIMIT ?1",
        )?
    };

    let rows = if prompt_id.is_empty() {
        stmt.query_map(params![sql_limit], scan_record)?
    } else {
        stmt.query_map(params![sql_limit, prompt_id], scan_record)?
    };

    let mut records = Vec::new();
    for r in rows {
        records.push(r?);
    }
    Ok(records)
}

/// LIKE search on `final_prompt` and `response`.
pub fn history_search(conn: &Connection, query: &str) -> SqlResult<Vec<RunRecord>> {
    let pattern = format!("%{}%", query);
    let mut stmt = conn.prepare(
        "SELECT id,session_id,turn_index,prompt_id,engine,input_text,
                final_prompt,response,duration_ms,error,created_at
         FROM run_history
         WHERE final_prompt LIKE ?1 OR response LIKE ?1
         ORDER BY created_at DESC",
    )?;
    let rows = stmt.query_map(params![pattern], scan_record)?;
    let mut records = Vec::new();
    for r in rows {
        records.push(r?);
    }
    Ok(records)
}

/// Deletes all history records.
pub fn history_clear_all(conn: &Connection) -> SqlResult<()> {
    conn.execute("DELETE FROM run_history", [])?;
    Ok(())
}

fn scan_record(row: &rusqlite::Row<'_>) -> SqlResult<RunRecord> {
    Ok(RunRecord {
        id: row.get(0)?,
        session_id: row.get(1)?,
        turn_index: row.get(2)?,
        prompt_id: row.get(3)?,
        engine: row.get(4)?,
        input_text: row.get(5)?,
        final_prompt: row.get(6)?,
        response: row.get(7)?,
        duration_ms: row.get(8)?,
        error: row.get(9)?,
        created_at: parse_dt(&row.get::<_, String>(10)?),
    })
}

// ---------------------------------------------------------------------------
// Seeding helpers
// ---------------------------------------------------------------------------

/// Seeds the database with default recipes if the prompts table is empty.
pub fn seed_default_recipes_if_empty(conn: &Connection) -> SqlResult<()> {
    use crate::prompt::default_recipes;
    let count: i64 =
        conn.query_row("SELECT COUNT(*) FROM prompts", [], |r| r.get(0))?;
    if count == 0 {
        for mut recipe in default_recipes() {
            // Re-extract variables from template in case they differ
            recipe.variables = extract_variables(&recipe.template);
            prompt_create(conn, &recipe)?;
        }
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

#[cfg(test)]
mod tests {
    use super::*;
    use crate::models::Prompt;
    use chrono::Utc;

    fn test_db() -> Connection {
        let conn = Connection::open_in_memory().unwrap();
        run_migrations(&conn).unwrap();
        conn
    }

    fn make_prompt(id: &str, sort_order: i64) -> Prompt {
        let now = Utc::now();
        Prompt {
            id: id.to_string(),
            name: format!("Prompt {}", id),
            description: "desc".into(),
            engine: "gemini".into(),
            template: "Hello {{selection}}".into(),
            variables: vec!["selection".into()],
            params: vec![],
            icon: "🤖".into(),
            sort_order,
            created_at: now,
            updated_at: now,
        }
    }

    // ── Prompt CRUD ────────────────────────────────────────────────────────

    #[test]
    fn create_and_get_prompt() {
        let conn = test_db();
        let p = make_prompt("p1", 0);
        prompt_create(&conn, &p).unwrap();
        let got = prompt_get(&conn, "p1").unwrap();
        assert_eq!(got.id, "p1");
        assert_eq!(got.engine, "gemini");
    }

    #[test]
    fn list_prompts_sorted_by_sort_order() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("b", 1)).unwrap();
        prompt_create(&conn, &make_prompt("a", 0)).unwrap();
        let list = prompt_list(&conn).unwrap();
        assert_eq!(list[0].id, "a");
        assert_eq!(list[1].id, "b");
    }

    #[test]
    fn get_nonexistent_prompt_returns_error() {
        let conn = test_db();
        assert!(prompt_get(&conn, "does-not-exist").is_err());
    }

    #[test]
    fn update_prompt() {
        let conn = test_db();
        let mut p = make_prompt("p1", 0);
        prompt_create(&conn, &p).unwrap();
        p.name = "Updated Name".into();
        prompt_update(&conn, &p).unwrap();
        let got = prompt_get(&conn, "p1").unwrap();
        assert_eq!(got.name, "Updated Name");
    }

    #[test]
    fn delete_prompt() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("p1", 0)).unwrap();
        prompt_delete(&conn, "p1").unwrap();
        assert!(prompt_get(&conn, "p1").is_err());
    }

    #[test]
    fn delete_nonexistent_returns_error() {
        let conn = test_db();
        assert!(prompt_delete(&conn, "ghost").is_err());
    }

    #[test]
    fn search_prompts_by_name() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("matching", 0)).unwrap();
        prompt_create(&conn, &make_prompt("other", 1)).unwrap();
        let results = prompt_search(&conn, "matching").unwrap();
        assert_eq!(results.len(), 1);
        assert_eq!(results[0].id, "matching");
    }

    #[test]
    fn search_empty_query_returns_empty() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("p1", 0)).unwrap();
        let results = prompt_search(&conn, "").unwrap();
        assert!(results.is_empty());
    }

    #[test]
    fn reorder_prompts() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("a", 0)).unwrap();
        prompt_create(&conn, &make_prompt("b", 1)).unwrap();
        prompt_reorder(&conn, &["b".to_string(), "a".to_string()]).unwrap();
        let list = prompt_list(&conn).unwrap();
        assert_eq!(list[0].id, "b");
        assert_eq!(list[1].id, "a");
    }

    #[test]
    fn prompt_next_sort_order_empty_db() {
        let conn = test_db();
        assert_eq!(prompt_next_sort_order(&conn), 0);
    }

    #[test]
    fn prompt_next_sort_order_with_data() {
        let conn = test_db();
        prompt_create(&conn, &make_prompt("p1", 5)).unwrap();
        assert_eq!(prompt_next_sort_order(&conn), 6);
    }

    // ── History CRUD ───────────────────────────────────────────────────────

    fn make_record(id: &str, prompt_id: &str) -> RunRecord {
        RunRecord {
            id: id.to_string(),
            session_id: "sess-1".into(),
            turn_index: 0,
            prompt_id: prompt_id.to_string(),
            engine: "gemini".into(),
            input_text: "input".into(),
            final_prompt: "final".into(),
            response: "the response".into(),
            duration_ms: 100,
            error: "".into(),
            created_at: Utc::now(),
        }
    }

    #[test]
    fn append_and_list_history() {
        let conn = test_db();
        history_append(&conn, &make_record("r1", "p1")).unwrap();
        let list = history_list(&conn, 0, "").unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].id, "r1");
    }

    #[test]
    fn list_history_filtered_by_prompt_id() {
        let conn = test_db();
        history_append(&conn, &make_record("r1", "p1")).unwrap();
        history_append(&conn, &make_record("r2", "p2")).unwrap();
        let list = history_list(&conn, 0, "p1").unwrap();
        assert_eq!(list.len(), 1);
        assert_eq!(list[0].prompt_id, "p1");
    }

    #[test]
    fn list_history_respects_limit() {
        let conn = test_db();
        history_append(&conn, &make_record("r1", "p1")).unwrap();
        history_append(&conn, &make_record("r2", "p1")).unwrap();
        let list = history_list(&conn, 1, "").unwrap();
        assert_eq!(list.len(), 1);
    }

    #[test]
    fn search_history_by_response() {
        let conn = test_db();
        history_append(&conn, &make_record("r1", "p1")).unwrap();
        let results = history_search(&conn, "the response").unwrap();
        assert_eq!(results.len(), 1);
    }

    #[test]
    fn clear_all_history() {
        let conn = test_db();
        history_append(&conn, &make_record("r1", "p1")).unwrap();
        history_clear_all(&conn).unwrap();
        let list = history_list(&conn, 0, "").unwrap();
        assert!(list.is_empty());
    }

    // ── Seed ──────────────────────────────────────────────────────────────

    #[test]
    fn seed_default_recipes_populates_empty_db() {
        let conn = test_db();
        seed_default_recipes_if_empty(&conn).unwrap();
        let list = prompt_list(&conn).unwrap();
        assert_eq!(list.len(), 4);
    }

    #[test]
    fn seed_default_recipes_is_idempotent() {
        let conn = test_db();
        seed_default_recipes_if_empty(&conn).unwrap();
        seed_default_recipes_if_empty(&conn).unwrap();
        let list = prompt_list(&conn).unwrap();
        assert_eq!(list.len(), 4);
    }
}
