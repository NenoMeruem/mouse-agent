# Prompt Agent — Development Plan

> Tổng hợp từ session thiết kế: phân tích codebase, UI/UX, Tauri integration, SQLite migration, distribution.

---

## Tổng quan

**Prompt Agent** là CLI tool + native overlay app cho phép người dùng định nghĩa "recipe" (prompt template) và chạy chúng với clipboard content, streaming kết quả realtime.

**Stack:**
- Backend: Go (CLI) — storage SQLite, LLM streaming
- UI: Tauri v2 (Rust shell) + WebView HTML/CSS
- Distribution: GoReleaser → Homebrew tap + AUR + GitHub Releases

---

## Phase 1 — Go CLI: fix bugs + SQLite migration ✅ DONE

### 1.1 SQLite storage ✅
- `internal/storage/sqlite.go` — `PromptStore` interface
- `internal/storage/history_store.go` — `HistoryStore` interface
- `internal/storage/migrate.go` — auto-migrate từ `prompts.json`
- DB path: `~/.prompt-agent/prompts.db`

### 1.2 PromptStore: Update() + Search() ✅
### 1.3 HistoryStore + history commands ✅

CLI commands hoạt động:
```bash
./prompt-agent history [--limit N] [--prompt <id>]
./prompt-agent history show <id>
./prompt-agent history search <query>
./prompt-agent history clear --before 30d / --all
```

### 1.4 Bugs đã fix ✅

| # | File | Bug | Trạng thái |
|---|---|---|---|
| 1 | `internal/cli/init.go` | Template syntax `{{.topic}}` sai | ✅ Đổi thành `{{topic}}` |
| 2 | `internal/config/config.go` | `env:OPENAI_API_KEY` không resolve | ✅ Thêm `resolveAPIKey()` |
| 3 | `internal/llm/gemini/client.go` | Fake streaming bằng 100-char chunks | ✅ Dùng `GenerateContentStream()` |
| 4 | `internal/cli/run_handler.go` | PromptEditor bắt buộc mỗi run | ✅ Chỉ show khi `--edit` flag |
| 5 | `internal/storage/json.go` | 3 prompt hardcode, không lưu file | ✅ Replaced bằng SQLite |

### 1.5 UX fixes ✅
- `--no-select` đọc từ stdin (pipe từ Tauri)
- Tee stdout để capture response → lưu vào history
- `--output json` flag cho list commands

---

## Phase 2 — Recipe system ✅ DONE

### 2.1 Prompt model mở rộng ✅
```go
type Prompt struct {
    ID, Name, Description, Engine, Template, Icon string
    Variables  []string
    Params     []string   // ["tone","length","complexity"]
    SortOrder  int
    CreatedAt, UpdatedAt time.Time
}
```

### 2.2 Param injector ✅
`internal/prompt/param_injector.go` — append suffix vào prompt dựa trên `--tone`, `--length`, `--complexity`.

### 2.3 Default recipes ✅
6 recipes seeded khi `init`: `explain`, `summarize`, `rephrase`, `fix-code`, `translate-vi`, `review-pr`.

### 2.4 Prompt reorder ✅
`prompt reorder <id1> <id2> ...` — set `sort_order` cho drag-drop trong Tauri UI.

---

## Phase 3 — Tauri overlay UI ✅ DONE

### Cấu trúc
```
src-tauri/src/main.rs        ← Rust: tray, hotkey, window, sidecar commands
ui/index.html + main.js + style.css  ← WebView UI
```

### Tauri commands đã implement

| Command | Purpose |
|---|---|
| `run_recipe` | Spawn sidecar, pipe stdin, stream stdout → `chunk` events |
| `list_recipes` | `prompt list --output json` |
| `save_recipe` | `prompt add ...` |
| `update_recipe` | `prompt update <id> ...` |
| `delete_recipe` | `prompt delete <id> --yes` |
| `reorder_recipes` | `prompt reorder <id1> <id2> ...` |
| `get_engine_configs` | `config get-engines` |
| `save_engine_config` | `config set-engine <id> --api-key --model --name` |
| `delete_engine_config` | `config delete-engine <id>` |
| `ping_engine` | `config ping-engine <id>` |
| `list_history` | `history --limit 50 --output-json` |
| `clear_history` | `history clear --all` |
| `hide_window` | Hide overlay window |
| `get_hotkey` | `config get-hotkey` — đọc hotkey hiện tại từ `config.yaml` |
| `set_hotkey` | `config set-hotkey <hk>` + unregister/re-register ngay lập tức |

### Build
```bash
make build-sidecar   # Go binary → src-tauri/binaries/prompt-agent-cli-<target-triple>
make dev-tauri       # sidecar + cargo tauri dev
make release         # .app + .dmg + Windows .exe
```

---

## Phase 4 — Distribution (đang thực hiện)

### 4.1 Cross-platform CLI build ✅

```bash
make build-windows   # dist/prompt-agent-windows-amd64.exe + arm64.exe
make build-all       # macOS + Windows + Linux → dist/
```

> macOS: CGO=1 (bắt buộc cho `golang.design/x/hotkey` Carbon API)
> Windows/Linux: CGO=0 (pure Go, cross-compile từ bất kỳ host nào)

### 4.2 `make release` ✅
Chạy `build-sidecar` + `build-windows` + `cargo tauri build` trong một lệnh.

Output:
```
src-tauri/target/release/bundle/macos/Prompt Agent.app
src-tauri/target/release/bundle/dmg/Prompt Agent_*.dmg
dist/prompt-agent-windows-amd64.exe
dist/prompt-agent-windows-arm64.exe
```

### 4.3 GoReleaser (TODO)

```yaml
# .goreleaser.yaml
project_name: prompt-agent
version: 2

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - main: ./cmd/prompt-agent
    binary: prompt-agent
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w
      - -X main.version={{.Version}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: checksums.txt

brews:
  - name: prompt-agent
    repository:
      owner: "{{ .Env.GITHUB_USERNAME }}"
      name: homebrew-prompt-agent
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/{{ .Env.GITHUB_USERNAME }}/prompt-agent"
    description: "AI prompt runner with recipe system"
    license: MIT
    install: |
      bin.install "prompt-agent"

aurs:
  - name: prompt-agent-bin
    private_key: "{{ .Env.AUR_SSH_PRIVATE_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/prompt-agent-bin.git"
    package: |-
      install -Dm755 "./prompt-agent" "${pkgdir}/usr/bin/prompt-agent"
```

### 4.4 GitHub Actions (TODO)

```yaml
# .github/workflows/release.yml
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: go test ./...
      - uses: goreleaser/goreleaser-action@v6
        with: { args: release --clean }
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
          AUR_SSH_PRIVATE_KEY: ${{ secrets.AUR_SSH_PRIVATE_KEY }}
          GITHUB_USERNAME: ${{ github.repository_owner }}

  tauri-release:
    strategy:
      matrix:
        os: [macos-latest, ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - run: make build-sidecar
      - uses: tauri-apps/tauri-action@v0
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          APPLE_CERTIFICATE: ${{ secrets.APPLE_CERTIFICATE }}
          APPLE_CERTIFICATE_PASSWORD: ${{ secrets.APPLE_CERTIFICATE_PASSWORD }}
          APPLE_ID: ${{ secrets.APPLE_ID }}
```

---

## Phase 5 — Polish (ongoing)

### Đã hoàn thành
- ✅ File logging (`~/.prompt-agent/prompt-agent.log`) — `internal/logger/` dùng `log/slog`
- ✅ Engine display name — `name` field trong `EngineConfig`, `GetEngineName()` với built-in fallback
- ✅ Windows CLI build — cross-compile từ macOS, `CGO_ENABLED=0`
- ✅ Settings UI 3 tabs — **Engines** / **Hotkeys** / **About**
- ✅ Tauri hotkey config từ UI — key recorder, lưu vào `config.yaml` (`app.hotkey`), apply ngay không cần restart
  - CLI: `config get-hotkey` / `config set-hotkey <hotkey>`
  - Tauri: `get_hotkey` / `set_hotkey` commands
  - Startup: register default `Alt+Space` → async re-register từ config nếu khác

### Còn lại
- History panel trong Tauri UI (tab hoặc slide-in)
- Recipe import/export JSON (share recipe packs)
- Winget manifest cho Windows (cần code-signing cert)
- Shell completions bundle vào Homebrew formula

---

## Conventions

- Error messages: bắt đầu bằng `❌`
- `config.GetEngineName(engine)` cho display name — không hardcode
- `config.GetEngineModel(engine)` cho model name — không hardcode
- `--raw` flag: suppress tất cả decorators (cho Tauri/piped use)
- `--output json` flag: trên list commands cho Tauri consumption
- Platform-specific code dùng build tags (`_darwin`, `_windows`, `_linux`)
- macOS builds: CGO=1 (bắt buộc). Windows/Linux: CGO=0

## File layout `~/.prompt-agent/`

| File | Mô tả |
|---|---|
| `config.yaml` | Engine configs, `app.hotkey`, UI, selection |
| `prompts.db` | SQLite — bảng `prompts` + `run_history` |
| `prompt-agent.log` | Structured log (slog text format, append) |
| `prompts.json.bak` | Backup sau khi migrate từ JSON (nếu có) |

### `config.yaml` structure

```yaml
app:
  hotkey: "Alt+Space"          # Tauri overlay trigger — configurable from Settings UI

engines:
  gemini:
    name: "Gemini Flash"       # display name (optional, fallback: "Google Gemini")
    api_key: env:GEMINI_API_KEY
    model: gemini-2.5-flash-lite
    timeout: 120s

ui:
  output: stdout               # stdout | tui

selection:
  provider: auto               # auto | macos | linux | windows
  trim_whitespace: true
  fail_on_empty_selection: false
```
