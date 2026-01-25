# Prompt Agent

A CLI tool to build and run AI prompts fast.

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