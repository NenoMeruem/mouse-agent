🤖 PHASE 6 – LLM GATEWAY & OUTPUT UX (CHI TIẾT)

🎯 Mục tiêu Phase 6

Kết nối LLM thật (OpenAI, Gemini)

Chuẩn hóa LLM Adapter

Streaming response

TUI output (Bubbletea)

Cancel request (Ctrl+C)

Error handling + retry

✅ Sau phase này: app dùng được thực tế

1. Kiến trúc tổng thể Phase 6
[Run Command]
   ↓
[Prompt Builder]
   ↓
[LLM Gateway]
   ├─ OpenAI Adapter
   ├─ Gemini Adapter
   ↓
[Streaming Channel]
   ↓
[Renderer]
   ├─ Stdout
   └─ TUI (Bubbletea)

2. LLM Gateway – Thiết kế CHUẨN (rất quan trọng)
2.1 Vì sao phải có Gateway?

Tránh code dính chặt OpenAI

Dễ thêm Gemini / Anthropic

Test dễ (mock)

Retry / timeout tập trung

2.2 Interface LLM chuẩn

📄 internal/llm/client.go

package llm

import "context"

type Request struct {
	Prompt string
	Model  string
}

type Chunk struct {
	Text string
	Err  error
	Done bool
}

type Client interface {
	Name() string
	Stream(ctx context.Context, req Request) (<-chan Chunk, error)
}


📌 Giải thích

Stream trả về channel → streaming

Chunk.Done = true → kết thúc

context.Context → cancel / timeout

3. OpenAI Adapter (Streaming)
3.1 API sử dụng

POST /v1/chat/completions

stream: true

3.2 Struct

📄 internal/llm/openai/client.go

type OpenAIClient struct {
	APIKey string
	Model  string
}

3.3 Streaming implementation (concept)
func (c *OpenAIClient) Stream(
	ctx context.Context,
	req Request,
) (<-chan Chunk, error) {

	ch := make(chan Chunk)

	go func() {
		defer close(ch)

		// HTTP request with stream=true
		// đọc từng line SSE
		// parse delta.content
		// ch <- Chunk{Text: "..."}
	}()

	return ch, nil
}


📌 Lưu ý

Dùng bufio.Scanner

Handle [DONE]

Không block goroutine

4. Gemini Adapter (không streaming ở MVP)

📄 internal/llm/gemini/client.go

Gemini streaming phức tạp hơn

Phase 6 có thể:

non-stream first

wrap thành fake stream (1 chunk)

👉 Kiến trúc vẫn giữ Stream

5. LLM Manager (Router)

📄 internal/llm/manager.go

type Manager struct {
	clients map[string]Client
}

func (m *Manager) Get(engine string) (Client, error)


Flow:

prompt.Engine → llmManager.Get(engine)

6. Run Command – nâng cấp cho LLM
6.1 Flow mới
1. Build prompt
2. Resolve LLM client
3. context.WithCancel
4. Start stream
5. Render stream
6. Handle cancel / error

6.2 Cancel bằng Ctrl+C
ctx, cancel := signal.NotifyContext(
	context.Background(),
	os.Interrupt,
)
defer cancel()


📌 Ctrl+C:

Dừng request

Không crash app

In “Canceled”

7. Output Renderer – TUI (Bubbletea)
7.1 Vì sao cần TUI?

Streaming mượt

Scroll

Copy

Markdown render

UX tốt hơn stdout rất nhiều

7.2 Renderer interface (chuẩn hóa)

📄 internal/output/renderer.go

type Renderer interface {
	RenderStream(ch <-chan llm.Chunk) error
}

8. TUI Renderer – Bubbletea
8.1 Model

📄 internal/output/tui/model.go

type Model struct {
	content string
	done    bool
	err     error
}

8.2 Update logic
- Nhận Chunk
- Append text
- Re-render

8.3 UX yêu cầu

⏳ Spinner khi đang load

📄 Scroll bằng arrow

❌ Ctrl+C → cancel

✅ Copy bằng mouse

9. Stdout Renderer (fallback)

Nếu user set:

ui:
  output: stdout


→ Print streaming inline:

AI: Hel
AI: Hello
AI: Hello world

10. Error Handling & Retry
10.1 Lỗi cần handle
Lỗi	Xử lý
401	Báo API key
Timeout	Retry 1 lần
Network	Fail fast
Model not found	Fallback
10.2 Retry strategy
max retry: 1
timeout: 30s

11. Config Phase 6

📄 config.yaml

engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4.1-mini
    timeout: 30s

ui:
  output: tui

12. Performance & UX rules

⏱ First token < 1s

🧠 Streaming không giật

🚫 Không block trigger

💥 Panic = bug

13. Manual Test Checklist
Test 1 – Streaming

Prompt dài

Token ra liên tục

Test 2 – Cancel

Ctrl+C giữa chừng

App không chết

Test 3 – Engine switch

openai → gemini

14. Checkpoint cuối Phase 6

✔️ Prompt → LLM thật
✔️ Streaming ổn định
✔️ TUI đẹp, usable
✔️ Cancel hoạt động

👉 Đến đây app đã là “daily driver” cho dev.

15. Những sai lầm nghiêm trọng ❌

❌ Không dùng context
❌ Dính chặt OpenAI SDK
❌ Render trong goroutine network
❌ Không xử lý Ctrl+C