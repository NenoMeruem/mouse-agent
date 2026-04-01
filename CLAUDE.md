# Prompt Agent

CLI tool + Tauri overlay app để chạy AI prompt recipes với clipboard content.
Inspired by Logi AI Prompt Builder.

## Commands

```bash
go build -o prompt-agent ./cmd/prompt-agent
go test ./...
go vet ./...
```

## Architecture

```
CLI (Cobra) → AppContext → PromptStore (SQLite) + HistoryStore (SQLite)
                        → LLM Manager → OpenAI / Gemini / Claude (streaming)
                        → Output: StdoutRenderer | TUIRenderer (Bubble Tea)
```

**Key files:**
- `internal/cli/run_handler.go` — core run flow, LLM dispatch
- `internal/storage/sqlite.go` — SQLite backend (thay thế json.go)
- `internal/llm/*/client.go` — LLM providers, tất cả implement `llm.Client`
- `internal/config/config.go` — Viper config, constants, helper getters
- `pkg/models/prompt.go` — Prompt + RunRecord data models

## Storage

**SQLite only** — `modernc.org/sqlite` (pure Go, no CGO).
DB path: `~/.prompt-agent/prompts.db`

Hai bảng chính:
- `prompts` — recipe definitions (id, name, engine, template, variables, params, icon)
- `run_history` — mỗi lần run (prompt_id, input_text, response, duration_ms)

> KHÔNG dùng json.go nữa. Nếu thấy code dùng JSONStore → đó là legacy, cần migrate.

## Data model

```go
type Prompt struct {
    ID, Name, Description, Engine, Template string
    Variables []string   // extracted từ {{variable}} placeholders
    Params    []string   // ["tone","length","complexity"] — UI pills
    Icon      string
    CreatedAt, UpdatedAt time.Time
}

type RunRecord struct {
    ID, PromptID, Engine   string
    InputText, FinalPrompt string
    Response               string
    DurationMs             int64
    Error                  string
    CreatedAt              time.Time
}
```

## Template syntax

Dùng `{{variable}}` (KHÔNG phải `{{.variable}}`).

```
"Explain this code:\n{{selection}}"
```

`selection` là reserved variable — lấy từ clipboard hoặc stdin.

## Config

File: `~/.prompt-agent/config.yaml`

```yaml
engines:
  gemini:
    api_key: env:GEMINI_API_KEY   # prefix "env:" được resolve thành os.Getenv()
    model: gemini-2.5-flash-lite
ui:
  output: stdout  # hoặc "tui"
```

**QUAN TRỌNG:** `api_key: env:SOMETHING` phải được resolve bằng `resolveAPIKey()` trong config.go.
Không được load thẳng string `"env:SOMETHING"` vào client.

## LLM providers

Tất cả implement interface `llm.Client`:
```go
type Client interface {
    Name() string
    Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}
```

Gemini phải dùng `GenerateContentStream()` — KHÔNG dùng `GenerateContent()` rồi fake chunk.

## Run flow

```
run_handler.go:
1. Load prompt từ SQLite
2. GetSelection() — clipboard HOẶC stdin (khi --no-select)
3. collectMissingVariables() — env vars → interactive input
4. InjectParams() — append tone/length suffix vào cuối prompt
5. [--edit flag only] PromptEditor TUI
6. runWithLLM() → spawn LLM → tee stdout → HistoryStore.Append()
```

**--no-select + stdin:** khi Tauri pipe text vào:
```go
stat, _ := os.Stdin.Stat()
if (stat.Mode() & os.ModeCharDevice) == 0 {
    bytes, _ := io.ReadAll(os.Stdin)
    data["selection"] = strings.TrimSpace(string(bytes))
}
```

## Tauri integration (Phase 3)

Go binary chạy như **sidecar** — Tauri spawn process, pipe stdin, đọc stdout line-by-line.

```rust
// Rust gọi Go:
app.shell().sidecar("prompt-agent")
   .args(["run", recipe_id, "--no-select", "--raw"])
   .spawn() // pipe selection qua stdin, emit stdout về WebView
```

```javascript
// WebView nhận:
await listen('chunk', ({ payload }) => output.textContent += payload);
await listen('done', () => { runBtn.disabled = false; });
```

Tauri cần flag `--output json` cho list commands:
```bash
./prompt-agent prompt list --output json   # Tauri dùng để load recipe list
```

## Known bugs (cần fix)

- [ ] `init.go` tạo template `{{.topic}}` → phải là `{{topic}}`
- [ ] `config.go` không resolve `env:` prefix
- [ ] `gemini/client.go` fake streaming, cần dùng `GenerateContentStream()`
- [ ] `run_handler.go` mở PromptEditor bắt buộc → chỉ khi `--edit` flag

## Conventions

- Tất cả storage operations phải thread-safe (SQLite handles this)
- Error messages bắt đầu bằng `❌` khi hiển thị ra CLI
- Không hardcode API keys, model names — dùng `config.GetEngineModel(engine)`
- Test files: `*_test.go` cạnh file source
- Atomic file writes nếu còn dùng file (temp file + os.Rename)

## Import plan chi tiết

@./PLAN.md
