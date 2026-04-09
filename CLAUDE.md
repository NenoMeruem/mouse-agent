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
  └─ internal/prompt/        ← Template builder + param injector
  └─ internal/selection/     ← Clipboard/selection provider (platform-specific)
  └─ internal/output/        ← StdoutRenderer, TUIRenderer (Bubble Tea), RawRenderer
  └─ internal/trigger/       ← Hotkey daemon + mouse trigger (platform-specific build tags)
  └─ pkg/models/             ← Prompt + RunRecord data models

src-tauri/                   ← Rust/Tauri shell (tray, global hotkey, WebView)
ui/                          ← WebView HTML/CSS/JS (horizontal layout: recipe list | params + output)
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
- `json.go` still exists as legacy — do not use it for new code; `PromptStore`/`HistoryStore` interfaces in `storage/storage.go` are the canonical API

## Template syntax

`{{variable}}` — **not** `{{.variable}}`. Reserved variable: `selection` (from clipboard or stdin).

## Config

File: `~/.prompt-agent/config.yaml`

```yaml
engines:
  gemini:
    api_key: env:GEMINI_API_KEY   # "env:" prefix resolved via resolveAPIKey() in config.go
    model: gemini-2.5-flash-lite
ui:
  output: stdout   # or "tui"
```

API keys: env vars (`OPENAI_API_KEY`, `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`) take priority over config file. Config file supports `env:VARNAME` indirection — always resolved by `config.GetEngineAPIKey()`.

## LLM providers

All implement `llm.Client` in `internal/llm/types.go`:
```go
type Client interface {
    Name() string
    Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
```

Engines registered in `registerAvailableEngines()` in `run_handler.go` — only engines with a resolvable API key are registered. Gemini uses `GenerateContentStream()` (real streaming, not faked).

## Tauri integration

Go binary runs as a **sidecar** — Tauri spawns it, pipes stdin, reads stdout line-by-line.

```bash
./prompt-agent run <id> --no-select --raw   # Tauri pipes selection via stdin
./prompt-agent prompt list --output json    # Tauri uses this to load recipe list
```

The sidecar binary must be placed at `src-tauri/binaries/prompt-agent-cli-<target-triple>`. `make build-sidecar` handles this automatically.

## Conventions

- Error messages displayed to CLI users start with `❌`
- `config.GetEngineModel(engine)` for model names — never hardcode
- Platform-specific code uses build tags (see `internal/trigger/`, `internal/output/clipboard_*.go`)
- `--raw` flag suppresses all decorators — required for Tauri/piped use
- `--output json` flag on list commands for Tauri consumption

## Detailed implementation plan

@./PLAN.md
