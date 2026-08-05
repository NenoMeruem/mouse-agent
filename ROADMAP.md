# Promptly Product Roadmap

Tài liệu hướng đi và lộ trình phát triển sản phẩm cho ứng dụng **Promptly** (Desktop Overlay AI Assistant).

---

## 🎯 Định hướng cốt lõi (Core Vision)

Promptly được định hướng trở thành **"Universal AI Layer"** cho hệ điều hành — ứng dụng hỗ trợ trí tuệ nhân tạo tức thì, hoạt động mượt mà trên mọi ứng dụng khác mà không gây ngắt quãng luồng công việc (flow state) của người dùng.

---

## 🚀 Giai đoạn 1: Nâng cao Trải nghiệm Người dùng (Near-Term Quality of Life - v1.1)

> **Mục tiêu**: Tối ưu tốc độ thao tác, đa dạng hóa định dạng đầu vào/đầu ra và giảm bớt các bước bấm chuột.

- [ ] **📷 Hỗ trợ Multimodal Input (OCR & Screen Snipping)**:
  - Cho phép chụp ảnh nhanh vùng màn hình (`Shortcut Snipping`) hoặc dán trực tiếp hình ảnh từ Clipboard.
  - Tích hợp các model Vision (Gemini 2.5 Flash Vision, GPT-4o, Claude Sonnet) để đọc chữ từ ảnh, giải thích UI/UX, hỗ trợ debug lỗi từ ảnh chụp màn hình.
- [ ] **⚡ Quick Launcher / Direct Hotkeys per Recipe**:
  - Gán phím tắt riêng cho từng Recipe yêu thích (Ví dụ: `Alt+Shift+E` để chạy trực tiếp *Explain Code*, `Alt+Shift+T` để *Dịch nhanh* mà không cần chọn thủ công trong danh sách).
- [ ] **🧠 Contextual Recipe Auto-Suggest (Smart Selection)**:
  - Tự động phân tích kiểu dữ liệu Clipboard (Code snippet, URL, StackTrace, Email, JSON).
  - Đẩy các Recipe liên quan nhất lên đầu danh sách gợi ý.
- [ ] **🎨 Rich Rendering Engine**:
  - Render Markdown nâng cao: Syntax Highlighting kèm nút **1-Click Copy Code**, hiển thị công thức toán (KaTeX/MathJax) và sơ đồ quy trình (Mermaid Diagrams).

---

## 🛡️ Giai đoạn 2: Local AI & Bảo mật Quyền riêng tư (v1.2)

> **Mục tiêu**: Tăng cường tính riêng tư, hỗ trợ mô hình AI offline hoàn toàn và bảo vệ dữ liệu nhạy cảm.

- [ ] **🏠 Local LLM Integration (Ollama / LocalAI / Llama.cpp)**:
  - Tích hợp thêm Provider cho các mô hình chạy cục bộ (Ollama API: `llama3`, `deepseek-r1`, `qwen2.5`).
  - Cho phép sử dụng AI hoàn toàn offline, không tốn chi phí API và bảo mật dữ liệu tuyệt đối.
- [ ] **🔒 Sensitive Data Redaction (PII & Secret Masking)**:
  - Tự động quét và che thông tin nhạy cảm (API Keys, Passwords, Email cá nhân, Số điện thoại, Số thẻ) trước khi gửi prompt lên Cloud API.
- [ ] **👤 Global System Persona & Memory**:
  - Cho phép định nghĩa "Persona / System Rules" toàn cục (Ví dụ: *"Tôi là Lập trình viên Senior Rust, trả lời cô đọng, đi thẳng vào vấn đề"*).
  - Tự động chèn Persona vào tất cả các phiên tương tác AI.

---

## ⚡ Giai đoạn 3: Tự động hóa & Hệ sinh thái (v2.0)

> **Mục tiêu**: Biến Promptly từ một công cụ tra cứu thành một trợ lý tự động hóa quy trình làm việc mạnh mẽ.

- [ ] **🔄 Direct Text Replacement (Auto-Insert Output)**:
  - Tính năng "Replace In-Place": Tự động dán kết quả đã xử lý đè trực tiếp lên đoạn văn bản đang được bôi đen ở ứng dụng gốc (VS Code, Slack, Word, Email) thông qua mô phỏng phím bấm.
- [ ] **⛓️ Chained Recipes & Pipelines**:
  - Cho phép kết hợp nhiều Recipe thành một chuỗi tự động. Ví dụ: *Tóm tắt văn bản* ➔ *Dịch sang tiếng Việt* ➔ *Chuyển thành định dạng Bullet Points*.
- [ ] **🌐 Web Search & Dynamic Context (RAG)**:
  - Tích hợp công cụ tìm kiếm web thời gian thực (DuckDuckGo / Tavily Search API).
  - Cho phép nhúng tài liệu cá nhân local (PDF, Markdown notes) để AI trả lời có ngữ cảnh chính xác.
- [ ] **📦 Recipe Hub & Sync**:
  - Xây dựng kho chia sẻ Recipe công khai từ cộng đồng (Community Recipe Store).
  - Hỗ trợ đồng bộ hóa riêng tư dữ liệu Recipe qua GitHub Gist hoặc Google Drive / iCloud.

---

## 📊 Tóm tắt Ưu tiên Sản phẩm (Prioritization Matrix)

| Tính năng | Tác động (Impact) | Độ khó (Complexity) | Ưu tiên |
|---|---|---|---|
| Direct Hotkey per Recipe | 🔥 Cao | 🟢 Thấp | P0 (Nên làm ngay) |
| Rich Markdown & Code Copy | 🔥 Cao | 🟢 Thấp | P0 (Nên làm ngay) |
| Local LLM (Ollama) | 🔥 Cao | 🟡 Trung bình | P1 (Giai đoạn tiếp) |
| Multimodal / Image Input | 🔥 Cao | 🟡 Trung bình | P1 (Giai đoạn tiếp) |
| PII / Secret Masking | 🟡 Trung bình | 🟢 Thấp | P1 (Giai đoạn tiếp) |
| Direct In-Place Text Replace | 🔥 Cao | 🔴 Cao (Tùy OS) | P2 (Đã lên kế hoạch) |
| Recipe Pipelines / Chaining | 🟡 Trung bình | 🟡 Trung bình | P2 (Đã lên kế hoạch) |
