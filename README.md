# Prompt Agent

A CLI tool + Tauri overlay app for running AI prompt recipes against clipboard content.

## Installation

### Build from source

Requires Go 1.24+.

```bash
git clone <repo>
cd prompt-builder-agent
go build -o prompt-agent ./cmd/prompt-agent
```

### Tauri app (optional)

Requires Rust + `cargo-tauri`.

```bash
make build-sidecar   # builds Go binary and copies it to src-tauri/binaries/
make dev-tauri       # launch Tauri in dev mode
```

## Quick start

```bash
# 1. Create config and seed default recipes
./prompt-agent init

# 2. Set your API key
export GEMINI_API_KEY="your-key-here"

# 3. Copy some text to clipboard, then run a recipe
./prompt-agent run explain
```

## Commands

### `init`

Creates `~/.prompt-agent/config.yaml` and seeds 6 default recipes into the SQLite store.

```bash
./prompt-agent init
```

### `run <recipe-id>`

Runs a recipe. By default reads selection from clipboard.

```bash
./prompt-agent run explain
./prompt-agent run summarize --length short
./prompt-agent run rephrase --tone casual --complexity simple
./prompt-agent run fix-code --engine openai
./prompt-agent run explain --dry-run        # preview prompt, no API call
./prompt-agent run explain --edit           # open editor before sending
./prompt-agent run explain --raw            # plain output (used by Tauri)
echo "some text" | ./prompt-agent run explain --no-select  # read from stdin
```

**Flags:**

| Flag | Description |
|---|---|
| `--engine` | Override recipe's engine (`gemini`, `openai`, `claude`) |
| `--tone` | `professional`, `casual`, `concise` |
| `--length` | `short`, `medium`, `long` |
| `--complexity` | `simple`, `normal`, `technical` |
| `--no-select` | Skip clipboard; read from stdin instead |
| `--raw` | Plain stdout, no formatting decorators |
| `--dry-run` | Preview final prompt without calling the API |
| `--edit` | Open prompt in TUI editor before sending |

### `prompt` — manage recipes

```bash
./prompt-agent prompt list                  # table view
./prompt-agent prompt list --output json    # JSON (used by Tauri)
./prompt-agent prompt show <id>
./prompt-agent prompt add                   # interactive mode
./prompt-agent prompt add --id my-recipe --template "Summarize: {{selection}}" --engine gemini
./prompt-agent prompt edit <id>             # opens $EDITOR
./prompt-agent prompt update <id> --name "New Name"
./prompt-agent prompt search <query>
./prompt-agent prompt delete <id>
```

### `history` — run history

```bash
./prompt-agent history                      # last 50 runs
./prompt-agent history --limit 20 --prompt explain
./prompt-agent history show <id-prefix>     # full input + prompt + response
./prompt-agent history search <query>
./prompt-agent history clear --before 30d
./prompt-agent history clear --all
```

### `daemon` — hotkey trigger

```bash
./prompt-agent daemon          # start hotkey listener (reads config)
./prompt-agent daemon setup    # generate skhd / xbindkeys config file
```

### Other

```bash
./prompt-agent version
./prompt-agent config verify   # check which engines are configured
./prompt-agent config engines  # list available engines
```

## Configuration

File: `~/.prompt-agent/config.yaml`

```yaml
engines:
  openai:
    api_key: "sk-your-key-here"      # or use env: prefix (see below)
    model: "gpt-4-mini"
  gemini:
    api_key: env:GEMINI_API_KEY      # resolved from $GEMINI_API_KEY at runtime
    model: "gemini-2.5-flash-lite"
  claude:
    api_key: env:ANTHROPIC_API_KEY
    model: "claude-sonnet-4-6"
ui:
  output: "stdout"                   # or "tui" for Bubble Tea renderer
triggers:
  enabled: true
  hotkeys:
    alt+space: explain
    alt+shift+r: review-pr
selection:
  provider: "auto"                   # auto, macos, linux, windows
  fail_on_empty_selection: false
  trim_whitespace: true
```

**API key resolution order** (highest priority first):
1. Environment variable (`OPENAI_API_KEY`, `GEMINI_API_KEY`, `ANTHROPIC_API_KEY`)
2. Config file value
3. Config file with `env:VARNAME` indirection

## Default recipes

Seeded on `init`:

| ID | Name | Supported params |
|---|---|---|
| `explain` | Explain | `tone`, `length` |
| `summarize` | Summarize | `length` |
| `rephrase` | Rephrase | `tone`, `complexity` |
| `fix-code` | Fix Code | — |
| `translate-vi` | Translate → VI | — |
| `review-pr` | Review PR | `complexity` |

## Template syntax

Templates use `{{variable}}` placeholders. `selection` is the reserved variable populated from clipboard or stdin.

```
Fix any bugs in the following code:\n\n{{selection}}
```

Variables not provided by clipboard are prompted interactively, or can be set via `PROMPT_<VARNAME>` environment variables.

## Hotkey daemon

Registers system-wide hotkeys that run a recipe against whatever text is currently in the clipboard — no terminal needed.

**macOS:** requires Accessibility permission (`System Preferences → Privacy & Security → Accessibility`).

**Linux (X11):** uses `golang.design/x/hotkey`. Alternatively, generate an `xbindkeys` config with `daemon setup`.

**macOS alternative:** generate an `skhd` config with `daemon setup` (requires `brew install skhd`).

## Clipboard requirements

- **macOS:** built-in `pbpaste`
- **Linux X11:** `xclip` or `xsel` (`sudo apt install xclip`)
- **Linux Wayland:** `wl-paste` (`sudo apt install wl-clipboard`)
- **Windows:** PowerShell `Get-Clipboard` (built-in)

## Storage

All data is stored in `~/.prompt-agent/prompts.db` (SQLite, pure Go — no CGO required). If a legacy `prompts.json` exists from an older version, it is auto-migrated on first run and renamed to `prompts.json.bak`.
