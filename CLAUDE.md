# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

CLI tool + Tauri overlay app for running AI prompt recipes against clipboard/stdin content.

## Commands

```bash
# Go CLI
make build                          # build ./prompt-agent binary
make test                           # go test ./...
go test ./internal/storage/...      # run a single package's tests
go vet ./...

# Tauri (requires Rust + cargo-tauri)
make build-sidecar                  # build Go binary + copy to src-tauri/binaries/ with target triple suffix
make dev-tauri                      # build sidecar then launch Tauri dev mode
```

## Architecture

```
cmd/prompt-agent/main.go
  └─ internal/cli/           ← Cobra commands (root, run, prompt, history, init, daemon, config)
       └─ run_handler.go     ← Core run flow: load prompt → get selection → build template → LLM → history
  └─ internal/app/context.go ← AppContext (singleton): wires PromptStore, HistoryStore, Builder, SelectionManager, Config
  └─ internal/config/        ← Viper config from ~/.prompt-agent/config.yaml
  └─ internal/storage/       ← SQLite backend (modernc.org/sqlite, pure Go, no CGO)
  └─ internal/llm/           ← LLM manager + per-provider streaming clients
  └─ internal/prompt/        ← Template builder + param injector + DefaultRecipes()
  └─ internal/selection/     ← Clipboard/selection provider (platform-specific)
  └─ internal/output/        ← StdoutRenderer, TUIRenderer (Bubble Tea), RawRenderer
  └─ internal/trigger/       ← Hotkey daemon + mouse trigger (platform-specific build tags)
  └─ pkg/models/             ← Prompt + RunRecord data models

src-tauri/src/main.rs        ← Rust/Tauri shell (tray, global hotkey, WebView, sidecar commands)
ui/                          ← WebView HTML/CSS/JS (horizontal: sidebar recipe list | main panel)
```

## Run flow (`run_handler.go`)

1. Load `Prompt` from SQLite via `PromptStore.Get(id)`
2. `GetSelection()` — clipboard OR stdin when `--no-select` (detects pipe via `os.Stdin.Stat()`)
3. `collectMissingVariables()` — tries `PROMPT_<VARNAME>` env var first, then interactive prompt
4. `Builder.Build()` — replaces `{{variable}}` placeholders in template
5. `InjectParams()` — appends tone/length/complexity suffix if flags set
6. `PromptEditor` TUI — only when `--edit` flag
7. `runWithLLM()` — tees stdout stream into `responseBuilder` + renderer, saves to `HistoryStore`

## Storage

**SQLite only** — `modernc.org/sqlite` (pure Go, no CGO).
- DB: `~/.prompt-agent/prompts.db`
- Tables: `prompts` and `run_history`
- `storage/migrate.go` auto-migrates from legacy `prompts.json` on first startup (rename to `.bak`)
- `NewSQLiteStore()` runs `ALTER TABLE prompts ADD COLUMN sort_order` on open — safe to re-run on existing DBs
- `PromptStore`/`HistoryStore` interfaces in `storage/storage.go` are the canonical API

## Data model

```go
// pkg/models/prompt.go
type Prompt struct {
    ID, Name, Description, Engine, Template, Icon string
    Variables  []string   // extracted from template {{var}} placeholders
    Params     []string   // subset of ["tone","length","complexity"]
    SortOrder  int        // display order in sidebar, lower = higher
    CreatedAt, UpdatedAt time.Time
}
```

`List()` orders by `sort_order ASC, created_at DESC`. New recipes auto-get `max(sort_order)+1`.

## Template syntax

`{{variable}}` — **not** `{{.variable}}`. Reserved variable: `selection` (from clipboard or stdin).

## CLI commands reference

```
prompt add|list|show|search|update|delete|reorder|edit
history [list]|show|search|clear
config set-engine|get-engines|delete-engine|ping-engine|verify
init        ← seeds DefaultRecipes() into fresh DB
run <id>    ← main entry point
```

`prompt reorder <id1> <id2> ...` — sets `sort_order = 0, 1, 2…` for the given IDs in order.

## Config

File: `~/.prompt-agent/config.yaml`

```yaml
engines:
  gemini:
    api_key: env:GEMINI_API_KEY   # "env:" prefix resolved via resolveAPIKey()
    model: gemini-2.5-flash-lite
ui:
  output: stdout   # or "tui"
```

API keys: env vars (`OPENAI_API_KEY`, `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`) take priority. `env:VARNAME` in config is always resolved by `config.GetEngineAPIKey()`.

## LLM providers

All implement `llm.Client` in `internal/llm/types.go`:
```go
type Client interface {
    Name() string
    Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
```

Engines registered in `registerAvailableEngines()` in `run_handler.go` — only engines with a resolvable API key are registered. Gemini uses `GenerateContentStream()` (real streaming).

## Tauri integration

Go binary runs as a **sidecar** — Tauri spawns it, pipes stdin, reads stdout line-by-line.

```bash
./prompt-agent run <id> --no-select --raw   # Tauri pipes selection via stdin
./prompt-agent prompt list --output json    # recipe list for UI
./prompt-agent prompt reorder <id>...       # persist drag-drop order
```

Sidecar binary path: `src-tauri/binaries/prompt-agent-cli-<target-triple>`. `make build-sidecar` handles this.

### Tauri commands (src-tauri/src/main.rs)

| Command | Purpose |
|---|---|
| `run_recipe` | Spawn sidecar, pipe selection via stdin, stream stdout chunks as `chunk` events |
| `list_recipes` | `prompt list --output json` |
| `save_recipe` | `prompt add ...` |
| `update_recipe` | `prompt update <id> ...` |
| `delete_recipe` | `prompt delete <id> --yes` |
| `reorder_recipes` | `prompt reorder <id1> <id2> ...` |
| `get_engine_configs` | `config get-engines` |
| `save_engine_config` | `config set-engine <id> --api-key --model` |
| `delete_engine_config` | `config delete-engine <id>` |
| `ping_engine` | `config ping-engine <id>` |
| `list_history` | `history --limit 50 --output-json` |
| `clear_history` | `history clear --all` |
| `hide_window` | Hide overlay window |

## UI (ui/)

Single-page WebView app (`index.html` + `main.js` + `style.css`). Views: `input`, `output`, `form`, `settings`, `history` — switched via `showView(name)`.

- Recipe list in sidebar uses **mouse-event drag-and-drop** (not HTML5 drag API — blocked by `user-select: none` in WebView). Drag starts on `mousedown` of the `⠿` handle.
- Engine form in Settings uses `<select>` for engine ID (gemini/claude/openai only) and `<input list="...">` datalist for model with per-engine presets. Model datalist updates on engine `change` event.
- `allRecipes` is the in-memory cache; `renderRecipeList(allRecipes)` re-renders the sidebar.
- Theme accent color: green (`--accent: #15803d`). Window: `780×600px`, transparent, decoration-less, `border-radius: 16px` on both `html` and `body` for WebView corner clipping.

## Default recipes

`internal/prompt/default_recipes.go` — seeded on `prompt-agent init` when DB is empty. Currently contains the owner's 4 Vietnamese-language translation/editing recipes. Update this file when the canonical recipe set changes.

## Conventions

- Error messages to CLI users start with `❌`
- `config.GetEngineModel(engine)` for model names — never hardcode
- Platform-specific code uses build tags (see `internal/trigger/`, `internal/output/clipboard_*.go`)
- `--raw` flag suppresses all decorators — required for Tauri/piped use
- `--output json` flag on list commands for Tauri consumption

## Detailed implementation plan

@./PLAN.md
