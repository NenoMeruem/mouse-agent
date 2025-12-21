Rất tốt 👍
**Phase 4 là phase “kích hoạt” (Activation Layer)** – nơi app của bạn **từ CLI thủ công → tool phản xạ tức thì theo hành vi người dùng**.

Ở phase này, ta **kết nối con người ↔ prompt system** thông qua **trigger**.

---

# ⚡ PHASE 4 – TRIGGER SYSTEM (CHI TIẾT)

> 🎯 **Mục tiêu Phase 4**
>
> * Có cơ chế trigger để chạy prompt **không cần gõ lệnh dài**
> * Hỗ trợ **hotkey** (MVP – bắt buộc)
> * Chuẩn bị kiến trúc cho **middle click / mouse trigger**
> * Tách trigger thành module độc lập (extensible)

❌ Chưa cần LLM streaming
❌ Chưa cần UI fancy

---

# 1. Triết lý thiết kế Trigger

### Trigger KHÔNG:

* Không chứa logic prompt
* Không chứa logic LLM

### Trigger CHỈ:

* Lắng nghe sự kiện (hotkey / mouse)
* Gọi **1 command chuẩn** của app

👉 Trigger = **external event → internal command**

---

# 2. Kiến trúc tổng thể Phase 4

```text
[User Action]
   ↓
[Trigger Listener]
   ↓
[Trigger Handler]
   ↓
prompt-agent run <prompt-id>
```

---

# 3. Phân loại Trigger (theo độ ưu tiên)

## Tier 1 – Hotkey (MVP, bắt buộc)

* Ổn định
* Dễ cross-platform
* Không cần quyền cao

## Tier 2 – Mouse (middle click)

* UX tốt
* OS-specific
* Làm sau nhưng **thiết kế sẵn hook**

---

# 4. Trigger Interface (CORE)

📄 `internal/trigger/trigger.go`

```go
package trigger

type Trigger interface {
	Start() error
	Stop() error
	Name() string
}
```

📌 Mục tiêu:

* Sau này có thể:

  * HotkeyTrigger
  * MouseTrigger
  * TrayTrigger

---

# 5. Phase 4A – Hotkey Trigger (MVP)

## 5.1 Chiến lược khuyến nghị (KHÔN NGOAN)

❗ **Không viết native hotkey listener trong Go ở MVP**

👉 Thay vào đó:

* Dùng **OS hotkey daemon**
* Trigger chỉ gọi `prompt-agent run ...`

### Vì sao?

* Native hotkey = cgo + permission + bug
* External daemon = ổn định, battle-tested

---

## 5.2 macOS – skhd (KHÔNG THỂ TỐT HƠN)

### Cài

```bash
brew install skhd
```

### Config

📄 `~/.skhdrc`

```text
alt + space : prompt-agent run explain_code
alt + shift + space : prompt-agent run summarize
```

📌 **Done – không cần code**

---

## 5.3 Linux – sxhkd

```bash
sxhkd &
```

```text
alt + space
  prompt-agent run explain_code
```

---

## 5.4 Windows – AutoHotkey

```ahk
!Space::
Run, prompt-agent.exe run explain_code
Return
```

---

## 6. Phase 4B – Internal Trigger Manager (Chuẩn bị)

Dù MVP dùng external hotkey, **ta vẫn thiết kế trigger manager**.

📄 `internal/trigger/manager.go`

```go
type Manager struct {
	triggers []Trigger
}

func (m *Manager) StartAll() error
func (m *Manager) StopAll() error
```

👉 Phase 5+ có thể bật internal trigger.

---

# 7. Phase 4C – Mouse Trigger (Thiết kế sẵn)

## 7.1 UX mong muốn

```text
User select text
→ middle click
→ menu prompt hiện ra
→ chọn prompt
→ run
```

---

## 7.2 Kiến trúc Mouse Trigger

```text
MouseEvent
  ↓
Context Menu
  ↓
Prompt Selector
  ↓
Run Prompt
```

---

## 7.3 macOS – hướng triển khai sau này

### Cách 1 – Karabiner (recommended)

* Map middle click → hotkey
* Dùng skhd xử lý tiếp

### Cách 2 – Native CGEventTap (Phase 6+)

* cgo
* Accessibility permission

📌 **Phase 4 chỉ cần define interface**

📄 `internal/trigger/mouse.go`

```go
type MouseTrigger struct {
	Button string // middle
}
```

---

# 8. Trigger Configuration (Phase 4)

## 8.1 Config schema

```yaml
triggers:
  enabled: true
  default_prompt: explain_code
```

📌 Cho phép:

* Run prompt mặc định
* Sau này override per trigger

---

## 9. Prompt Run Command (liên kết Phase 3)

Trigger sẽ gọi:

```bash
prompt-agent run explain_code
```

Nên cần đảm bảo:

* `run` command **idempotent**
* Không interactive
* Không panic

---

# 10. UX tối ưu cho Trigger

### Nguyên tắc

* Trigger → kết quả trong < 1s
* Không block terminal
* Không yêu cầu input

### Best practices

* Default prompt
* Silent mode
* Error hiển thị ngắn

---

# 11. Checkpoint cuối Phase 4

✔️ Có hotkey trigger chạy được
✔️ Không cần gõ lệnh tay
✔️ Kiến trúc sẵn sàng cho mouse
✔️ Không lock-in OS

---

# 12. Những sai lầm cần tránh ❌

❌ Viết native hotkey sớm
❌ Gộp trigger logic vào CLI
❌ Hard-code prompt ID
❌ Block UI khi trigger

---

# ➡️ PHASE 5 (PREVIEW) – SELECTION TEXT

Phase 5 sẽ làm:

* Lấy selection text (pbpaste / xclip)
* Inject vào prompt builder
* `run` command hoàn chỉnh

---

Nếu bạn muốn, mình có thể:
1️⃣ Viết **run command chuẩn cho trigger**
2️⃣ Thiết kế **prompt selector menu (TUI)**
3️⃣ Lập **roadmap mouse trigger native**
4️⃣ Review **hotkey UX flow thực tế**

👉 Bạn muốn **đi tiếp Phase 5 hay đào sâu Phase 4 (mouse trigger)?**
