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
```

### Verify configuration

```bash
./prompt-agent config verify
```

This command checks your configuration and shows which LLM engines are available:
- Shows config file location
- Lists configured engines from `config.yaml`
- Checks environment variables for API keys
- Displays available engines and helpful setup tips

### View available commands

```bash
./prompt-agent --help
```

## Configuration

Config file location: `~/.prompt-agent/config.yaml`

Environment prefix: `PROMPT_AGENT_`

### Setting API Keys

API keys can be configured in two ways (environment variables take priority):

**Option 1: Config File** (`~/.prompt-agent/config.yaml`)
```yaml
engines:
  openai:
    api_key: "sk-your-key-here"
    model: "gpt-4-mini"
  gemini:
    api_key: "your-gemini-key-here"
    model: "gemini-2.5-flash-lite"
ui:
  output: "stdout"
```

**Option 2: Environment Variables**
```bash
export OPENAI_API_KEY="sk-your-key-here"
export GEMINI_API_KEY="your-gemini-key-here"
```

Use `./prompt-agent config verify` to verify your setup and see which engines are available.

### Full Configuration Example

```yaml
engines:
  openai:
    api_key: "sk-your-key-here"
    model: "gpt-4-mini"
  gemini:
    api_key: "your-gemini-key-here"
    model: "gemini-2.5-flash-lite"
ui:
  output: "stdout"
triggers:
  enabled: true
  default_prompt: explain_code
  hotkeys:
    alt+space: explain_code
    alt+shift+space: code_review
selection:
  provider: "auto"
  fail_on_empty_selection: false
  trim_whitespace: true
```

## Prompt storage

Prompts are persisted in a JSON file under your home directory (the path is
`~/.prompt-agent/prompts.json` by default).  The `init` command will create this
file with a sample entry so you can see the expected structure.

An example prompt record looks like:

```json
{
  "id": "example",
  "name": "Example Prompt",
  "description": "This is a sample prompt. Edit or delete it.",
  "engine": "openai",
  "template": "Write a short description of {{.topic}}.",
  "variables": ["topic"],
  "created_at": "2026-01-01T00:00:00Z",
  "updated_at": "2026-01-01T00:00:00Z"
}
```

(Internally the SDK loads an array of these objects and manages them via the
`storage.JSONStore` implementation.)

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
- **Linux**: Uses `xclip`, `xsel`, or `wl-paste` (with fallback chain)
- **Windows**: Uses `Get-Clipboard` PowerShell
- **Factory**: OS-specific build tags for automatic provider selection

#### Linux Clipboard Setup

On Linux, the app automatically tries multiple clipboard tools in this order:
1. **xclip** (X11) - Most common
2. **xsel** (X11) - Alternative
3. **wl-paste** (Wayland) - For Wayland desktop environments

**Installation:**
```bash
# For X11 systems (Ubuntu/Debian)
sudo apt-get install xclip
# or
sudo apt-get install xsel

# For Wayland systems
sudo apt-get install wl-clipboard
```

If none of these tools are installed, the app will show a helpful error message asking you to install one of them.

### CLI
- **Cobra**: Command framework with subcommands
- **Viper**: Configuration management
- **Interactive Input**: bufio for multi-line prompts