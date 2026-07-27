# Architecture & Technical Documentation

This document outlines the architecture, data models, IPC flow, and key components of **Promptly**.

---

## Overview

Promptly is designed as a lightweight, low-latency desktop overlay app built with **Tauri v2** and **Rust**. 

```
┌─────────────────────────────────────────────────────────┐
│                      Tauri Overlay                      │
│   (Vanilla HTML5 / CSS3 / ES Modules JavaScript UI)     │
└────────────────────────────┬────────────────────────────┘
                             │ Tauri IPC (invoke / emit)
┌────────────────────────────▼────────────────────────────┐
│                      Rust Backend                       │
│                                                         │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────┐  │
│  │ Storage Module│   │ Configuration │   │ LLM Engine│  │
│  │ (rusqlite DB) │   │ (serde_yaml)  │   │ Manager   │  │
│  └───────────────┘   └───────────────┘   └─────┬─────┘  │
└────────────────────────────────────────────────┼────────┘
                                                 │ HTTPS SSE / REST
                                   ┌─────────────┼─────────────┐
                                   ▼             ▼             ▼
                                Gemini        OpenAI        Claude
```

---

## 1. Backend Architecture (Rust)

Location: `src-tauri/src/`

### Modules Summary

- **`main.rs`**: Entrypoint for Tauri runtime. Configures global shortcuts, single instance handler, macOS accessory mode (hides app from Dock), system tray, managed application state (`AppState`), and handles IPC commands.
- **`config.rs`**: Manages `AppConfig` and `EngineConfig` structs, reading/writing `~/.promptly/config.yaml`. Handles API key resolution logic (including `env:VAR_NAME` resolution and fallback defaults).
- **`storage.rs`**: SQLite connection & schema initialization at `~/.promptly/prompts.db`. Manages CRUD operations for prompt recipes and run history execution records.
- **`prompt.rs`**: Prompt builder utility that parses `{{variable}}` placeholders and appends optional parameters (`tone`, `length`, `complexity`).
- **`models.rs`**: Core data structures (`Prompt`, `RunRecord`, `Message`, `LlmRequest`, `LlmChunk`).
- **`llm/`**: Modular LLM provider integrations:
  - `llm/gemini.rs`: Google Gemini API streaming client (`v1beta` models/streamGenerateContent).
  - `llm/openai.rs`: OpenAI Chat Completions streaming client (`v1/chat/completions`).
  - `llm/claude.rs`: Anthropic Messages API streaming client (`v1/messages`).

---

## 2. Frontend Architecture (Web UI)

Location: `ui/`

### Structure
- **`index.html`**: Single Page Application structure containing the sidebar, recipe inputs, response panel, follow-up chat, run history modal, settings tabs, and modal dialogues.
- **`style.css`**: Pure CSS layout utilizing CSS variables for theme swatches (*Claude*, *Forest*, *Ocean*, *Lavender*), dark mode support, and glassmorphism styling.
- **`main.js`**: Vanilla JS application logic. Listens for Tauri window events (`selection`, `chunk`, `error`, `done`) and handles state updates for recipes, engines, hotkeys, and history.

---

## 3. Database Schema (SQLite)

Database path: `~/.promptly/prompts.db`

### `prompts` table
```sql
CREATE TABLE IF NOT EXISTS prompts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    engine TEXT NOT NULL,
    template TEXT NOT NULL,
    variables TEXT,      -- JSON array of extracted {{variable}} names
    params TEXT,         -- JSON array of enabled parameter controls
    icon TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

### `history` table
```sql
CREATE TABLE IF NOT EXISTS history (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    turn_index INTEGER NOT NULL DEFAULT 0,
    prompt_id TEXT NOT NULL,
    engine TEXT NOT NULL,
    input_text TEXT,
    final_prompt TEXT NOT NULL,
    response TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at DATETIME NOT NULL
);
```

---

## 4. Configuration Schema (YAML)

Config path: `~/.promptly/config.yaml`

```yaml
hotkey: "Alt+Space"

engines:
  gemini-flash:
    provider: "gemini"
    name: "Gemini Flash"
    api_key: "env:GEMINI_API_KEY"
    model: "gemini-2.5-flash-lite"
    timeout_secs: 120

ui:
  output: "stdout"
```

---

## 5. Tauri IPC Commands

| Command | Signature / Purpose |
|---|---|
| `run_recipe` | `(recipe_id, selection, engine, tone, length, complexity, session_id)` → Streams response |
| `send_chat` | `(messages_json, engine, session_id, turn_index, prompt_id)` → Multi-turn follow-up stream |
| `list_recipes` | `()` → Returns JSON list of recipes sorted by `sort_order` |
| `save_recipe` | `(id, name, description, template, engine, params, icon)` → Creates recipe |
| `update_recipe` | `(id, name, description, template, engine, params, icon)` → Updates recipe |
| `delete_recipe` | `(id)` → Deletes recipe |
| `reorder_recipes` | `(ids)` → Updates recipe sort order |
| `get_engine_configs` | `()` → Returns JSON list of configured engines |
| `save_engine_config` | `(engine, provider, api_key, model, name)` → Upserts engine config |
| `delete_engine_config` | `(engine)` → Deletes engine config |
| `ping_engine` | `(engine)` → Tests API key & quota connectivity |
| `list_history` | `()` → Returns JSON array of latest 50 run records |
| `clear_history` | `()` → Clears all run history |
| `get_hotkey_cmd` | `()` → Gets active global trigger hotkey |
| `set_hotkey` | `(hotkey)` → Updates global shortcut live without app restart |
| `get_clipboard` | `()` → Reads OS clipboard content |
| `hide_window` | `()` → Hides the overlay window |
| `export_data` | `()` → Returns JSON string of all recipes for backup |
| `import_data` | `(json_content)` → Imports recipes from JSON content |

---

## 6. Testing & Quality Assurance

Rust unit tests cover configuration resolution, template variable parsing, parameter injection, database CRUD queries, and LLM request payload serialization.

Run unit tests via:
```bash
make test
```
