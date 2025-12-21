# Prompt Agent

A CLI tool to build and run AI prompts fast.

## Phase 1 - CLI Skeleton & Config ✅

Phase 1 features:
- ✅ CLI with Cobra framework
- ✅ Config management with Viper (YAML + Environment variables)
- ✅ `version` command
- ✅ `init` command to create default config
- ✅ `prompt` command stub for future subcommands
- ✅ Graceful config loading (works without config initially)

## Phase 2 - Prompt System ✅

Phase 2 features:
- ✅ Prompt model with ID, Name, Engine, Template, Variables
- ✅ SQLite storage for prompts at `~/.prompt-agent/prompts.db`
- ✅ `prompt add` - Add new prompts interactively
- ✅ `prompt list` - List all prompts
- ✅ `prompt show <id>` - View prompt details
- ✅ `prompt delete <id>` - Delete prompts
- ✅ Prompt builder with variable substitution ({{variable}})
- ✅ Automatic variable extraction from templates

## Phase 3 - Selection & Execution ✅

Phase 3 features:
- ✅ Clipboard selection via `pbpaste` (macOS), `xclip` (Linux), `Get-Clipboard` (Windows)
- ✅ `run <prompt-id>` - Execute prompt with clipboard content
- ✅ Environment variable support for additional variables (`PROMPT_*`)
- ✅ Interactive variable input for missing parameters
- ✅ Raw output mode (`--raw`) for integration with other tools
- ✅ Dry-run mode (`--dry-run`) for preview
- ✅ `prompt list` - Display all prompts in table format
- ✅ `prompt show <id>` - Show detailed prompt information
- ✅ `prompt delete <id>` - Delete prompts with confirmation
- ✅ Variable extraction from templates using regex (`{{variable}}`)
- ✅ Prompt builder - Merge template with data
- ✅ Unit tests for builder and variable extraction

## Phase 4 - Trigger System ✅

Phase 4 features:
- ✅ Trigger interface for extensible trigger system
- ✅ Hotkey trigger support (MVP via external daemon)
- ✅ Mouse trigger interface design (Phase 5+)
- ✅ Trigger Manager for multi-trigger orchestration
- ✅ Daemon command for running triggers in background
- ✅ Configuration support for hotkey mappings
- ✅ macOS skhd integration guide
- ✅ Linux sxhkd integration guide

## Phase 5 - Selection Text & Run Pipeline ✅

Phase 5 features:
- ✅ Selection Manager for OS-specific clipboard access
- ✅ macOS clipboard provider (pbpaste) fully working
- ✅ Linux/Windows provider stubs with clear interfaces
- ✅ Enhanced run command with complete pipeline
- ✅ Variable validation before prompt building
- ✅ Environment variable support (`PROMPT_*`)
- ✅ Interactive variable input fallback
- ✅ Formatted and raw output modes
- ✅ Graceful error handling with user-friendly messages
- ✅ Output renderer abstraction for future extensibility
- ✅ `--no-select` flag for default-only execution
- ✅ End-to-end trigger → selection → run pipeline

## Installation

### Build from source

```bash
go build -o prompt-agent ./cmd/prompt-agent
```

## Usage

### Initialize configuration

```bash
./prompt-agent init
```

This creates `~/.prompt-agent/config.yaml` with default settings.

### Manage Prompts

#### Add a new prompt

```bash
./prompt-agent prompt add
```

Interactive prompts:
- ID: Unique identifier (e.g., `explain_code`)
- Name: Display name
- Description: Optional description
- Engine: LLM engine (default: `openai`)
- Template: Prompt template with variables like `{{selection}}`

Variables are automatically extracted from template.

#### List all prompts

```bash
./prompt-agent prompt list
```

Output:
```
ID           ENGINE  NAME
-            -       -
code_review  openai  Code Review
explain_code openai  Explain Code
```

#### Show prompt details

```bash
./prompt-agent prompt show explain_code
```

#### Delete a prompt

```bash
./prompt-agent prompt delete explain_code
```

Requires confirmation before deletion.

### Run Prompts (Phase 3)

Run a prompt with clipboard content:

```bash
./prompt-agent run explain_code
```

The `run` command:
1. Gets your current clipboard content
2. Fills in the template with clipboard as `{{selection}}`
3. Prompts for any additional required variables
4. Shows the final prompt

#### Environment Variables

Provide variables via environment (no interactive prompt):

```bash
PROMPT_FILENAME="main.go" PROMPT_LANGUAGE="Go" \
  ./prompt-agent run code_review
```

Pattern: `PROMPT_<VARIABLE_NAME>`

#### Output Modes

Raw output (no formatting):

```bash
./prompt-agent run explain_code --raw
```

Preview mode (dry-run):

```bash
./prompt-agent run explain_code --dry-run
```

### Hotkey Triggers (Phase 4)

Configure hotkeys to run prompts without opening terminal:

**For macOS (recommended):**

1. Install skhd: `brew install skhd`
2. Add to `~/.skhdrc`:
   ```
   alt + space : prompt-agent run explain_code
   ```
3. Start: `brew services start skhd`

**For Linux (sxhkd):**

1. Install: `sudo apt-get install sxhkd`
2. Add to `~/.config/sxhkd/sxhkdrc`
3. Start: `sxhkd &`

See [PHASE4_TRIGGER_SETUP.md](./PHASE4_TRIGGER_SETUP.md) for detailed setup guide.

### Check version

```bash
./prompt-agent version
````

### View available commands

```bash
./prompt-agent --help
```

## Configuration

Config file location: `~/.prompt-agent/config.yaml`

Environment prefix: `PROMPT_AGENT_`

Example config:
```yaml
engines:
  openai:
    api_key: ""
    model: "gpt-4-mini"
ui:
  output: "stdout"
triggers:
  enabled: true
  default_prompt: explain_code
  hotkeys:
    alt+space: explain_code
    alt+shift+space: code_review
```

## Database

Prompts are stored in SQLite at: `~/.prompt-agent/prompts.db`

Schema:
```sql
CREATE TABLE prompts (
  id TEXT PRIMARY KEY,
  name TEXT,
  description TEXT,
  engine TEXT,
  template TEXT,
  variables TEXT,  -- JSON array
  created_at DATETIME,
  updated_at DATETIME
);
```

## Project Structure

```
prompt-agent/
├─ cmd/
│  └─ prompt-agent/
│     └─ main.go              # Entry point
├─ internal/
│  ├─ app/
│  │  └─ context.go           # App context initialization
│  ├─ cli/
│  │  ├─ root.go              # Root command
│  │  ├─ version.go           # Version command
│  │  ├─ init.go              # Init command
│  │  ├─ prompt.go            # Prompt command
│  │  ├─ prompt_add.go        # Add prompt command
│  │  ├─ prompt_list.go       # List prompts command
│  │  ├─ prompt_show.go       # Show prompt details command
│  │  ├─ prompt_delete.go     # Delete prompt command
│  │  ├─ run.go               # Run prompt command
│  │  ├─ run_handler.go       # Run command logic
│  │  └─ daemon.go            # Trigger daemon command
│  ├─ config/
│  │  └─ config.go            # Config loading with Viper
│  ├─ storage/
│  │  ├─ storage.go           # Storage interface
│  │  └─ sqlite.go            # SQLite implementation
│  ├─ prompt/
│  │  ├─ builder.go           # Prompt template builder
│  │  ├─ variables.go         # Variable extraction
│  │  └─ prompt_test.go       # Unit tests
│  ├─ selection/
│  │  ├─ provider.go          # Selection interface
│  │  ├─ macos.go             # macOS pbpaste implementation
│  │  ├─ linux.go             # Linux xclip implementation
│  │  ├─ windows.go           # Windows Get-Clipboard implementation
│  │  └─ factory.go           # OS-specific provider factory
│  ├─ app/
│  │  └─ context.go           # App context management
│  └─ trigger/
│     ├─ trigger.go           # Trigger interface
│     ├─ manager.go           # Trigger manager
│     ├─ hotkey.go            # Hotkey trigger implementation
│     └─ mouse.go             # Mouse trigger design (Phase 5+)
├─ pkg/
│  └─ models/
│     └─ prompt.go            # Prompt model
├─ go.mod
├─ go.sum
├─ prompt-agent               # Compiled binary
├─ README.md
├─ PHASE4_TRIGGER_SETUP.md    # Trigger setup guide
└─ PHASE2_SUMMARY.md          # Phase 2 summary
```

## Architecture

### Storage Layer
- **Interface**: `PromptStore` interface for abstraction
- **Implementation**: SQLite with JSON serialization for variables
- **Path**: `~/.prompt-agent/prompts.db`

### Builder
- **SimpleBuilder**: Template variable substitution using `{{variable}}` syntax
- **Variable Extraction**: Regex-based detection of all variables in templates

### Selection Provider
- **Interface**: `Provider` interface for clipboard access
- **macOS**: Uses `pbpaste` command
- **Linux**: Uses `xclip` command (stub)
- **Windows**: Uses `Get-Clipboard` PowerShell (stub)
- **Factory**: OS-specific build tags for automatic provider selection

### CLI
- **Cobra**: Command framework with subcommands
- **Viper**: Configuration management
- **Interactive Input**: bufio for multi-line prompts

## Testing

### Unit Tests

Run unit tests:
```bash
go test ./internal/prompt -v
```

Tests cover:
- Variable extraction from templates
- Template building with data substitution
- Partial data handling

### Integration Testing

Phase 3 provides full end-to-end capability:

1. Create a prompt with variables
2. Copy content to clipboard
3. Run prompt with `./prompt-agent run <id>`
4. View filled template

## Next Steps (Phase 5+)

Phase 4 is complete with trigger infrastructure. Next phases:

- [ ] **Phase 5**: Mouse trigger integration (middle-click context menu)
- [ ] **Phase 5**: Improved selection (Accessibility API on macOS)
- [ ] **Phase 6**: LLM API integration (OpenAI, Gemini, Claude)
- [ ] **Phase 6**: `prompt run --send` to send to LLM with streaming
- [ ] **Phase 7**: Result formatting and syntax highlighting
- [ ] **Phase 7**: Prompt templates library / marketplace
- [ ] **Phase 8**: History tracking and prompt versioning
- [ ] **Phase 8**: Custom filters and processors

## Development Guide

### Building

```bash
go build -o prompt-agent ./cmd/prompt-agent
```

### Testing

```bash
go test ./...
```

### Running with Custom Path

```bash
./prompt-agent init  # Creates ~/.prompt-agent/
```

---

**Status**: Phase 1-4 complete ✅
- Phase 1: CLI Skeleton ✅
- Phase 2: Prompt System ✅  
- Phase 3: Selection & Execution ✅
- Phase 4: Trigger System ✅

- Phase 1: CLI Skeleton - Complete
- Phase 2: Prompt System - Complete  
- Phase 3: Selection & Execution - Complete

- ✅ Prompt CRUD operations
- ✅ SQLite storage
- ✅ Builder engine
- ✅ CLI UX polish
