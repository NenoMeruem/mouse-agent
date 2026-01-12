# 🚀 Quick Reference Card

## ⚡ 30 Second Start

```bash
# 1. Set API key
export OPENAI_API_KEY="sk-..."

# 2. Build
cd ~/Workspace/Start-up/mouse-agent
go build ./cmd/prompt-agent

# 3. Create config
mkdir -p ~/.prompt-agent
echo 'engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
ui:
  output: stdout' > ~/.prompt-agent/config.yaml

# 4. Create prompt
./prompt-agent prompt add \
  --id test \
  --name "Test" \
  --template "{{selection}}" \
  --engine openai

# 5. Run
echo "hello world" | pbcopy
./prompt-agent run test
```

---

## 🎯 Most Used Commands

```bash
# List all prompts
./prompt-agent prompt list

# Show one prompt
./prompt-agent prompt show <id>

# Create new prompt
./prompt-agent prompt add --id <id> --name <name> --template <template> --engine openai

# Delete prompt
./prompt-agent prompt delete <id>

# Run prompt (no streaming)
./prompt-agent run <id> --dry-run

# Run prompt (with streaming)
./prompt-agent run <id>

# Beautiful UI
./prompt-agent run <id> --output tui

# Cancel: Press Ctrl+C
```

---

## 📝 Template Examples

### Explain Code
```bash
./prompt-agent prompt add \
  --id explain \
  --template "Explain this code:\n{{selection}}" \
  --engine openai
```

### Translate
```bash
./prompt-agent prompt add \
  --id translate \
  --template "Translate to English:\n{{selection}}" \
  --engine gemini
```

### Summarize
```bash
./prompt-agent prompt add \
  --id summarize \
  --template "Summarize in 3 sentences:\n{{selection}}" \
  --engine openai
```

### Fix Bugs
```bash
./prompt-agent prompt add \
  --id fix \
  --template "Fix this code:\n{{selection}}" \
  --engine openai
```

---

## 🔑 API Keys

### OpenAI
```bash
export OPENAI_API_KEY="sk-proj-..."
# Get from: https://platform.openai.com/api-keys
```

### Gemini
```bash
export GEMINI_API_KEY="AIzaSyD..."
# Get from: https://makersuite.google.com/app/apikey
```

---

## ⚙️ Config File Location

```
~/.prompt-agent/config.yaml
```

### Minimal Config
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
ui:
  output: stdout
```

### Full Config
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
  theme: dark
default_engine: openai
```

---

## 🐛 Common Errors

| Error | Fix |
|-------|-----|
| `401 Unauthorized` | Check API key: `echo $OPENAI_API_KEY` |
| `429 Rate Limited` | Wait 10 seconds then retry |
| `Config not found` | Create: `mkdir -p ~/.prompt-agent` |
| `Nothing happens` | Check: `./prompt-agent run <id> --dry-run` |
| `TUI broken` | Use: `output: stdout` in config |

---

## 💡 Pro Tips

### Alias
```bash
# Add to ~/.zshrc
alias exp='./prompt-agent run explain'
alias sum='./prompt-agent run summarize'
```

### One-Liner
```bash
# Copy, explain, copy result
pbpaste | ./prompt-agent run explain | pbcopy
```

### Piping
```bash
# Cat file, explain, save result
cat script.py | ./prompt-agent run explain > result.txt
```

---

## 📊 File Locations

```
Project:   /Users/macos/Workspace/Start-up/mouse-agent
Config:    ~/.prompt-agent/config.yaml
Logs:      ~/.prompt-agent/logs/app.log
Database:  ~/.prompt-agent/prompts.db
Binary:    ./prompt-agent
```

---

## 🎮 Usage Flow

```
1. Copy text to clipboard
   ↓
2. Run: ./prompt-agent run <prompt-id>
   ↓
3. AI streams response
   ↓
4. Response complete
   ↓
5. (Optional) Copy result with: | pbcopy
```

---

## 🌟 Supported Engines

- ✅ **OpenAI**: gpt-4, gpt-4-mini, gpt-3.5-turbo
- ✅ **Google Gemini**: gemini-2.5-flash-lite
- 🔜 **Claude**: Coming Phase 7
- 🔜 **Local LLM**: Coming Phase 8

---

## 📞 Help

```bash
./prompt-agent --help              # All commands
./prompt-agent run --help          # Run help
./prompt-agent prompt --help       # Prompt help
./prompt-agent daemon --help       # Daemon help
./prompt-agent version             # Version info
```

---

## ✅ Setup Checklist

```bash
# 1. Build
[ ] go build ./cmd/prompt-agent

# 2. API Key
[ ] export OPENAI_API_KEY="sk-..."

# 3. Config
[ ] mkdir -p ~/.prompt-agent
[ ] cat > ~/.prompt-agent/config.yaml (with engines)

# 4. Test
[ ] ./prompt-agent run --help
[ ] echo "test" | pbcopy
[ ] ./prompt-agent run test --dry-run

# 5. Create Prompt
[ ] ./prompt-agent prompt add --id <id> ...

# 6. Run
[ ] ./prompt-agent run <id>
```

✅ **Done!**

---

**Print this and keep nearby! 📌**
