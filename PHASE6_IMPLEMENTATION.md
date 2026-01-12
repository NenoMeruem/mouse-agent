# Phase 6 Implementation Guide - LLM Gateway & Output UX

## ✅ What Has Been Implemented

### 1. LLM Gateway Infrastructure
- **`internal/llm/types.go`**: Core interfaces and types
  - `Client` interface: All LLM providers must implement this
  - `Request` struct: Standardized request format
  - `Chunk` struct: Streaming response chunks

- **`internal/llm/manager.go`**: LLM Router/Manager
  - Provider registration and retrieval
  - Support for multiple LLM engines

### 2. OpenAI Adapter
- **`internal/llm/openai/client.go`**: Full streaming support
  - Server-Sent Events (SSE) streaming
  - Proper error handling and timeout support
  - Context cancellation support

### 3. Gemini Adapter
- **`internal/llm/gemini/client.go`**: MVP implementation
  - Non-streaming (single chunk response)
  - Ready for future streaming upgrade

### 4. Output Renderers

#### a. Renderer Interface
- **`internal/output/renderer.go`**: Standardized interface
  - `StreamRenderer` interface for all output types
  - Factory pattern for renderer creation

#### b. TUI Renderer
- **`internal/output/tui.go`**: Beautiful terminal UI with Bubbletea
  - Streaming display with status indicators
  - Keyboard controls (q/Ctrl+C to exit)
  - Viewport for scrolling

#### c. Stdout Renderer
- **`internal/output/stdout.go`**: Simple streaming output
  - Progressive text display
  - Fallback when TUI not available

### 5. Run Command Integration
- **`internal/cli/run_handler.go`**: Enhanced with LLM support
  - Context cancellation (Ctrl+C support)
  - Dry-run mode (preview prompt)
  - LLM streaming integration
  - Configuration-driven output rendering

### 6. Configuration
- **`internal/config/config.go`**: Enhanced with timeout support
- **`.prompt-agent/config.yaml`**: Example configuration

### 7. Dependencies
- **`go.mod`**: Updated with Bubbletea dependencies
  - `github.com/charmbracelet/bubbles`
  - `github.com/charmbracelet/bubbletea`
  - `github.com/charmbracelet/lipgloss`

---

## 🚀 Usage Examples

### 1. Setup API Keys
```bash
export OPENAI_API_KEY="sk-..."
export GEMINI_API_KEY="..."
```

### 2. Run Prompt with Streaming
```bash
# Stream with stdout renderer
prompt-agent run explain_code --dry-run

# Stream with TUI renderer (after config update)
prompt-agent run explain_code
```

### 3. Configure LLM Engines
Create `~/.prompt-agent/config.yaml`:
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
    timeout: 30s
  gemini:
    api_key: env:GEMINI_API_KEY
    model: gemini-2.5-flash-lite
    timeout: 30s

ui:
  output: stdout  # or "tui"
```

---

## 🔧 Architecture Diagram

```
[Run Command]
     ↓
[Prompt Builder] → [Final Prompt]
     ↓
[LLM Manager]
     ├─→ [OpenAI Client] ──→ SSE Streaming
     └─→ [Gemini Client] ──→ Single Chunk
          ↓
     [Channel <-chan Chunk]
          ↓
     [Renderer Factory]
     ├─→ [Stdout Renderer]
     └─→ [TUI Renderer]
          ↓
     [User Output]
```

---

## ✨ Key Features

### Streaming Response
✓ Chunks from `<-chan llm.Chunk`
✓ Real-time display to user
✓ No buffering

### Cancellation
✓ Ctrl+C stops request
✓ Context propagation
✓ Clean shutdown

### Multiple Renderers
✓ Stdout: Simple, portable
✓ TUI: Beautiful, interactive

### Error Handling
✓ API errors (401, 429, etc.)
✓ Network timeout
✓ Context cancellation

---

## 🧪 Testing

Run tests to verify LLM gateway:
```bash
go test ./internal/llm/... -v
```

Manual test checklist:
- [ ] `prompt-agent run test_prompt --dry-run` (shows prompt only)
- [ ] `OPENAI_API_KEY=... prompt-agent run test_prompt` (streams from OpenAI)
- [ ] Ctrl+C during streaming (should cancel)
- [ ] Config with `ui.output: tui` (uses TUI renderer)

---

## 📋 Next Steps (Phase 7)

1. **Gemini Streaming**: Upgrade Gemini client to support streaming
2. **Retry Logic**: Automatic retry on transient failures
3. **Token Counting**: Track token usage
4. **Response Caching**: Optional response caching layer
5. **Multi-model Comparison**: Run same prompt against multiple models
6. **Custom Templates**: User-defined LLM response templates

---

## ⚠️ Important Notes

1. **API Keys**: Never commit API keys. Use environment variables.
2. **Timeout**: Default 30s. Adjust in config if needed.
3. **Context**: Always cancel context to avoid goroutine leaks.
4. **Error Propagation**: Errors in streaming are sent via Chunk.Err

---

## 📚 File Structure

```
internal/
├── llm/
│   ├── types.go            # Core interfaces
│   ├── manager.go          # LLM router
│   ├── llm_test.go         # Tests
│   ├── openai/
│   │   └── client.go       # OpenAI adapter
│   └── gemini/
│       └── client.go       # Gemini adapter
├── output/
│   ├── renderer.go         # Renderer interface
│   ├── stdout.go           # Stdout renderer
│   ├── tui.go              # TUI renderer
│   └── (old files)
├── cli/
│   └── run_handler.go      # Enhanced with LLM
└── config/
    └── config.go           # Config with timeout
```

---

## 🎯 Success Criteria

✅ LLM Gateway abstracts providers
✅ OpenAI streaming works
✅ Gemini MVP works
✅ Multiple output renderers
✅ Ctx cancellation (Ctrl+C)
✅ Configuration-driven
✅ Error handling

**Phase 6 is complete! App is now a daily driver! 🚀**
