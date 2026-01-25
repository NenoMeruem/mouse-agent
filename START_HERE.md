# 🎯 Mouse Agent - Complete Usage Summary

**What is Mouse Agent?** AI-powered text processor that streams real-time responses from OpenAI or Google Gemini directly in your terminal.

---

## ⚡ 60-Second Start

```bash
# 1. API Key
export OPENAI_API_KEY="sk-proj-..."

# 2. Build
cd ~/Workspace/Start-up/mouse-agent && go build ./cmd/prompt-agent

# 3. Config (create ~/.prompt-agent/config.yaml)
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
ui:
  output: stdout

# 4. Create Prompt
./prompt-agent prompt add \
  --id hello \
  --template "Say hello to {{selection}}" \
  --engine gemini

# 5. Use It
echo "world" | pbcopy
./prompt-agent run hello

# OUTPUT:
# 🤖 AI: Hello world! How can I help you today?
# ✓ Done
```

**✅ Done!**

---

## 🎮 How It Works

```
1. Copy text to clipboard (Cmd+C)
   ↓
2. Run: ./prompt-agent run my_prompt
   ↓
3. AI streams response in real-time
   ↓
4. See beautiful output (or copy result)
```

---

## 📚 Documentation Guide

| Want To... | Read This | Time |
|-----------|-----------|------|
| Get started ASAP | [QUICK_REFERENCE.md](QUICK_REFERENCE.md) | 2 min |
| Learn everything | [USAGE_GUIDE.md](USAGE_GUIDE.md) | 20 min |
| See diagrams | [VISUAL_GUIDE.md](VISUAL_GUIDE.md) | 10 min |
| Browse all docs | [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) | 5 min |
| Setup Gemini | [GEMINI_REAL_TEST_SETUP.md](GEMINI_REAL_TEST_SETUP.md) | 5 min |

---

## 🔥 Most Used Commands

```bash
# Create a prompt
./prompt-agent prompt add \
  --id read \
  --template "Bạn là một senior software egnineer. Hãy dịch đoạn văn sau sao cho dễ hiểu: {{selection}}" \
  --engine gemini

./prompt-agent prompt add \
  --id exp \
  --template "Bạn là một senior software egnineer. Hãy phân tích mã code nêu ra các ưu điểm, nhược điểm và gợi ý improve nếu có: {{selection}}" \
  --engine gemini

# List all prompts
./prompt-agent prompt list

# Run a prompt
echo "text" | pbcopy
./prompt-agent run <name>

# Delete a prompt
./prompt-agent prompt delete <name>

# Preview (no API call)
./prompt-agent run <name> --dry-run

# Beautiful UI
./prompt-agent run <name> --output tui
```

---

## 🎨 Real Examples

### Example 1: Explain Code
```bash
./prompt-agent prompt add \
  --id explain \
  --template "Explain this code:\n{{selection}}" \
  --engine openai

# Copy code, then:
./prompt-agent run explain
```

### Example 2: Translate
```bash
./prompt-agent prompt add \
  --id translate \
  --template "Translate to English:\n{{selection}}" \
  --engine gemini

# Copy text, then:
./prompt-agent run translate
```

### Example 3: Fix Bugs
```bash
./prompt-agent prompt add \
  --id fix \
  --template "Fix this code:\n{{selection}}" \
  --engine openai

# Copy buggy code, then:
./prompt-agent run fix
```

---

## 🔑 Getting API Keys

### OpenAI
1. Visit: https://platform.openai.com/api-keys
2. Create new key
3. Set: `export OPENAI_API_KEY="sk-proj-..."`

### Google Gemini
1. Visit: https://makersuite.google.com/app/apikey
2. Create new key
3. Set: `export GEMINI_API_KEY="AIzaSyD..."`

---

## ⚙️ Configuration

### Minimal (~/.prompt-agent/config.yaml)
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
ui:
  output: stdout
```

### Full
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
    timeout: 30s
  gemini:
    api_key: env:GEMINI_API_KEY
    model: gemini-2.5-flash-lite
ui:
  output: tui  # or "stdout"
  theme: dark
default_engine: openai
```

---

## 🧩 Template Variables

| Variable | Meaning | Example |
|----------|---------|---------|
| `{{selection}}` | Your clipboard | "Hello world" |
| `{{date}}` | Today | "2025-01-10" |
| `{{time}}` | Now | "14:30:45" |
| `{{random}}` | Random text | "abc123xyz" |
| `{{env:VAR}}` | Env variable | env:USER |

**Example**:
```bash
./prompt-agent prompt add \
  --id log \
  --template "[{{date}} {{time}}] {{selection}}" \
  --engine openai
```

---

## 🚀 Advanced Usage

### Bash Alias
```bash
echo 'alias explain="./prompt-agent run explain"' >> ~/.zshrc
source ~/.zshrc

# Now use:
explain
```

### Copy → Process → Paste
```bash
pbpaste | ./prompt-agent run explain | pbcopy
```

### Chain Multiple Prompts
```bash
pbpaste | \
  ./prompt-agent run translate_en | \
  ./prompt-agent run summarize | \
  pbcopy
```

### Batch Process Files
```bash
for file in *.py; do
  cat "$file" | ./prompt-agent run explain_code
done
```

---

## 🐛 Troubleshooting

| Problem | Solution |
|---------|----------|
| `401 Unauthorized` | Check API key: `echo $OPENAI_API_KEY` |
| `429 Rate Limited` | Wait 10 seconds, try again |
| `Config not found` | Create: `mkdir -p ~/.prompt-agent` |
| `Nothing happens` | Use: `./prompt-agent run <id> --dry-run` |
| `TUI broken` | Use: `output: stdout` in config |

**For more**, see [USAGE_GUIDE.md - Troubleshooting](USAGE_GUIDE.md#-troubleshooting)

---

## 📊 Performance

| Operation | Time |
|-----------|------|
| Build | ~5 seconds |
| Create prompt | <1 second |
| With API | 1-3 seconds |
| Cancel (Ctrl+C) | Instant |

---

## ✨ Features

✅ **LLM Support**: OpenAI (streaming), Gemini (non-streaming)  
✅ **Output**: Beautiful TUI or simple stdout  
✅ **Variables**: Dynamic template variables  
✅ **Config**: YAML-based settings  
✅ **Database**: SQLite prompt storage  
✅ **Streaming**: Real-time response rendering  
✅ **Timeout**: Configurable request timeout  
✅ **Error Handling**: Comprehensive error messages  

---

## 🎯 Workflow Examples

### Daily Workflow

```
Morning:
  1. Copy code snippets from PR review
  2. Run: ./prompt-agent run explain
  3. Understand changes
  
Afternoon:
  1. Copy error messages
  2. Run: ./prompt-agent run fix
  3. Get suggestions
  
Evening:
  1. Copy documentation
  2. Run: ./prompt-agent run summarize
  3. Quick review
```

### Writing Workflow

```
1. Write draft (copy to clipboard)
2. Run: ./prompt-agent run improve_grammar
3. Copy corrected version
4. Paste into document
5. Continue writing
```

### Translation Workflow

```
1. Copy Vietnamese text
2. Run: ./prompt-agent run translate_en
3. Get English translation
4. Copy and paste
```

---

## 📱 Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Cmd+C | Copy to clipboard |
| Cmd+V | Paste from clipboard |

---

## 🌐 Supported Engines

| Engine | Status | Speed | Notes |
|--------|--------|-------|-------|
| Gemini | ✅ | Medium | Single response |
| Local | 🔜 | - | Phase 8 |

---

## 📈 Next Steps

1. **Setup**: Follow [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
2. **Learn**: Read [USAGE_GUIDE.md](USAGE_GUIDE.md)
3. **Create**: Make your first custom prompt
4. **Integrate**: Add to your workflow with aliases
5. **Explore**: Try advanced features

---

## 💡 Pro Tips

**Tip 1**: Prefix your selection with instructions
```bash
# Copy: "Code: def hello(): print('hi')"
./prompt-agent run explain
# AI knows it's code and explains better
```

**Tip 3**: Combine with grep for batch processing
```bash
grep "error" logfile.txt | ./prompt-agent run analyze
```

**Tip 4**: Save outputs for reference
```bash
./prompt-agent run explain > explanations.txt
```

---

## ✅ Checklist

```
Setup Checklist:
  [ ] Get API key from OpenAI/Gemini
  [ ] Build: go build ./cmd/prompt-agent
  [ ] Create config: ~/.prompt-agent/config.yaml
  [ ] Set API key: export OPENAI_API_KEY=...
  [ ] Test dry-run: ./prompt-agent run test --dry-run
  [ ] Create first prompt
  [ ] Run and see result
  [ ] Copy result to clipboard (optional)
  
Usage Checklist:
  [ ] Copy text to clipboard
  [ ] Run: ./prompt-agent run <id>
  [ ] Read streaming response
  [ ] Copy result (if needed)
  [ ] Continue working
```

---

## 🎊 Ready!

You're all set! Start using Mouse Agent:

```bash
# Copy your text
echo "Explain machine learning" | pbcopy

# Run your prompt
./prompt-agent run explain

# 🤖 AI: Machine learning is a subset of...
```

**Happy coding! 🚀**

---
