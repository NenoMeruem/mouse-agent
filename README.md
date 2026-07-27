# Promptly

> **Your AI, one keystroke away.**

**Promptly** is a blazing-fast, privacy-first desktop overlay app that lets you run AI prompts on anything — just copy text, press a hotkey, and stream results in seconds. Built natively with **Tauri v2** and **Rust**, it lives seamlessly on top of your desktop without heavy electron bloat or constant tab switching.

---

### Key Features

- ⚡ **Instant Desktop Overlay**: Press `Alt+Space` from any application — a sleek, floating UI appears instantly.
- 📋 **Clipboard-Aware**: Automatically captures clipboard text and injects it into template placeholders (`{{selection}}`).
- 🔁 **Real-Time LLM Streaming**: Stream responses chunk-by-chunk from **Google Gemini**, **OpenAI**, or **Anthropic Claude**.
- 🎛 **Dynamic Recipes & Parameters**: Create custom prompt templates with configurable parameters (`Style / Tone`, `Length`, `Complexity`).
- 💬 **Multi-Turn Chat**: Follow up on AI outputs with continuous multi-turn chat directly in the overlay.
- ⚙️ **Flexible Engine Management**: Configure custom LLM providers, model names, API keys, and environment variable fallbacks (`env:VAR_NAME`).
- 🌗 **Themes & Dark Mode**: Select between built-in themes (*Claude*, *Forest*, *Ocean*, *Lavender*) with full Dark Mode support.
- 🔔 **System Tray Integration**: Quietly runs in your system tray with quick-toggle, history viewer, and quit controls.
- 💾 **Local-First & Privacy-Focused**: Stores all data locally in SQLite (`~/.promptly/prompts.db`) and YAML config (`~/.promptly/config.yaml`). No tracking, no subscription.
- 📦 **Import & Export**: Backup or restore your prompt recipes and engine configurations via CSV or JSON.

---

## Project Structure

```
prompt-builder-agent/
├── src-tauri/             # Rust Backend (Tauri v2)
│   ├── src/
│   │   ├── main.rs        # Main entrypoint & Tauri command handlers
│   │   ├── config.rs      # YAML configuration loader & engine manager
│   │   ├── storage.rs     # SQLite database migrations & queries
│   │   ├── prompt.rs      # Template engine & parameter injection
│   │   ├── models.rs      # Data models & serialization
│   │   └── llm/           # Provider implementations (Gemini, OpenAI, Claude)
│   ├── Cargo.toml         # Rust dependencies & package metadata
│   └── tauri.conf.json    # Tauri app window & bundle configuration
├── ui/                    # Frontend WebApp (Vanilla HTML/CSS/JS)
│   ├── index.html         # Main overlay markup & settings panels
│   ├── style.css          # Glassmorphism UI, themes & layout styles
│   └── main.js            # Client UI state & Tauri IPC bridge
├── pkg/                   # Go data model references
├── Makefile               # Build & development task shortcuts
└── README.md              # Project documentation
```

---

## Requirements

- **Rust**: 1.75+ ([install via rustup](https://rustup.rs/))
- **Node.js / npm**: 18+ (for Tauri CLI tooling)
- **Tauri CLI**: `cargo install tauri-cli` or `npx @tauri-apps/cli`

---

## Getting Started

### Development Mode

Run the app with hot-reloading for both backend and frontend:

```bash
make dev
```
*(Runs `cd src-tauri && cargo tauri dev` under the hood)*

### Building for Release

Generate standalone native installers (macOS `.dmg`/`.app`, Linux `.AppImage`/`.deb`, Windows `.exe`/`.msi`):

```bash
make build
```
*(Artifacts will be placed in `src-tauri/target/release/bundle/`)*

### Quick Commands

| Command | Description |
|---|---|
| `make dev` | Run app in development mode with live reload |
| `make build` | Compile release production build & installer bundle |
| `make build-debug` | Compile fast unoptimized debug binary |
| `make test` | Execute backend Rust unit test suite |
| `make lint` | Run Clippy linter checks |
| `make fmt` | Check code formatting adherence |
| `make clean` | Clean target directory and build artifacts |

---

## Configuration

Settings are automatically saved in `~/.promptly/config.yaml`:

```yaml
hotkey: "Alt+Space"                   # Global hotkey trigger

engines:
  gemini-flash:
    provider: "gemini"
    name: "Gemini Flash"              # Header display name
    api_key: env:GEMINI_API_KEY       # Environment variable fallback
    model: "gemini-2.5-flash-lite"
  openai-main:
    provider: "openai"
    name: "GPT-4o Mini"
    api_key: "sk-your-key-here"
    model: "gpt-4o-mini"
  claude-sonnet:
    provider: "claude"
    name: "Claude Sonnet"
    api_key: env:ANTHROPIC_API_KEY
    model: "claude-sonnet-4-6"

ui:
  output: "stdout"
```

### API Key Resolution Order
1. Environment variable (`GEMINI_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`)
2. `api_key` field in `config.yaml`
3. `env:VAR_NAME` indirection in `config.yaml`

---

## Default Recipes

On first launch, Promptly automatically seeds SQLite with built-in recipes:

| ID | Recipe Name | Template & Usage | Supported Parameters |
|---|---|---|---|
| `explain` | Explain | Explains text or code clearly | `Tone`, `Length` |
| `summarize` | Summarize | Summarizes key points concisely | `Length` |
| `rephrase` | Rephrase | Rewrites text into desired tone/style | `Tone`, `Complexity` |
| `fix-code` | Fix Code | Fixes bugs and improves code quality | — |
| `translate-vi` | Translate → VI | Translates content to Vietnamese | — |
| `review-pr` | Review PR | Analyzes code changes & suggests improvements | `Complexity` |

---

## Global Hotkey & Window Managers

### Changing the Hotkey
Open **Promptly Settings → Hotkeys → Change**, press your desired key combination, and save. It applies instantly without restarting.

### Linux / Wayland & Tiling Window Managers
Standard global hotkey hooks might be blocked on Wayland (e.g. Hyprland, Sway, Swaylock). Promptly includes a single-instance listener that allows keybindings from your Window Manager to toggle the overlay seamlessly:

1. Keep Promptly running in the background.
2. Bind a key in your window manager to launch the compiled binary (e.g. `promptly`).
3. Executing the binary again signals the running process to toggle window visibility and capture updated clipboard content.

**Example configurations:**
- **Hyprland (`hyprland.conf`):** `bind = ALT, space, exec, promptly`
- **Sway (`~/.config/sway/config`):** `bindsym Mod1+space exec promptly`
- **i3 (`~/.config/i3/config`):** `bindsym Mod1+space exec promptly`

---

## System Storage & Logs

- **SQLite Database**: `~/.promptly/prompts.db` (stores recipes, variable rules, and execution run history)
- **YAML Config**: `~/.promptly/config.yaml` (engine settings, hotkey triggers, UI options)

---

## License

MIT License. Built for speed, efficiency, and personal productivity.

