Tuyệt 👍 Go là **lựa chọn rất đúng** cho app kiểu này (CLI-first, system hook, concurrency, binary gọn).
Dưới đây là **plan phát triển chi tiết theo từng giai đoạn**, đủ để bạn **đi từ 0 → MVP → usable product**.

---

# 🧠 TỔNG QUAN MỤC TIÊU

**Tên tạm:** `prompt-agent`

**Chức năng cốt lõi (MVP):**

* Lưu prompt template
* Lấy selection text từ hệ điều hành
* Trigger bằng hotkey / middle click
* Build prompt
* Gửi lên OpenAI / Gemini
* Hiển thị kết quả trong CLI / TUI

---

# 🧩 KIẾN TRÚC TỔNG THỂ (Go)

```text
cmd/
 └─ prompt-agent/        # entrypoint CLI
internal/
 ├─ config/              # load config, env
 ├─ prompt/              # prompt template + builder
 ├─ selection/           # lấy selected text
 ├─ trigger/             # mouse / hotkey
 ├─ llm/                 # OpenAI, Gemini adapters
 ├─ output/              # stdout, TUI
 ├─ storage/             # sqlite / json
 └─ app/                 # orchestration
pkg/
 └─ models/              # struct dùng chung
```

---

# 🗺️ ROADMAP PHÁT TRIỂN CHI TIẾT

## PHASE 0 – Chuẩn bị & định hướng (1–2 ngày)

### Việc cần làm

* Chọn OS target đầu tiên → **macOS (khuyên)**
* Xác định phạm vi MVP
* Đặt chuẩn coding & cấu trúc repo

### Tech stack

| Mục     | Công nghệ                     |
| ------- | ----------------------------- |
| CLI     | `cobra`                       |
| Config  | `viper`                       |
| HTTP    | `net/http`                    |
| TUI     | `bubbletea`                   |
| Storage | SQLite (`modernc.org/sqlite`) |
| JSON    | `encoding/json`               |

---

## PHASE 1 – CLI Skeleton & Config (2–3 ngày)

### Mục tiêu

Có CLI chạy được, load config, gọi command

### Commands cơ bản

```bash
prompt-agent init
prompt-agent prompt add
prompt-agent prompt list
prompt-agent run <prompt-id>
```

### Cấu trúc

```go
func main() {
  cmd.Execute()
}
```

### Config file

```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4.1-mini
  gemini:
    api_key: env:GEMINI_API_KEY

ui:
  output: tui # stdout | tui
```

---

## PHASE 2 – Prompt Storage & Builder (3–4 ngày)

### Prompt schema (rất quan trọng)

```go
type Prompt struct {
  ID          string
  Name        string
  Engine      string
  Template    string
  Variables   []string
  CreatedAt  time.Time
}
```

### Ví dụ template

```text
Bạn là chuyên gia.
Hãy giải thích đoạn sau:

{{selection}}
```

### Builder

```go
func BuildPrompt(tmpl string, data map[string]string) string
```

### Lưu trữ

* SQLite:

```sql
prompts(id TEXT PRIMARY KEY, name TEXT, engine TEXT, template TEXT)
```

---

## PHASE 3 – Selection Text Collector (OS-specific) (2–3 ngày)

### macOS implementation (MVP)

```go
cmd := exec.Command("pbpaste")
out, _ := cmd.Output()
selection := string(out)
```

📌 **Note:**
MVP chỉ cần vậy — sau này nâng cấp Accessibility API.

### Interface chuẩn

```go
type SelectionProvider interface {
  GetSelection() (string, error)
}
```

---

## PHASE 4 – Trigger System (3–5 ngày)

### Giai đoạn 1 (khuyên dùng)

➡️ **Hotkey trước, middle click sau**

#### Cách dễ:

* Dùng `skhd` (macOS hotkey daemon)
* Map:

```text
alt + space → prompt-agent run last
```

### Giai đoạn 2 (native)

* CGEventTap
* cgo wrapper (nâng cấp sau)

### Interface

```go
type Trigger interface {
  Listen() error
}
```

---

## PHASE 5 – LLM Gateway (5–6 ngày)

### Interface chung

```go
type LLMClient interface {
  Send(prompt string) (string, error)
}
```

### OpenAI Adapter

```go
POST /v1/chat/completions
```

* Retry
* Timeout
* Streaming (optional)

### Gemini Adapter

```go
POST generative.tuning
```

📌 **Best practice**

* Mỗi engine 1 file
* Không hard-code model

---

## PHASE 6 – Output Layer (CLI / TUI) (3–4 ngày)

### Mode 1 – stdout

```bash
prompt-agent run explain
```

### Mode 2 – TUI (khuyên)

* Markdown render
* Scroll
* Copy
* History

### Interface

```go
type Renderer interface {
  Render(text string) error
}
```

---

## PHASE 7 – App Orchestration (2–3 ngày)

### Flow

```go
selection := selectionProvider.GetSelection()
prompt := promptBuilder.Build(template, selection)
response := llm.Send(prompt)
renderer.Render(response)
```

### App struct

```go
type App struct {
  Storage
  Selection
  LLM
  Renderer
}
```

---

## PHASE 8 – UX polish & tiện ích (tuỳ chọn)

* Prompt preview
* Prompt chaining
* History viewer
* Cache kết quả
* Cost tracking
* Multi-engine fallback

---

## 🧪 Testing Strategy

* Unit test:

  * Prompt builder
  * LLM adapter mock
* Integration test:

  * Run command
* Manual test:

  * Selection → trigger → output

---

## 📦 Packaging & Distribution

* `go build -ldflags "-s -w"`
* Homebrew tap
* Single binary

---

## ⏱️ ƯỚC LƯỢNG THỜI GIAN (Solo dev)

| Phase            | Thời gian |
| ---------------- | --------- |
| Core MVP         | ~2–3 tuần |
| Stable beta      | ~1 tháng  |
| Polished product | ~2 tháng  |

---

## 🚀 NEXT STEP GỢI Ý

Nếu bạn muốn, mình có thể:

1. Thiết kế **CLI spec chi tiết**
2. Viết **prompt schema nâng cao**
3. Tạo **repo skeleton Go**
4. Demo **prompt-agent MVP flow**
5. Gợi ý **hotkey + mouse hook cụ thể cho macOS**

👉 Bạn muốn **bắt đầu từ Phase nào ngay bây giờ?**
