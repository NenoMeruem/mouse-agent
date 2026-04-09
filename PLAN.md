# Prompt Agent — Development Plan

> Tổng hợp từ session thiết kế: phân tích codebase, UI/UX, Tauri integration, SQLite migration, distribution.

---

## Tổng quan

**Prompt Agent** là CLI tool + native overlay app cho phép người dùng định nghĩa "recipe" (prompt template) và chạy chúng với clipboard content, streaming kết quả realtime.

**Stack:**
- Backend: Go (existing CLI) — storage SQLite, LLM streaming
- UI: Tauri v2 (Rust shell) + WebView HTML/CSS
- Distribution: GoReleaser → Homebrew tap + AUR + GitHub Releases

---

## Phase 1 — Go CLI: fix bugs + SQLite migration (1–2 tuần)

> Foundation. Phải xong trước khi làm bất cứ thứ gì khác.

### 1.1 SQLite storage — thay thế JSONStore hoàn toàn

**Tại sao:** History 500 runs vào JSON = load toàn bộ mỗi lần append. SQLite hỗ trợ LIMIT/OFFSET, FTS5, concurrent read tốt hơn.

**Library:** `modernc.org/sqlite` (pure Go, không CGO → cross-compile bình thường).

**Schema:**

```sql
CREATE TABLE prompts (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    engine      TEXT NOT NULL,
    template    TEXT NOT NULL,
    variables   TEXT DEFAULT '[]',   -- JSON array
    params      TEXT DEFAULT '[]',   -- JSON array: ["tone","length"]
    icon        TEXT DEFAULT '',
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);

CREATE TABLE run_history (
    id           TEXT PRIMARY KEY,
    prompt_id    TEXT NOT NULL,
    engine       TEXT NOT NULL,
    input_text   TEXT DEFAULT '',
    final_prompt TEXT NOT NULL,
    response     TEXT DEFAULT '',
    duration_ms  INTEGER DEFAULT 0,
    error        TEXT DEFAULT '',
    created_at   DATETIME NOT NULL,
    FOREIGN KEY (prompt_id) REFERENCES prompts(id)
);

CREATE INDEX idx_history_prompt_id ON run_history(prompt_id);
CREATE INDEX idx_history_created_at ON run_history(created_at DESC);
```

**Files cần tạo/sửa:**
- `internal/storage/sqlite.go` — implement `PromptStore` interface (thay json.go)
- `internal/storage/history_store.go` — `HistoryStore` interface + SQLite impl
- `internal/storage/migrate.go` — tự động migrate từ `prompts.json` cũ
- Xóa `internal/storage/json.go` sau khi migrate xong
- `internal/app/context.go` — inject `HistoryStore` vào `AppContext`

**Migration logic (`migrate.go`):**
```go
// Khi khởi động:
// 1. Check ~/.prompt-agent/prompts.json tồn tại
// 2. Nếu có và prompts.db chưa có data → đọc JSON, INSERT vào SQLite
// 3. Rename prompts.json → prompts.json.bak
// 4. Chỉ chạy một lần
```

### 1.2 Thêm Update() và Search() vào PromptStore

```go
type PromptStore interface {
    Create(prompt *models.Prompt) error
    List() ([]models.Prompt, error)
    Get(id string) (*models.Prompt, error)
    Update(prompt *models.Prompt) error   // MỚI
    Delete(id string) error
    Search(query string) ([]models.Prompt, error) // MỚI — SQLite LIKE
}
```

**CLI commands mới:**
- `prompt edit <id>` — mở $EDITOR, save → Update()
- `prompt search <query>` — tìm trong name + template

### 1.3 HistoryStore + commands

```go
type HistoryStore interface {
    Append(record *models.RunRecord) error
    List(limit int, promptID string) ([]models.RunRecord, error)
    Search(query string) ([]models.RunRecord, error) // SQLite FTS5
    Clear(before time.Time) error
}
```

**CLI commands:**
```bash
./prompt-agent history              # list 20 gần nhất
./prompt-agent history --limit 50 --prompt explain
./prompt-agent history show <id>    # full response, dùng pager
./prompt-agent history search "pipeline failed"
./prompt-agent history clear --before 30d
```

### 1.4 Fix 3 bugs quan trọng

**Bug 1 — Template syntax:**
- `init.go` tạo sample với `{{.topic}}` nhưng `builder.go` dùng `{{topic}}`
- Fix: đổi tất cả sample template trong `init.go` sang `{{topic}}`

**Bug 2 — Config env: prefix không được resolve:**
```go
// internal/config/config.go — thêm vào Load() hoặc GetEngineAPIKey()
func resolveAPIKey(raw string) string {
    if strings.HasPrefix(raw, "env:") {
        return os.Getenv(strings.TrimPrefix(raw, "env:"))
    }
    return raw
}
```

**Bug 3 — Gemini fake streaming:**
```go
// internal/llm/gemini/client.go
// Đổi từ GenerateContent() sang GenerateContentStream()
iter := model.GenerateContentStream(ctx, genai.Text(req.Prompt))
for {
    resp, err := iter.Next()
    if err == iterator.Done { break }
    // emit từng chunk thật
}
```

### 1.5 UX fixes

**Bỏ PromptEditor bắt buộc:**
```go
// run_handler.go — đảo logic
if editFlag { // chỉ mở khi --edit
    editedPrompt, confirmed, err := PromptEditor(finalPrompt, promptDef.Engine)
    ...
}
// Mặc định: gửi thẳng
```

**Stdin read thay clipboard khi `--no-select`:**
```go
// Chuẩn bị cho Tauri pipe selection text vào
if noSelection {
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) == 0 {
        bytes, _ := io.ReadAll(os.Stdin)
        data["selection"] = strings.TrimSpace(string(bytes))
    }
}
```

**Tee stdout để capture response:**
```go
// runWithLLM() — thu thập response song song với stream
var responseBuilder strings.Builder
startTime := time.Now()

teeCh := make(chan llm.Chunk, 10)
go func() {
    for chunk := range ch {
        if chunk.Text != "" {
            responseBuilder.WriteString(chunk.Text)
        }
        teeCh <- chunk
    }
    close(teeCh)
}()

renderer.RenderStream(teeCh)

// Lưu vào history sau khi xong
historyStore.Append(&models.RunRecord{
    ID:          uuid.New().String(),
    PromptID:    promptDef.ID,
    Engine:      engine,
    InputText:   data["selection"],
    FinalPrompt: finalPrompt,
    Response:    responseBuilder.String(),
    DurationMs:  time.Since(startTime).Milliseconds(),
    CreatedAt:   time.Now(),
})
```

**Thêm `--output json` flag cho list commands (chuẩn bị cho Tauri):**
```bash
./prompt-agent prompt list --output json  # Tauri sẽ dùng flag này
```

---

## Phase 2 — Recipe system (2–3 tuần)

### 2.1 Extend Prompt model

```go
// pkg/models/prompt.go — thêm fields mới (schema đã có sẵn từ Phase 1)
type Prompt struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Engine      string    `json:"engine"`
    Template    string    `json:"template"`
    Variables   []string  `json:"variables"`
    Params      []string  `json:"params,omitempty"`  // MỚI: ["tone","length","complexity"]
    Icon        string    `json:"icon,omitempty"`    // MỚI: tên icon hoặc emoji
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 2.2 Param injector

```go
// internal/prompt/param_injector.go
var paramSuffixes = map[string]map[string]string{
    "tone": {
        "professional": "Reply in a professional tone.",
        "casual":       "Reply in a casual, friendly tone.",
        "concise":      "Be very concise and direct.",
    },
    "length": {
        "short":  "Keep the response under 3 sentences.",
        "medium": "Aim for 1-2 paragraphs.",
        "long":   "Provide a detailed, comprehensive response.",
    },
    "complexity": {
        "simple":    "Explain like I'm 5 years old. Use simple words.",
        "normal":    "",
        "technical": "Use technical terms. Assume expert-level knowledge.",
    },
}

func InjectParams(prompt string, paramValues map[string]string) string {
    var suffixes []string
    for param, value := range paramValues {
        if m, ok := paramSuffixes[param]; ok {
            if s, ok := m[value]; ok && s != "" {
                suffixes = append(suffixes, s)
            }
        }
    }
    if len(suffixes) == 0 {
        return prompt
    }
    return prompt + "\n\n" + strings.Join(suffixes, " ")
}
```

**Flags thêm vào `run` command:**
```bash
./prompt-agent run explain --tone casual --length short --complexity simple
```

### 2.3 Default recipes (thay 3 prompt hardcode hiện tại)

Bootstrap 6 recipes khi `init`:

| ID | Name | Params | Engine |
|---|---|---|---|
| `explain` | Explain | tone, length | gemini |
| `summarize` | Summarize | length | gemini |
| `rephrase` | Rephrase | tone, complexity | gemini |
| `fix-code` | Fix code | — | gemini |
| `translate-vi` | Translate → VI | — | gemini |
| `review-pr` | Review PR | complexity | gemini |

---

## Phase 3 — Tauri overlay UI (2–3 tuần)

### 3.1 Project structure

```
prompt-agent/
├── src-tauri/
│   ├── src/
│   │   └── main.rs          ← Rust: tray, hotkey, window, sidecar commands
│   ├── binaries/
│   │   └── prompt-agent-{target-triple}  ← Go binary (build script tạo)
│   └── tauri.conf.json
├── ui/
│   ├── index.html           ← WebView UI (horizontal layout)
│   ├── style.css
│   └── main.js
└── cmd/prompt-agent/        ← Go backend (giữ nguyên)
```

### 3.2 tauri.conf.json

```json
{
  "productName": "Prompt Agent",
  "bundle": {
    "externalBin": ["binaries/prompt-agent"]
  },
  "app": {
    "windows": [{
      "label": "overlay",
      "width": 700,
      "height": 260,
      "resizable": false,
      "decorations": false,
      "alwaysOnTop": true,
      "visible": false,
      "transparent": true,
      "skipTaskbar": true
    }]
  },
  "plugins": {
    "shell": {
      "scope": [{
        "name": "binaries/prompt-agent",
        "sidecar": true,
        "args": true
      }]
    },
    "global-shortcut": {},
    "clipboard-manager": {}
  }
}
```

### 3.3 main.rs — core logic

```rust
// Tauri command: spawn Go sidecar, pipe selection, stream stdout
#[tauri::command]
async fn run_recipe(
    app: tauri::AppHandle,
    recipe_id: String,
    selection: String,
    engine: String,
    tone: String,
    length: String,
) -> Result<(), String> {
    let mut args = vec!["run", &recipe_id, "--no-select", "--raw"];
    if !engine.is_empty() { args.extend(["--engine", &engine]); }
    if !tone.is_empty()   { args.extend(["--tone", &tone]); }
    if !length.is_empty() { args.extend(["--length", &length]); }

    let cmd = app.shell().sidecar("prompt-agent")
        .unwrap().args(&args);

    let (mut rx, mut child) = cmd.spawn().map_err(|e| e.to_string())?;

    // Pipe selection text qua stdin
    child.write(selection.as_bytes()).ok();
    drop(child); // close stdin

    tauri::async_runtime::spawn(async move {
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    app.emit("chunk", String::from_utf8_lossy(&line).to_string()).ok();
                }
                CommandEvent::Terminated(_) => {
                    app.emit("done", ()).ok();
                    break;
                }
                _ => {}
            }
        }
    });
    Ok(())
}

// Tauri command: list recipes từ Go binary
#[tauri::command]
async fn list_recipes(app: tauri::AppHandle) -> Result<String, String> {
    let output = app.shell()
        .sidecar("prompt-agent").unwrap()
        .args(["prompt", "list", "--output", "json"])
        .output().await.map_err(|e| e.to_string())?;
    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_global_shortcut::Builder::new()
            .with_handler(|app, _shortcut, event| {
                if event.state == ShortcutState::Pressed {
                    let w = app.get_webview_window("overlay").unwrap();
                    if w.is_visible().unwrap() {
                        w.hide().unwrap();
                    } else {
                        // Đọc clipboard, emit về UI
                        let clip = app.clipboard().read_text().unwrap_or_default();
                        w.emit("selection", clip).ok();
                        w.show().unwrap();
                        w.set_focus().unwrap();
                    }
                }
            })
            .build())
        .plugin(tauri_plugin_positioner::init())
        .setup(|app| {
            app.global_shortcut().register("Alt+Space")?;

            // System tray
            TrayIconBuilder::new()
                .tooltip("Prompt Agent")
                .on_tray_icon_event(|tray, event| {
                    if let TrayIconEvent::Click { .. } = event {
                        let w = tray.app_handle()
                            .get_webview_window("overlay").unwrap();
                        w.show().unwrap();
                        w.set_focus().unwrap();
                    }
                })
                .build(app)?;

            // macOS: ẩn dock icon
            #[cfg(target_os = "macos")]
            app.set_activation_policy(tauri::ActivationPolicy::Accessory);

            Ok(())
        })
        // Ẩn window khi mất focus
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::Focused(false) = event {
                if window.label() == "overlay" {
                    window.hide().unwrap();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![run_recipe, list_recipes])
        .run(tauri::generate_context!())
        .unwrap();
}
```

### 3.4 Build script — rename binary

```makefile
# Makefile
build-sidecar:
	go build -o prompt-agent ./cmd/prompt-agent
	TARGET=$$(rustc -vV | grep host | cut -d' ' -f2); \
	cp prompt-agent src-tauri/binaries/prompt-agent-$$TARGET
	@echo "Sidecar built for $$TARGET"

dev-tauri: build-sidecar
	cd src-tauri && cargo tauri dev
```

### 3.5 UI — horizontal layout (WebView)

Layout: **recipe list trái (180px) | params + streaming output phải (flex)**

```javascript
// ui/main.js — core logic
import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';

// Load recipes khi khởi động
const recipes = await invoke('list_recipes');
renderRecipeList(JSON.parse(recipes));

// Nhận selection từ Rust khi window show
await listen('selection', ({ payload }) => {
    selectionEl.textContent = payload;
    currentSelection = payload;
});

// Stream chunks về UI
await listen('chunk', ({ payload }) => {
    outputEl.textContent += payload;
});

await listen('done', () => {
    runBtn.disabled = false;
    copyBtn.style.display = 'block';
});

// Run button
runBtn.onclick = async () => {
    outputEl.textContent = '';
    runBtn.disabled = true;
    copyBtn.style.display = 'none';

    await invoke('run_recipe', {
        recipeId: activeRecipe,
        selection: currentSelection,
        engine: activeEngine,
        tone: activeTone,
        length: activeLength,
    });
};

// Hotkeys
document.addEventListener('keydown', e => {
    if (e.key === 'Enter' && !e.shiftKey) runBtn.click();
    if (e.key === 'Escape') invoke('hide_window');
});
```

---

## Phase 4 — Distribution (1–2 tuần)

### 4.1 GoReleaser config

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
      - CGO_ENABLED=0   # modernc/sqlite là pure Go
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

changelog:
  sort: asc
  filters:
    exclude: ['^docs:', '^test:', 'Merge pull request']

# Homebrew tap
brews:
  - name: prompt-agent
    repository:
      owner: "{{ .Env.GITHUB_USERNAME }}"
      name: homebrew-prompt-agent
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/{{ .Env.GITHUB_USERNAME }}/prompt-agent"
    description: "AI prompt runner with recipe system — overlay"
    license: MIT
    test: |
      system "#{bin}/prompt-agent version"
    install: |
      bin.install "prompt-agent"

# AUR (Arch Linux)
aurs:
  - name: prompt-agent-bin
    homepage: "https://github.com/{{ .Env.GITHUB_USERNAME }}/prompt-agent"
    description: "AI prompt runner with recipe system"
    maintainers:
      - "Your Name <your@email.com>"
    license: MIT
    private_key: "{{ .Env.AUR_SSH_PRIVATE_KEY }}"
    git_url: "ssh://aur@aur.archlinux.org/prompt-agent-bin.git"
    package: |-
      install -Dm755 "./prompt-agent" "${pkgdir}/usr/bin/prompt-agent"
      install -Dm644 "./LICENSE" "${pkgdir}/usr/share/licenses/prompt-agent/LICENSE"
```

### 4.2 GitHub Actions — release pipeline

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Run tests
        run: go test ./...

      - name: GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
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
        with:
          go-version: '1.24'

      - name: Build Go sidecar
        run: make build-sidecar

      - name: Build Tauri app
        uses: tauri-apps/tauri-action@v0
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          APPLE_CERTIFICATE: ${{ secrets.APPLE_CERTIFICATE }}        # macOS signing
          APPLE_CERTIFICATE_PASSWORD: ${{ secrets.APPLE_CERTIFICATE_PASSWORD }}
          APPLE_ID: ${{ secrets.APPLE_ID }}                          # macOS notarize
```

### 4.3 Homebrew tap setup

```bash
# Một lần, tạo repo tap
brew tap-new YOUR_GITHUB_USERNAME/homebrew-prompt-agent
cd /opt/homebrew/Library/Taps/YOUR_GITHUB_USERNAME/homebrew-prompt-agent
git remote add origin git@github.com:YOUR_GITHUB_USERNAME/homebrew-prompt-agent.git
git push -u origin main
```

User cài:
```bash
brew tap YOUR_GITHUB_USERNAME/prompt-agent
brew install prompt-agent
```

### 4.4 AUR setup

```bash
# Một lần, tạo AUR account và upload SSH key tại aur.archlinux.org
# GoReleaser sẽ tự push PKGBUILD sau mỗi release
```

User cài:
```bash
yay -S prompt-agent-bin
# hoặc
paru -S prompt-agent-bin
```

### 4.5 Shell completions (bonus, Cobra built-in)

```bash
# Thêm vào .goreleaser.yaml archives section
# GoReleaser sẽ bundle completions tự động vào Homebrew formula

# Generate completions (Cobra built-in):
./prompt-agent completion bash > completions/prompt-agent.bash
./prompt-agent completion zsh  > completions/prompt-agent.zsh
./prompt-agent completion fish > completions/prompt-agent.fish
```

---

## Phase 5 — Polish (ongoing)

- History panel trong Tauri UI (tab hoặc slide-in)
- Recipe import/export JSON (share recipe packs)
- Customizable hotkey trong Settings UI (re-register global shortcut mà không restart)
- Winget manifest cho Windows (cần code-signing cert)
- `prompt-agent prompt list --output json` để Tauri fetch recipes

---

## Known bugs phải fix trước khi merge bất cứ thứ gì

| # | File | Bug | Fix |
|---|---|---|---|
| 1 | `internal/cli/init.go` | Template syntax `{{.topic}}` sai | Đổi thành `{{topic}}` |
| 2 | `internal/config/config.go` | `env:OPENAI_API_KEY` không được resolve | Thêm `resolveAPIKey()` |
| 3 | `internal/llm/gemini/client.go` | Fake streaming bằng 100-char chunks | Dùng `GenerateContentStream()` |
| 4 | `internal/cli/run_handler.go` | PromptEditor bắt buộc mỗi lần run | Chỉ show khi `--edit` flag |
| 5 | `internal/storage/json.go` | 3 prompt hardcode vào memory, không save ra file | Replace bằng SQLite |
| 6 | `README.md` | Nói storage là SQLite nhưng code dùng JSON | Sync sau khi Phase 1 xong |

---

## Dependencies cần thêm

```go
// go.mod additions
require (
    modernc.org/sqlite v1.34.0          // SQLite pure Go
    github.com/google/uuid v1.6.0       // UUID cho history records (đã có)
)
```

```toml
# src-tauri/Cargo.toml additions
[dependencies]
tauri = { version = "2", features = ["tray-icon"] }
tauri-plugin-global-shortcut = "2"
tauri-plugin-positioner = "2"
tauri-plugin-shell = "2"
tauri-plugin-clipboard-manager = "2"
```