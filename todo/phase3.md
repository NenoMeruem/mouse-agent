Tuyệt 👍
**Phase 3 là bước “biến prompt thành hành động”**: lấy **selection text thật**, build prompt thật, và **chạy được end-to-end (chưa gọi LLM)**.
Mình sẽ mô tả **rất chi tiết**, theo đúng tinh thần: *code theo là chạy được*.

---

# 🧩 PHASE 3 – SELECTION + RUN PROMPT (CHI TIẾT)

> 🎯 **Mục tiêu Phase 3**
>
> * Lấy selection text từ OS
> * Inject selection vào prompt
> * Có command `prompt-agent run <prompt-id>`
> * Preview final prompt trong CLI
>   ❌ Chưa gửi lên OpenAI / Gemini
>   ❌ Chưa cần mouse hook

---

## 1. Kết quả mong muốn sau Phase 3

Bạn có thể:

```bash
# Select text ở bất kỳ app nào
prompt-agent run explain_code
```

CLI in ra:

```text
=== PROMPT PREVIEW ===
Bạn là senior dev.
Hãy giải thích đoạn sau:

func main() { ... }
```

---

## 2. Thiết kế Selection Layer (OS abstraction)

### 2.1 Tư duy kiến trúc

Selection **luôn là OS-specific**, nên cần abstraction:

```text
SelectionProvider
 ├─ macOS
 ├─ Linux
 └─ Windows
```

---

### 2.2 Interface chuẩn

📄 `internal/selection/provider.go`

```go
package selection

type Provider interface {
	Get() (string, error)
}
```

---

## 3. macOS Selection Provider (MVP)

### 3.1 Cách đơn giản nhất (đủ cho Phase 3)

Dùng clipboard:

```bash
pbpaste
```

📄 `internal/selection/macos.go`

```go
//go:build darwin

package selection

import (
	"bytes"
	"os/exec"
	"strings"
)

type MacOSProvider struct{}

func (p *MacOSProvider) Get() (string, error) {
	cmd := exec.Command("pbpaste")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(out.String()), nil
}
```

📌 **Note**

* Đây là clipboard, không phải “real selection”
* Nhưng **MVP OK**
* Phase sau nâng cấp Accessibility API

---

### 3.2 Linux & Windows (stub)

📄 `internal/selection/linux.go`

```go
//go:build linux

package selection

func (p *LinuxProvider) Get() (string, error) {
	return "", errors.New("not implemented")
}
```

---

## 4. Selection Factory

📄 `internal/selection/factory.go`

```go
package selection

func NewProvider() Provider {
	return &MacOSProvider{}
}
```

(Sau này switch theo OS)

---

## 5. Run Command (CLI CORE)

## 5.1 UX

```bash
prompt-agent run explain_code
```

Optional flags:

```bash
--dry-run   # chỉ preview prompt
--raw       # không format
```

---

## 5.2 Command definition

📄 `internal/cli/run.go`

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run <prompt-id>",
	Short: "Run prompt with current selection",
	Args:  cobra.ExactArgs(1),
	RunE:  runPrompt,
}
```

---

## 6. App Orchestration (Phase 3 brain)

### 6.1 AppContext mở rộng

📄 `internal/app/context.go`

```go
type AppContext struct {
	PromptStore storage.PromptStore
	Builder     prompt.Builder
	Selection   selection.Provider
}
```

---

### 6.2 runPrompt flow

📄 `internal/cli/run_handler.go`

```go
func runPrompt(cmd *cobra.Command, args []string) error {
	id := args[0]

	promptDef, err := appCtx.PromptStore.Get(id)
	if err != nil {
		return err
	}

	selectionText, err := appCtx.Selection.Get()
	if err != nil {
		return err
	}

	data := map[string]string{
		"selection": selectionText,
	}

	finalPrompt, err := appCtx.Builder.Build(
		promptDef.Template,
		data,
	)
	if err != nil {
		return err
	}

	fmt.Println("=== PROMPT PREVIEW ===")
	fmt.Println(finalPrompt)

	return nil
}
```

📌 **Key insight**

* Phase 3 = **Prompt rendering engine**
* LLM chưa cần

---

## 7. Variable Validation (Quan trọng)

### 7.1 Check missing variables

```go
for _, v := range promptDef.Variables {
	if _, ok := data[v]; !ok {
		return fmt.Errorf("missing variable: %s", v)
	}
}
```

📌 Giúp tránh prompt lỗi

---

## 8. UX Improvements (Phase 3)

### 8.1 Selection empty

Nếu không có selection:

```text
No text selected. Please select text first.
```

### 8.2 Confirm preview

```bash
prompt-agent run explain_code --confirm
```

---

## 9. Test checklist

### Manual

* Select text
* `prompt-agent run`
* Prompt render đúng

### Unit

* Selection provider mock
* Run handler mock store

---

## 10. Checkpoint cuối Phase 3

✔️ Selection hoạt động
✔️ Prompt render đúng
✔️ CLI flow mượt
✔️ Sẵn sàng gửi LLM

---

## 11. Sau Phase 3 bạn đã có gì?

Bạn đã có:

```text
User selects text
→ prompt-agent run
→ prompt + selection merged
→ preview in CLI
```

🎯 **80% core UX đã hoàn chỉnh**

---

# ➡️ PHASE 4 (Preview)

Phase 4 sẽ:

* LLM Gateway (OpenAI / Gemini)
* Streaming output
* Error handling
* Cost / timeout

---

Nếu bạn muốn, mình có thể:
1️⃣ Viết **code hoàn chỉnh cho run command**
2️⃣ Nâng cấp selection → Accessibility API
3️⃣ Thiết kế prompt variable system nâng cao
4️⃣ Chuẩn bị **Phase 4 ngay**

👉 Bạn muốn **đi tiếp Phase 4 hay đào sâu Phase 3** (UX / selection)?
