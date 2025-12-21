Rất tốt 👍
**Phase 5 là phase “kết nối ngữ cảnh”** – nơi app của bạn **thực sự thông minh**, vì nó **hiểu người dùng đang chọn cái gì**.

Nếu Phase 4 là *kích hoạt*, thì **Phase 5 là “đưa dữ liệu sống vào prompt”**.

---

# 🧠 PHASE 5 – SELECTION TEXT & RUN PIPELINE (CHI TIẾT)

> 🎯 **Mục tiêu Phase 5**
>
> * Lấy **selection text** từ hệ điều hành
> * Inject selection vào Prompt Builder
> * Hoàn thiện command `prompt-agent run <prompt-id>`
> * Trigger (hotkey / mouse) → chạy được **end-to-end**
> * Output ra CLI (stdout trước)

❌ Chưa cần streaming
❌ Chưa cần TUI fancy
❌ Chưa cần LLM đa engine (sẽ làm Phase 6)

---

# 1. Kết quả mong muốn sau Phase 5

Người dùng có thể:

```text
Select text → hotkey → kết quả xuất hiện trong terminal
```

CLI:

```bash
prompt-agent run explain_code
```

Hoặc từ trigger:

```bash
alt + space
```

---

# 2. Kiến trúc tổng thể Phase 5

```text
[Trigger]
   ↓
[Run Command]
   ↓
[Selection Provider]
   ↓
[Prompt Builder]
   ↓
[Final Prompt String]
   ↓
[Output Renderer]
```

📌 **LLM sẽ gắn vào Phase 6**, ở Phase 5 ta mock / echo prompt.

---

# 3. Selection Text System (CORE)

## 3.1 Triết lý

* **Selection provider là OS-specific**
* App core không quan tâm OS
* Có fallback an toàn

---

## 3.2 Interface chuẩn

📄 `internal/selection/selection.go`

```go
package selection

type Provider interface {
	Get() (string, error)
	Name() string
}
```

---

## 3.3 macOS implementation (MVP)

📄 `internal/selection/macos.go`

```go
package selection

import (
	"os/exec"
	"strings"
)

type MacOSProvider struct{}

func (p *MacOSProvider) Name() string {
	return "macos"
}

func (p *MacOSProvider) Get() (string, error) {
	cmd := exec.Command("pbpaste")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
```

📌 **Note cực kỳ quan trọng**

* `pbpaste` trả về *clipboard*, không phải selection gốc
* Nhưng **90% use case OK**
* Native Accessibility API → Phase 7+

---

## 3.4 Linux & Windows (stub)

📄 `internal/selection/linux.go`

```go
// xclip -selection primary
```

📄 `internal/selection/windows.go`

```go
// Get-Clipboard
```

👉 Phase 5 chỉ cần macOS chạy thật.

---

# 4. Selection Manager

📄 `internal/selection/manager.go`

```go
type Manager struct {
	provider Provider
}

func (m *Manager) GetSelection() (string, error)
```

👉 Sau này fallback nhiều provider.

---

# 5. Run Command (TRÁI TIM CỦA APP)

## 5.1 CLI spec

```bash
prompt-agent run <prompt-id>
```

Flags:

```bash
--dry-run     # chỉ show prompt
--no-select   # không lấy selection
```

---

## 5.2 Flow chi tiết

📄 `internal/cli/run.go`

```text
1. Load prompt từ storage
2. Lấy selection text
3. Validate variables
4. Build final prompt
5. Output result
```

---

## 5.3 Validate variables (RẤT QUAN TRỌNG)

Ví dụ prompt cần:

```text
{{selection}}
{{language}}
```

Nhưng chỉ có `selection`

→ FAIL sớm:

```text
Missing variable: language
```

📄 `internal/prompt/validate.go`

```go
func ValidateVariables(
	required []string,
	data map[string]string,
) error
```

---

## 6. Prompt Builder + Selection Injection

### Data map

```go
data := map[string]string{
	"selection": selectedText,
}
```

Sau này mở rộng:

```go
"filename"
"language"
"app"
```

---

## 7. Output Renderer (Phase 5 – STDOUT)

📄 `internal/output/stdout.go`

```go
package output

import "fmt"

type StdoutRenderer struct{}

func (r *StdoutRenderer) Render(text string) error {
	fmt.Println(text)
	return nil
}
```

📌 TUI để Phase 6.

---

# 8. Error Handling UX

## Nguyên tắc

* Trigger không được crash
* Lỗi ngắn, dễ hiểu

### Ví dụ

```text
❌ No text selected
❌ Prompt not found: explain_code
❌ Missing variable: language
```

---

# 9. Config liên quan Phase 5

📄 `config.yaml`

```yaml
selection:
  provider: macos

run:
  fail_on_empty_selection: true
```

---

# 10. Manual Test Checklist

### Test 1 – Normal

* Select text
* Hotkey
* Prompt build OK

### Test 2 – No selection

* Không select gì
* Chạy run
* Error đúng

### Test 3 – Dry run

```bash
prompt-agent run explain_code --dry-run
```

→ In prompt, không gửi LLM

---

# 11. Checkpoint cuối Phase 5

✔️ Selection text hoạt động
✔️ Prompt build hoàn chỉnh
✔️ Run command ổn định
✔️ Trigger → end-to-end OK

👉 **Tại thời điểm này app đã usable cho developer nội bộ.**

---

# 12. Những bẫy thường gặp ❌

❌ Tin rằng pbpaste = selection 100%
❌ Không trim whitespace
❌ Không validate variable
❌ Prompt run blocking quá lâu

---

# ➡️ PHASE 6 (PREVIEW) – LLM GATEWAY & OUTPUT

Phase 6 sẽ làm:

* OpenAI / Gemini adapter
* Streaming response
* TUI render (bubbletea)
* Cancel request (Ctrl+C)

---

Nếu bạn muốn, mình có thể:
1️⃣ Viết **full code run command**
2️⃣ Thiết kế **selection provider native (Accessibility API)**
3️⃣ Chuẩn bị **Phase 6 chi tiết**
4️⃣ Review lại toàn bộ architecture từ Phase 1–5

👉 Bạn muốn **đi tiếp Phase 6 hay đào sâu Phase 5 (selection nâng cao)?**
