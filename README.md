# Prompt Agent

> **Your AI, one keystroke away.**

**Prompt Agent** is a blazing-fast desktop overlay that lets you run AI on anything — just copy text, press a hotkey, and get results in seconds. No browser tabs, no context switching, no wasted time.

---

### Stop switching tabs. Start getting answers.

Every day you copy text, switch to ChatGPT, paste it, wait, copy back. Over and over.

**Prompt Agent kills that workflow dead.**

Press `Alt+Space` from anywhere on your screen — a sleek overlay appears, your clipboard is already loaded, and your AI recipes are one click away. Pick a recipe, hit Run, and stream results in real time — all without leaving what you're doing.

---

### Built for people who actually use AI

- **Explain** code you just copied from Stack Overflow
- **Summarize** a wall of documentation before reading it
- **Rephrase** your email to sound more professional
- **Review** a pull request while staying in your IDE
- **Translate** on the fly — Vietnamese, English, whatever you need

---

### Your recipes, your rules

Create your own **AI recipes** — reusable prompt templates tailored to exactly how you work. Give them a name, pick your tone, length, and complexity. Store them, reorder them, share them.

Think of it as keyboard shortcuts, but for AI.

---

### Privacy-first, local-first

- Your API key, your data, your control
- Connects directly to **Gemini, OpenAI, or Claude** — no middleman
- Everything stored locally in `~/.prompt-agent/`
- No subscription. No cloud sync. No tracking.

---

### Works the way you work

| | |
|---|---|
| ⚡ **Instant overlay** | One hotkey, no loading screen |
| 📋 **Clipboard-aware** | Auto-reads what you just copied |
| 🔁 **Real-time streaming** | See responses as they generate |
| 🎛 **Customizable hotkey** | Change the trigger to whatever feels natural |
| 🗂 **Run history** | Every result saved and searchable |
| 🖥 **macOS + Linux** | Native app, zero Electron bloat |

---

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
./prompt-agent prompt reorder <id1> <id2> ...
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

### `config` — configuration

```bash
./prompt-agent config verify                        # check which engines are configured
./prompt-agent config get-engines                   # list engine configs as JSON
./prompt-agent config set-engine gemini \
    --api-key env:GEMINI_API_KEY \
    --model gemini-2.5-flash-lite \
    --name "Gemini Flash"                           # set display name
./prompt-agent config delete-engine <id>
./prompt-agent config ping-engine gemini            # test API key + quota
./prompt-agent config get-hotkey                    # print current app trigger hotkey
./prompt-agent config set-hotkey "Ctrl+Shift+Space" # change app trigger hotkey
```

### `daemon` — CLI hotkey trigger (optional)

```bash
./prompt-agent daemon          # start hotkey listener (reads config)
./prompt-agent daemon setup    # generate skhd / xbindkeys / AutoHotkey config
```

### Other

```bash
./prompt-agent version
```

## Configuration

File: `~/.prompt-agent/config.yaml`

```yaml
app:
  hotkey: "Alt+Space"                      # global shortcut to show/hide the overlay

engines:
  openai:
    name: "GPT-4o"                         # display name (optional)
    api_key: "sk-your-key-here"            # or use env: prefix (see below)
    model: "gpt-4-mini"
  gemini:
    name: "Gemini Flash"
    api_key: env:GEMINI_API_KEY            # resolved from $GEMINI_API_KEY at runtime
    model: "gemini-2.5-flash-lite"
  claude:
    name: "Claude Sonnet"
    api_key: env:ANTHROPIC_API_KEY
    model: "claude-sonnet-4-6"
ui:
  output: "stdout"                         # or "tui" for Bubble Tea renderer
selection:
  provider: "auto"                         # auto, macos, linux, windows
  fail_on_empty_selection: false
  trim_whitespace: true
```

**`app.hotkey`:** global shortcut used by the Tauri overlay to show/hide the window. Configurable via **Settings → Hotkeys** in the UI, or via CLI (`config set-hotkey`). Default: `Alt+Space`. Takes effect immediately without restarting the app.

**Engine `name` field:** controls what is shown in the response header (`🚀 Streaming from Gemini Flash...`), the TUI editor badge, and dry-run output. Falls back to built-in display names (`openai` → `OpenAI`, `gemini` → `Google Gemini`, `claude` → `Claude`) if not set.

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

## Build

```bash
make build           # current platform binary → ./prompt-agent
make build-windows   # cross-compile → dist/prompt-agent-windows-amd64.exe + arm64.exe
make build-all       # all platforms → dist/
make release         # macOS .app + .dmg (Tauri) + Windows .exe → dist/
make test            # go test ./...
make clean           # remove all build artifacts
```

> **Note:** macOS builds require CGO (`golang.design/x/hotkey` uses Carbon API). Windows and Linux are cross-compiled with `CGO_ENABLED=0` — no extra toolchain needed.

## Tauri overlay hotkey

Press the global shortcut (default `Alt+Space`) from anywhere on your system to show/hide the overlay. The app reads your clipboard automatically when it opens.

**Change the hotkey:** open the overlay → Settings → Hotkeys → click **Change** → press your new key combination. Saved to `config.yaml` and applied instantly.

Or via CLI:
```bash
./promptly config set-hotkey "Ctrl+Shift+Space"
```

> [!NOTE]
> **Linux / Wayland & Window Manager users (Arch Linux, Sway, Hyprland, etc.):**
> Standard global hotkey interceptors do not function under Wayland. However, the app includes a single-instance plugin that resolves this:
> 1. Start the Tauri app in the background (e.g. run `./src-tauri/target/debug/promptly` or the production build).
> 2. Bind your preferred hotkey at your OS or Window Manager level to execute the `promptly` binary path.
> 3. Pressing the hotkey runs `promptly` again. The new process will notify the running background process, toggling the overlay window instantly.
> 
> **Example configurations:**
> - **Hyprland (`hyprland.conf`):** `bind = ALT, space, exec, /path/to/promptly`
> - **Sway (`~/.config/sway/config`) or i3 (`~/.config/i3/config`):** `bindsym Mod1+space exec /path/to/promptly`
> - **sxhkd (`~/.config/sxhkd/sxhkdrc`):**
>   ```
>   alt + space
>       /path/to/promptly
>   ```
> *Tip: You can symlink your compiled Tauri binary to your system PATH (e.g., `sudo ln -sf $(pwd)/src-tauri/target/debug/promptly /usr/local/bin/promptly`) to make configuration cleaner.*

## CLI hotkey daemon (optional)

A separate daemon that runs recipes directly from hotkeys without opening the overlay UI.

**macOS:** requires Accessibility permission (`System Preferences → Privacy & Security → Accessibility`).

**Linux:** run `./promptly daemon setup` to generate the correct configuration. The tool will automatically detect your display server (Wayland vs X11), scan for your installed terminal emulator (Alacritty, Kitty, Konsole, GNOME Terminal, etc.) to use appropriate window holding commands, and output:
- **X11:** Writes configured hotkeys to `~/.xbindkeysrc`.
- **Wayland / Window Managers:** Outputs ready-to-copy configuration blocks for Hyprland, Sway, i3, and sxhkd.

**macOS alternative:** generate an `skhd` config with `daemon setup` (requires `brew install skhd`).

**Windows:** generate an AutoHotkey v2 script with `daemon setup` (requires [AutoHotkey](https://www.autohotkey.com/)).

## Clipboard requirements

- **macOS:** built-in `pbpaste`
- **Linux X11:** `xclip` or `xsel` (`sudo apt install xclip` or `sudo pacman -S xclip`)
- **Linux Wayland:** `wl-paste` (`sudo apt install wl-clipboard` or `sudo pacman -S wl-clipboard`)
- **Windows:** PowerShell `Get-Clipboard` (built-in)

## Storage & Logging

All data is stored in `~/.prompt-agent/`:

| File | Description |
|---|---|
| `prompts.db` | SQLite database — recipes + run history |
| `config.yaml` | Engine configs, UI settings, hotkeys |
| `prompt-agent.log` | Structured log file (errors, warnings, run traces) |

If a legacy `prompts.json` exists from an older version, it is auto-migrated on first run and renamed to `prompts.json.bak`.
