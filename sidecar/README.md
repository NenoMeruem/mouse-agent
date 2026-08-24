# Promptly AI Engine — FastAPI + LangChain Sidecar

Microservice FastAPI cung cấp sức mạnh **LangChain**, **LangGraph**, và **Local RAG** cho ứng dụng desktop **Promptly**.

---

## 🌟 Tính năng Cốt lõi

- ⚡ **FastAPI & Real-Time SSE Streaming**: Stream token và trạng thái tác tử qua Server-Sent Events cực nhanh.
- 🔗 **LangChain LCEL Chained Pipelines**: Ghép nối nhiều bước xử lý Prompt liên tiếp (`/api/recipes/run`).
- 🤖 **LangGraph Autonomous Agent**: Vòng lặp tác tử tự trị hỗ trợ Tool Calling, Multi-step reasoning và tự sửa sai (`/api/agent/stream`).
- 🛠️ **Desktop Tool Calling**: Web Search (DuckDuckGo / Tavily), cào dữ liệu URL, đọc file nội bộ, chạy lệnh shell an toàn, tính toán biểu thức.
- 📚 **Local RAG & ChromaDB**: Đánh chỉ mục tài liệu (Markdown, PDF, Codebase) vào vector store cục bộ và tra cứu ngữ cảnh (`/api/rag/ingest`, `/api/rag/query`).
- 🔄 **Multi-Provider Fallback**: Tự động hỗ trợ Google Gemini, OpenAI, Claude Sonnet và Ollama (chạy local offline).

---

## 📁 Cấu trúc Thư mục

```
sidecar/
├── main.py                 # FastAPI Application & Lifespan setup
├── config.py               # Pydantic Settings & Environment loader
├── schemas.py              # Request/Response models & Stream event types
├── llm_factory.py          # Khởi tạo Chat Model & Smart Fallbacks
├── requirements.txt        # Danh sách thư viện Python
├── .env.example            # Mẫu biến môi trường API keys
├── chains/
│   ├── recipe_pipeline.py  # LCEL Chained Pipelines & Variable injection
│   └── rag_engine.py       # ChromaDB Vector Store & Document Loaders
├── agents/
│   └── graph_runner.py     # LangGraph ReAct Agent Loop
├── tools/
│   └── desktop_tools.py    # Native Desktop Tools (Search, File, Calc, Shell)
└── routers/
    ├── health.py           # Health check & status API
    ├── recipes.py          # /api/recipes/run SSE endpoint
    ├── rag.py              # /api/rag/ingest & /api/rag/query SSE
    └── agent.py            # /api/agent/stream SSE endpoint
```

---

## 🚀 Hướng dẫn Cài đặt & Khởi chạy

### 1. Khởi tạo Virtual Environment

```bash
# Di chuyển vào thư mục sidecar (hoặc từ root project)
cd sidecar

# Tạo virtualenv với Python 3.10+
python3 -m venv venv

# Kích hoạt virtualenv (macOS/Linux)
source venv/bin/activate

# Cài đặt dependencies
pip install -r requirements.txt
```

### 2. Cấu hình Biến Môi trường

Sao chép file `.env.example` thành `.env`:

```bash
cp .env.example .env
```

Chỉnh sửa `.env` và điền API keys của bạn (ít nhất một trong các key sau):
- `GEMINI_API_KEY`
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `TAVILY_API_KEY` (Tùy chọn, phục vụ web search chất lượng cao)

### 3. Chạy Server Development

Từ thư mục gốc của project:

```bash
# Chạy trực tiếp qua uvicorn
uvicorn sidecar.main:app --host 127.0.0.1 --port 8000 --reload
```

---

## 📡 Tài liệu API & Endpoints

Khi server đang chạy, truy cập Interactive Swagger UI tại:
👉 **`http://127.0.0.1:8000/docs`**

### 1. Health Check
- **`GET /api/health`**: Kiểm tra trạng thái server và các API key đã được cấu hình.

### 2. Chained Recipe Execution (SSE Streaming)
- **`POST /api/recipes/run`**
- Payload mẫu:
```json
{
  "recipe_id": "review-pr",
  "selection": "def add(a, b): return a - b",
  "parameters": {
    "tone": "professional",
    "provider": "gemini"
  },
  "steps": [
    {
      "step_id": "find_bug",
      "template": "Analyze this code for bugs: {{selection}}",
      "output_key": "bug_analysis"
    },
    {
      "step_id": "write_fix",
      "template": "Based on this bug analysis:\n{{selection}}\nWrite the fixed code and a unit test.",
      "output_key": "fix"
    }
  ]
}
```

### 3. Desktop Document Ingestion & RAG
- **`POST /api/rag/ingest`**: Nạp thư mục tài liệu vào ChromaDB.
```json
{
  "directory_or_file": "~/Documents/notes",
  "collection_name": "personal_notes"
}
```
- **`POST /api/rag/query`**: Tra cứu và sinh câu trả lời có ngữ cảnh (SSE).
```json
{
  "query": "Làm thế nào để cấu hình hotkey trong Promptly?",
  "collection_name": "personal_notes",
  "provider": "gemini"
}
```

### 4. Autonomous Agent Loop (LangGraph SSE)
- **`POST /api/agent/stream`**: Chạy vòng lặp Agent có khả năng gọi công cụ.
```json
{
  "prompt": "Hãy tìm kiếm thông tin về bản phát hành Python 3.14 mới nhất và tính 2^10",
  "provider": "gemini",
  "enable_web_search": true,
  "enable_local_tools": true
}
```
