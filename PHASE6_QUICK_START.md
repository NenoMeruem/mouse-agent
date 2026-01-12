# Phase 6 Quick Start Guide 🚀

## 5-Minute Setup

### 1. Get API Keys
```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# OR Google Gemini
export GEMINI_API_KEY="..."
```

### 2. Create Config
Create `~/.prompt-agent/config.yaml`:
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
    timeout: 30s

ui:
  output: stdout
```

### 3. Build & Test
```bash
cd /Users/macos/Workspace/Start-up/mouse-agent

# Build
go build ./cmd/prompt-agent

# Create a test prompt
./prompt-agent prompt add \
  --id test_llm \
  --name "Test LLM" \
  --template "Explain this in one sentence: {{selection}}" \
  --engine openai

# Run with dry-run (no API call)
./prompt-agent run test_llm --dry-run

# Run with real API
echo "machine learning" | pbcopy  # macOS
./prompt-agent run test_llm
```

---

## Architecture Overview

```
┌─────────────────────────────────────────┐
│  User: prompt-agent run my_prompt       │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  1. Load prompt from DB                 │
│  2. Build template with variables       │
│  3. Select LLM provider (OpenAI/Gemini) │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  LLM Client.Stream(ctx, request)        │
│  ├─ OpenAI: SSE streaming               │
│  └─ Gemini: Single chunk                │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  Renderer.RenderStream(<-chan Chunk)    │
│  ├─ Stdout: Simple output               │
│  └─ TUI: Beautiful UI (Bubbletea)       │
└─────────────────────────────────────────┘
                   ↓
┌─────────────────────────────────────────┐
│  🤖 AI: [streaming text here...]        │
│  ✓ Done                                 │
└─────────────────────────────────────────┘
```

---

## Key Features

| Feature | Status | Notes |
|---------|--------|-------|
| OpenAI streaming | ✅ | SSE, real-time chunks |
| Gemini MVP | ✅ | Non-streaming for now |
| TUI renderer | ✅ | Bubbletea, beautiful UI |
| Stdout renderer | ✅ | Simple, portable |
| Ctrl+C support | ✅ | Cancels mid-stream |
| Error handling | ✅ | API & network errors |
| Config file | ✅ | ~/.prompt-agent/config.yaml |
| Dry-run mode | ✅ | Test without API calls |

---

## Common Commands

### Preview Prompt (No API Call)
```bash
./prompt-agent run my_prompt --dry-run
```

### Stream from OpenAI
```bash
OPENAI_API_KEY=sk-... ./prompt-agent run my_prompt
```

### Stream from Gemini
```bash
GEMINI_API_KEY=... ./prompt-agent run my_prompt
```

### Cancel Mid-Stream
```bash
./prompt-agent run my_prompt
# Then press: Ctrl+C
```

### Use TUI Renderer (Beautiful UI)
```bash
# 1. Update ~/.prompt-agent/config.yaml
ui:
  output: tui

# 2. Run
./prompt-agent run my_prompt
```

---

## File Structure

```
internal/
├── llm/                    # LLM Gateway
│   ├── types.go           # Core interfaces
│   ├── manager.go         # Router
│   ├── openai/client.go   # OpenAI adapter
│   └── gemini/client.go   # Gemini adapter
│
├── output/                 # Output renderers
│   ├── renderer.go        # Interface
│   ├── stdout.go          # Simple renderer
│   └── tui.go             # TUI renderer
│
└── cli/
    └── run_handler.go     # Enhanced with LLM
```

---

## Example Workflow

```bash
# 1. Set API key
export OPENAI_API_KEY="sk-..."

# 2. Build project
cd ~/Workspace/Start-up/mouse-agent
go build ./cmd/prompt-agent

# 3. Create a prompt
./prompt-agent prompt add \
  --id explain \
  --name "Explain Code" \
  --template "Explain this code:\n{{selection}}" \
  --engine openai

# 4. Copy code to clipboard (macOS)
echo "def hello(name):\n  print(f'Hi {name}')" | pbcopy

# 5. Run the prompt (streams from OpenAI)
./prompt-agent run explain

# 6. Output
🚀 Streaming from openai...
🤖 AI: This Python function defines a simple greeting function that:

1. Takes a parameter `name` as input
2. Uses an f-string to create a formatted message
3. Prints the greeting with the provided name

For example, calling `hello("Alice")` would output "Hi Alice".

✓ Done
```

---

## Troubleshooting

### Error: API error 401
```
❌ API error 401: Invalid authentication
```
→ Check your API key is correct in env var or config

### Error: API error 429
```
❌ API error 429: Rate limit exceeded
```
→ Wait a moment and try again (Phase 7 will add retry)

### Nothing happens
→ Check if you're using `--dry-run` flag
→ Check if API key is set: `echo $OPENAI_API_KEY`

### TUI looks broken
→ Try `output: stdout` in config instead

---

## Next Steps

1. **Create prompts**: Use `prompt-agent prompt add`
2. **Configure LLM**: Update `~/.prompt-agent/config.yaml`
3. **Test streaming**: Run a prompt and watch it stream
4. **Integrate with workflow**: Add to shell scripts, aliases, etc.

---

## Files to Read

- **[PHASE6_COMPLETE.md](PHASE6_COMPLETE.md)** - Full implementation details
- **[PHASE6_IMPLEMENTATION.md](PHASE6_IMPLEMENTATION.md)** - Architecture guide
- **[todo/phase6.md](todo/phase6.md)** - Original Phase 6 requirements

---

## Support

### Commands
```bash
./prompt-agent --help                 # Show all commands
./prompt-agent run --help             # Run command help
./prompt-agent prompt --help          # Prompt management
```

### Check Version
```bash
./prompt-agent version
```

### View Prompts
```bash
./prompt-agent prompt list
./prompt-agent prompt show <id>
```

---

**Phase 6 Complete! 🎉 Your app is ready for daily use!**
