# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o prompt-agent ./cmd/prompt-agent

# Run
./prompt-agent <command> [flags]

# Test all
go test ./...

# Test a specific package
go test ./internal/storage -v
go test ./internal/prompt -v

# Static analysis
go vet ./...
```

## CLI Commands

| Command | Description |
|---|---|
| `init` | Initialize config and example prompts |
| `config verify` | Verify configuration and available LLM engines |
| `prompt add/list/show/delete` | Manage prompts |
| `run <prompt-id>` | Execute a prompt using clipboard as input |
| `daemon` | Start trigger daemon (hotkeys) |

`run` flags: `--dry-run` (preview without API call), `--raw` (skip formatting), `--no-select` (skip clipboard).

## Architecture

### Request Flow

```
CLI → root.PersistentPreRunE (app.InitializeContext + config.Load)
    → run_handler.go
        ├── Load prompt from JSONStore
        ├── GetSelection() from clipboard (OS-specific)
        ├── Collect missing {{variables}} interactively
        ├── Build final prompt via SimpleBuilder
        ├── Register LLM providers into LLM Manager
        ├── provider.Stream(ctx, request) → <-chan Chunk
        └── renderer.RenderStream(ch) → stdout or TUI
```

### Key Abstractions

**LLM Manager** (`internal/llm/`) — Plugin registry for LLM providers. Each provider implements the `Client` interface (`types.go`) with a `Stream()` method returning `<-chan Chunk`. Three providers: `openai/`, `gemini/`, `claude/`.

**AppContext** (`internal/app/context.go`) — Dependency container wired at startup: `PromptStore`, `Builder`, `SelectionManager`, `Config`. Passed through Cobra's `PersistentPreRunE`.

**Storage** (`internal/storage/`) — `PromptStore` interface backed by `JSONStore` (JSON file at `~/.prompt-agent/prompts.json`). Thread-safe via `sync.RWMutex` + atomic writes (temp file + `os.Rename`).

**Selection** (`internal/selection/`) — OS-specific clipboard abstraction. Factory in `factory.go` picks macOS (`pbpaste`), Linux (`xclip`/`xsel`/`wl-paste` fallback chain), or Windows (PowerShell).

**Output** (`internal/output/`) — `StreamRenderer` interface with `StdoutStreamRenderer` and `TUIRenderer` (Bubble Tea). Selected via `ui.output` config (`stdout` or `tui`).

**Prompt Builder** (`internal/prompt/builder.go`) — `SimpleBuilder` substitutes `{{variable}}` placeholders in templates using `strings.ReplaceAll`.

**Triggers** (`internal/trigger/`) — Event-driven daemon for hotkeys. `TriggerManager` holds a thread-safe registry of `Trigger` implementations.

### Data Model

```go
// pkg/models/prompt.go
type Prompt struct {
    ID, Name, Description, Engine, Template string
    Variables                                []string
    CreatedAt, UpdatedAt                     time.Time
}
```

## Configuration

**File:** `~/.prompt-agent/config.yaml`

```yaml
engines:
  openai:
    api_key: "sk-..."       # or env OPENAI_API_KEY
    model: "gpt-4-mini"
  gemini:
    api_key: "AIzaSy..."    # or env GEMINI_API_KEY
    model: "gemini-2.5-flash-lite"
  claude:
    api_key: "sk-ant-..."   # or env ANTHROPIC_API_KEY
    model: "claude-sonnet-4-6"
ui:
  output: "stdout"          # or "tui"
triggers:
  enabled: true
  hotkeys:
    alt+space: "explain_code"
selection:
  provider: "auto"
```

Config is managed by Viper with `PROMPT_AGENT_` env prefix. Defaults are defined as constants in `internal/config/config.go`.
