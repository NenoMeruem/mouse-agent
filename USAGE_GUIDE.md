# 📖 Mouse Agent - Hướng Dẫn Sử Dụng Đầy Đủ

**Version**: Phase 6  
**Last Updated**: January 10, 2026  
**Language**: Tiếng Việt + English

---

## 📚 Mục Lục

1. [Cài Đặt Nhanh (5 phút)](#cài-đặt-nhanh)
2. [Các Lệnh Cơ Bản](#các-lệnh-cơ-bản)
3. [Cấu Hình](#cấu-hình)
4. [Các Ví Dụ Thực Tế](#các-ví-dụ-thực-tế)
5. [Sử Dụng Advanced](#sử-dụng-advanced)
6. [Troubleshooting](#troubleshooting)

---

## ⚡ Cài Đặt Nhanh

### Bước 1: Chuẩn Bị

```bash
# 1a. Clone hoặc vào project
cd /Users/macos/Workspace/Start-up/mouse-agent

# 1b. Chọn LLM provider (OpenAI hoặc Gemini)
# Option A: OpenAI
export OPENAI_API_KEY="sk-proj-..."

# Option B: Google Gemini
export GEMINI_API_KEY="AIzaSyD..."
```

### Bước 2: Build

```bash
go build ./cmd/prompt-agent
```

**Kết quả**:
```bash
$ ls -la prompt-agent
-rwxr-xr-x  prompt-agent  (executable file)
```

### Bước 3: Tạo Config

```bash
# Tạo thư mục cấu hình
mkdir -p ~/.prompt-agent

# Tạo file config.yaml
cat > ~/.prompt-agent/config.yaml << 'EOF'
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
  output: stdout  # hoặc "tui" cho giao diện đẹp
EOF

cat ~/.prompt-agent/config.yaml
```

### Bước 4: Test

```bash
# Kiểm tra version
./prompt-agent version

# Kiểm tra API connection (dry-run, không gọi API)
./prompt-agent run test --dry-run

# ✅ Sẽ in ra: [Dry-run mode - no API call]
```

✅ **Xong! Sẵn sàng dùng!**

---

## 🎯 Các Lệnh Cơ Bản

### 1️⃣ Quản Lý Prompts

#### Xem danh sách prompts
```bash
./prompt-agent prompt list

# Output:
# ID              Name                Created
# explain_code    Explain Code        2025-01-10
# summarize       Summarize Text      2025-01-10
# translate_en    Translate to EN     2025-01-10
```

#### Xem chi tiết một prompt
```bash
./prompt-agent prompt show explain_code

# Output:
# ID:       explain_code
# Name:     Explain Code
# Template: Explain this code:\n{{selection}}
# Engine:   openai
# Created:  2025-01-10
```

#### Thêm prompt mới
```bash
./prompt-agent prompt add \
  --id explain_code \
  --name "Explain Code" \
  --template "Explain this code in detail:\n{{selection}}" \
  --engine openai
```

**Các biến có sẵn**:
- `{{selection}}` - Text được select
- `{{clipboard}}` - Nội dung clipboard
- `{{time}}` - Thời gian hiện tại
- `{{date}}` - Ngày hiện tại

#### Xóa prompt
```bash
./prompt-agent prompt delete explain_code
```

---

### 2️⃣ Chạy Prompts

#### Chạy với clipboard
```bash
# 1. Copy text to clipboard
echo "Python is a programming language" | pbcopy

# 2. Run prompt
./prompt-agent run explain_code

# Output:
# 🤖 AI: [streaming response here...]
# ✓ Done
```

#### Chạy dry-run (preview, không gọi API)
```bash
./prompt-agent run explain_code --dry-run

# Output:
# Template: Explain this code in detail:
# Selection: [Python is a programming language]
# [Dry-run mode - no API call]
```

#### Chạy với output beautify (TUI)
```bash
# 1. Update config
cat > ~/.prompt-agent/config.yaml << 'EOF'
ui:
  output: tui  # ← Change this
EOF

# 2. Run
./prompt-agent run explain_code

# Output: Beautiful terminal UI with streaming
```

#### Hủy (Ctrl+C)
```bash
./prompt-agent run explain_code
# Press: Ctrl+C → Cancel immediately
```

---

### 3️⃣ Daemon Mode

#### Start daemon
```bash
./prompt-agent daemon start

# Output:
# ✓ Daemon started (PID: 12345)
# Listening on: 127.0.0.1:8080
```

#### Stop daemon
```bash
./prompt-agent daemon stop

# Output:
# ✓ Daemon stopped
```

#### Check status
```bash
./prompt-agent daemon status

# Output:
# Status: Running
# PID: 12345
# Uptime: 2h 15m
```

---

## ⚙️ Cấu Hình

### Config File Location
```bash
~/.prompt-agent/config.yaml
```

### Ví Dụ Config Đầy Đủ

```yaml
# ===== Engines =====
engines:
  # OpenAI Configuration
  openai:
    api_key: env:OPENAI_API_KEY      # From env var
    model: gpt-4-mini                # Model name
    timeout: 30s                     # Timeout
    temperature: 0.7                 # Optional
    max_tokens: 2000                 # Optional
  
  # Google Gemini Configuration
  gemini:
    api_key: env:GEMINI_API_KEY
    model: gemini-2.5-flash-lite
    timeout: 30s

# ===== UI Settings =====
ui:
  output: stdout          # "stdout" or "tui"
  theme: dark            # Optional: "light" or "dark"
  colors: enabled        # Optional: true/false

# ===== Default Engine =====
default_engine: openai   # Falls back to this

# ===== Logging =====
logging:
  level: info            # "debug", "info", "warn", "error"
  file: ~/.prompt-agent/logs/app.log
```

### Cách Set API Key

#### Option 1: Environment Variable (Recommended)
```bash
# Tạm thời
export OPENAI_API_KEY="sk-proj-..."

# Vĩnh viễn (thêm vào ~/.zshrc)
echo 'export OPENAI_API_KEY="sk-proj-..."' >> ~/.zshrc
source ~/.zshrc
```

#### Option 2: Trực tiếp trong Config
```yaml
engines:
  openai:
    api_key: sk-proj-...  # Direct (not recommended)
```

#### Option 3: Config File (Recommended)
```yaml
engines:
  openai:
    api_key: env:OPENAI_API_KEY  # Read from env
```

---

## 💡 Các Ví Dụ Thực Tế

### Ví Dụ 1: Giải Thích Code

```bash
# 1. Create prompt
./prompt-agent prompt add \
  --id explain \
  --name "Explain Code" \
  --template "Giải thích code này:\n{{selection}}" \
  --engine openai

# 2. Copy code
cat << 'EOF' | pbcopy
def fib(n):
    if n <= 1:
        return n
    return fib(n-1) + fib(n-2)
EOF

# 3. Run
./prompt-agent run explain

# Output:
# 🤖 AI: Đây là hàm tính Fibonacci:
# - Input: số n
# - Nếu n <= 1: trả về n
# - Nếu không: trả về fib(n-1) + fib(n-2)
# - Độ phức tạp: O(2^n)
# ✓ Done
```

### Ví Dụ 2: Dịch Văn Bản

```bash
# 1. Create prompt
./prompt-agent prompt add \
  --id translate_en \
  --name "Translate to English" \
  --template "Dịch sang Tiếng Anh:\n{{selection}}" \
  --engine gemini

# 2. Copy text
echo "Xin chào thế giới" | pbcopy

# 3. Run
./prompt-agent run translate_en

# Output:
# 🤖 AI: Hello world
# ✓ Done
```

### Ví Dụ 3: Tóm Tắt

```bash
# 1. Create
./prompt-agent prompt add \
  --id summarize \
  --name "Summarize" \
  --template "Tóm tắt điểm chính trong 3 câu:\n{{selection}}" \
  --engine openai

# 2. Copy long text
pbcopy << 'EOF'
Machine learning is a subset of artificial intelligence that enables 
systems to learn and improve from experience without being explicitly 
programmed. It focuses on developing algorithms that can access data 
and use it to learn for themselves...
EOF

# 3. Run
./prompt-agent run summarize

# Output:
# 🤖 AI: Machine learning là một nhánh của AI cho phép các hệ thống 
# học từ dữ liệu và cải thiện từ kinh nghiệm. Nó phát triển các 
# thuật toán có thể tự học từ dữ liệu. ML được sử dụng rộng rãi 
# trong các ứng dụng thực tế.
# ✓ Done
```

### Ví Dụ 4: Lỗi Sửa

```bash
# 1. Create
./prompt-agent prompt add \
  --id fix_error \
  --name "Fix Error" \
  --template "Sửa lỗi trong code này:\n{{selection}}" \
  --engine openai

# 2. Copy buggy code
echo "for i in range(10)\n    print(i)" | pbcopy

# 3. Run
./prompt-agent run fix_error

# Output:
# 🤖 AI: Lỗi: thiếu dấu : sau for
# Sửa: for i in range(10):
# ✓ Done
```

---

## 🚀 Sử Dụng Advanced

### 1. Tự Động Hóa

#### Tạo Alias
```bash
# Thêm vào ~/.zshrc
alias explain='pbpaste | xclip -i | ./prompt-agent run explain'

# Sử dụng
explain
```

#### Shell Script
```bash
#!/bin/bash
# save as: explain.sh

SELECTION=$(pbpaste)
echo "$SELECTION" | ./prompt-agent run explain_code
```

### 2. Integration với Workflow

#### Copy + Run + Paste Result
```bash
# Copy code, run, auto-paste result
pbpaste | ./prompt-agent run explain_code | pbcopy
```

#### Chain Prompts
```bash
# Dịch → Tóm tắt → Copy
pbpaste | \
  ./prompt-agent run translate_en | \
  ./prompt-agent run summarize | \
  pbcopy
```

### 3. Batch Processing

```bash
#!/bin/bash
# Process multiple files

for file in *.py; do
  echo "Processing: $file"
  cat "$file" | ./prompt-agent run explain_code
done
```

---

## 🔧 Troubleshooting

### ❌ Error: API error 401

**Nguyên nhân**: API key sai hoặc hết hạn

**Cách fix**:
```bash
# 1. Kiểm tra API key
echo $OPENAI_API_KEY

# 2. Copy mới từ:
# OpenAI: https://platform.openai.com/api-keys
# Gemini: https://makersuite.google.com/app/apikey

# 3. Set lại
export OPENAI_API_KEY="sk-proj-..."

# 4. Test
./prompt-agent run test --dry-run
```

### ❌ Error: API error 429

**Nguyên nhân**: Rate limit (quá nhiều request)

**Cách fix**:
```bash
# 1. Chờ vài giây
sleep 10

# 2. Thử lại
./prompt-agent run explain

# 3. Hoặc: Nâng cấp plan
# https://platform.openai.com/account/billing/overview
```

### ❌ Error: Config not found

**Nguyên nhân**: File config chưa tồn tại

**Cách fix**:
```bash
# Tạo config
mkdir -p ~/.prompt-agent
cat > ~/.prompt-agent/config.yaml << 'EOF'
engines:
  openai:
    api_key: env:OPENAI_API_KEY
    model: gpt-4-mini
ui:
  output: stdout
EOF

# Verify
cat ~/.prompt-agent/config.yaml
```

### ❌ Không gì xảy ra

**Nguyên nhân**: Đang dùng `--dry-run` hoặc clipboard trống

**Cách fix**:
```bash
# 1. Kiểm tra clipboard
pbpaste

# 2. Copy text trước
echo "test" | pbcopy

# 3. Bỏ --dry-run
./prompt-agent run explain  # ← Remove --dry-run

# 4. Kiểm tra API key
echo $OPENAI_API_KEY
```

### ❌ TUI Looks Broken

**Nguyên nhân**: Terminal không support TUI

**Cách fix**:
```bash
# Đổi sang stdout
cat > ~/.prompt-agent/config.yaml << 'EOF'
ui:
  output: stdout  # ← Change to stdout
EOF

./prompt-agent run explain
```

---

## 📱 Command Reference

```bash
# ===== Prompts =====
./prompt-agent prompt list              # List all
./prompt-agent prompt show <id>         # Show details
./prompt-agent prompt add \             # Create
  --id <id> \
  --name <name> \
  --template <template> \
  --engine <engine>
./prompt-agent prompt delete <id>       # Delete

# ===== Run =====
./prompt-agent run <id>                 # Run with streaming
./prompt-agent run <id> --dry-run       # Preview only
./prompt-agent run <id> --output tui    # Beautiful UI

# ===== Daemon =====
./prompt-agent daemon start             # Start
./prompt-agent daemon stop              # Stop
./prompt-agent daemon status            # Check status

# ===== Other =====
./prompt-agent version                  # Show version
./prompt-agent --help                   # Help
```

---

## 🎨 Template Variables

| Variable | Ý Nghĩa | Ví Dụ |
|----------|---------|-------|
| `{{selection}}` | Text từ clipboard | Code, text |
| `{{clipboard}}` | Same as selection | Same |
| `{{time}}` | Giờ hiện tại | 14:30:45 |
| `{{date}}` | Ngày hiện tại | 2025-01-10 |
| `{{random}}` | Random string | abc123xyz |
| `{{env:VAR}}` | Environment variable | env:USER |

**Ví dụ**:
```bash
./prompt-agent prompt add \
  --id log \
  --template "[{{date}} {{time}}] {{selection}}" \
  --engine openai
```

---

## 🌍 Supported LLM Engines

| Engine | Model | Status | Speed |
|--------|-------|--------|-------|
| OpenAI | gpt-4-mini, gpt-4 | ✅ | Fast (streaming) |
| Google Gemini | gemini-2.5-flash-lite | ✅ | Medium |
| Claude | (Phase 7) | 🔜 | - |
| Local LLM | (Phase 8) | 🔜 | - |

---

## 💾 Keyboard Shortcuts

| Key | Tác Dụng |
|-----|---------|
| `Ctrl+C` | Cancel (hủy streaming) |
| `Ctrl+D` | Exit |
| `↑` / `↓` | History (trong CLI) |

---

## 📊 Performance Tips

### Tip 1: Chọn Model Đúng
```bash
# Fast & cheap (gpt-4-mini)
--engine openai --model gpt-4-mini

# Better quality (gpt-4)
--engine openai --model gpt-4
```

### Tip 2: Set Timeout
```bash
# Quick timeout
timeout: 5s   # Nhanh nhưng có thể timeout

# Safe timeout
timeout: 30s  # Cân bằng
```

### Tip 3: Cache Results
```bash
# Lưu output
./prompt-agent run explain > result.txt
```

---

## 🆘 Tìm Giúp Đỡ

### View Logs
```bash
tail -f ~/.prompt-agent/logs/app.log
```

### Check Version
```bash
./prompt-agent version
```

### Debug Mode
```bash
# Set log level
LOGLEVEL=debug ./prompt-agent run explain
```

### Contact Support
- 📝 Check: [PHASE6_QUICK_START.md](PHASE6_QUICK_START.md)
- 🐛 Bugs: Check issues
- 💬 Questions: Check documentation

---

## ✅ Checklist Bắt Đầu

- [ ] Build app: `go build ./cmd/prompt-agent`
- [ ] Set API key: `export OPENAI_API_KEY=...`
- [ ] Create config: `~/.prompt-agent/config.yaml`
- [ ] Test dry-run: `./prompt-agent run test --dry-run`
- [ ] Create first prompt: `prompt add --id first ...`
- [ ] Run prompt: `./prompt-agent run first`
- [ ] Copy to clipboard: `pbcopy` / `xclip` / `Get-Clipboard`
- [ ] Check output: See streaming response

---

## 🎉 Ready to Go!

```bash
cd /Users/macos/Workspace/Start-up/mouse-agent
go build ./cmd/prompt-agent
export OPENAI_API_KEY="sk-..."
./prompt-agent run --help

# ✅ Bắt đầu sử dụng!
```

---

**Last Updated**: January 10, 2026  
**Version**: Phase 6  
**Status**: ✅ Production Ready
