# 🎨 Mouse Agent - Visual Guide

## 📊 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    USER WORKFLOW                            │
└─────────────────────────────────────────────────────────────┘

Step 1: Copy to Clipboard
┌──────────────────────┐
│  Select text/code    │
│  Press: Cmd+C        │
│  (On any app)        │
└──────────┬───────────┘
           │
           ↓
Step 2: Run Prompt
┌──────────────────────────────────────────┐
│  $ ./prompt-agent run explain            │
│  (or press hotkey)                       │
└──────────┬───────────────────────────────┘
           │
           ↓
Step 3: Processing
┌──────────────────────────────────────────┐
│  1. Load prompt from database            │
│  2. Get clipboard content                │
│  3. Build template with variables        │
│  4. Choose LLM (OpenAI/Gemini)          │
└──────────┬───────────────────────────────┘
           │
           ↓
Step 4: LLM Streaming
┌──────────────────────────────────────────┐
│  Send request to LLM API                 │
│  Receive streaming response              │
│  Real-time text chunks                   │
└──────────┬───────────────────────────────┘
           │
           ↓
Step 5: Display Output
┌──────────────────────────────────────────┐
│  🤖 AI: [streaming text...]              │
│  [Beautiful rendering in real-time]      │
└──────────┬───────────────────────────────┘
           │
           ↓
Step 6: Done
┌──────────────────────────────────────────┐
│  ✓ Done                                  │
│  (Copy result or continue)               │
└──────────────────────────────────────────┘
```

---

## 🔄 Request Flow

```
USER INPUT
    │
    ├─ Clipboard content
    ├─ Selected prompt ID
    └─ LLM engine choice
           │
           ↓
TEMPLATE PROCESSING
    │
    ├─ Load prompt template
    ├─ Insert {{selection}}
    ├─ Insert {{date}}, {{time}}
    └─ Build final request
           │
           ↓
LLM PROVIDER SELECTION
    │
    ├─ OpenAI → Stream via SSE
    ├─ Gemini → Single response
    └─ Choose based on config
           │
           ↓
API REQUEST
    │
    ├─ Send HTTP request
    ├─ Add API key
    └─ Set timeout
           │
           ↓
RESPONSE HANDLING
    │
    ├─ Receive chunks
    ├─ Parse content
    └─ Handle errors
           │
           ↓
OUTPUT RENDERING
    │
    ├─ stdout → Simple text
    ├─ tui → Beautiful UI
    └─ Real-time display
           │
           ↓
USER OUTPUT
    │
    └─ Final response displayed
```

---

## 🎯 Common Workflows

### Workflow 1: Explain Code

```
┌────────────────────────────────────┐
│ 1. Open code file in editor        │ Cmd+C (copy)
├────────────────────────────────────┤
│ 2. Select function/method          │ Select text
├────────────────────────────────────┤
│ 3. Press hotkey or run             │ ./prompt-agent run explain
├────────────────────────────────────┤
│ 4. AI explains in real-time        │ 🤖 AI: This function...
├────────────────────────────────────┤
│ 5. Read and understand             │ ✓ Done
└────────────────────────────────────┘
```

### Workflow 2: Translate Text

```
┌────────────────────────────────────┐
│ 1. Copy Vietnamese text            │ Cmd+C
├────────────────────────────────────┤
│ 2. Run translate prompt            │ ./prompt-agent run translate
├────────────────────────────────────┤
│ 3. Get English translation         │ 🤖 AI: English text...
├────────────────────────────────────┤
│ 4. Copy result (optional)          │ | pbcopy
└────────────────────────────────────┘
```

### Workflow 3: Fix Bug + Review

```
┌────────────────────────────────────┐
│ 1. Copy buggy code                 │ Cmd+C
├────────────────────────────────────┤
│ 2. Run fix prompt                  │ ./prompt-agent run fix
├────────────────────────────────────┤
│ 3. AI suggests fix                 │ 🤖 AI: Change X to Y...
├────────────────────────────────────┤
│ 4. Run explain on fixed code       │ ./prompt-agent run explain
├────────────────────────────────────┤
│ 5. Understand the fix              │ ✓ Done
└────────────────────────────────────┘
```

---

## 🧩 Component Diagram

```
MOUSE-AGENT
├─ CLI
│  ├─ prompt (manage)
│  ├─ run (execute)
│  └─ daemon (background)
│
├─ LLM Gateway
│  ├─ Manager (router)
│  ├─ OpenAI client
│  │  ├─ Stream (SSE)
│  │  ├─ Error handling
│  │  └─ Timeout
│  └─ Gemini client
│     ├─ Stream (single chunk)
│     ├─ Error handling
│     └─ Timeout
│
├─ Output Renderers
│  ├─ Stdout (simple)
│  │  └─ Plain text output
│  └─ TUI (beautiful)
│     ├─ Bubbletea UI
│     ├─ Spinner animation
│     └─ Colors
│
├─ Storage
│  ├─ SQLite database
│  └─ Config file
│
└─ Utils
   ├─ Selection (clipboard)
   ├─ Trigger (hotkey)
   └─ Logger
```

---

## 📱 State Diagram

```
┌─────────────┐
│    IDLE     │ → Ready to accept input
└────┬────────┘
     │ User runs: ./prompt-agent run <id>
     ↓
┌─────────────────────────┐
│  LOADING_PROMPT         │ → Get prompt from DB
└────┬────────────────────┘
     │ Prompt loaded
     ↓
┌─────────────────────────┐
│  BUILDING_TEMPLATE      │ → Insert variables
└────┬────────────────────┘
     │ Template ready
     ↓
┌─────────────────────────┐
│  CONNECTING_API         │ → HTTP request
└─┬──────────────┬────────┘
  │ Success      │ Error
  ↓              ↓
┌────────┐    ┌──────────────┐
│STREAMING│   │ ERROR DISPLAY│
└────┬────┘   └──────┬───────┘
     │                │
     │ Complete       │ Retry/Cancel
     ↓                ↓
  ┌──────────────────────────┐
  │  DONE                    │ → Output displayed
  └──────────────────────────┘
```

---

## 🚀 Performance Graph

```
STARTUP TIME
└─ 0ms ─ 50ms ─ 100ms ─ 150ms ─ 200ms ─ 250ms
   │      │       │       │       │       │
   Start  Load    Build   API     Stream  Done
   (5ms)  DB      Template Connect (50ms) (10ms)
          (30ms)  (30ms)  (60ms)
```

**Typical Timing**:
- Dry-run: ~100ms (no API call)
- With API: ~1-3s (including network + streaming)
- Depends on: Network speed, LLM response time

---

## 🎮 Keyboard Layout

```
MacBook Keyboard
┌─────────────────────────────────────┐
│  Cmd+C (Copy to clipboard)          │ ← Use for selection
├─────────────────────────────────────┤
│  Run: ./prompt-agent run <id>       │ ← Terminal command
├─────────────────────────────────────┤
│  Ctrl+C (Cancel streaming)          │ ← During response
├─────────────────────────────────────┤
│  Tab (Autocomplete)                 │ ← In terminal
└─────────────────────────────────────┘
```

---

## 📊 Configuration Hierarchy

```
DEFAULT SETTINGS
        │
        ├─ Engine: OpenAI
        ├─ Model: gpt-4-mini
        ├─ Timeout: 30s
        └─ Output: stdout
        
        ↓ (overridden by)

~/.prompt-agent/config.yaml
        │
        ├─ engines:
        │  ├─ openai: {...}
        │  └─ gemini: {...}
        ├─ ui:
        │  ├─ output: tui
        │  └─ theme: dark
        └─ default_engine: openai
        
        ↓ (overridden by)

COMMAND LINE FLAGS
        │
        ├─ --output tui
        ├─ --engine gemini
        ├─ --dry-run
        └─ --timeout 60s
```

**Priority**: CLI Flags > Config File > Defaults

---

## 🌊 Data Flow

```
INPUT
  │
  ├─ Clipboard ({{selection}})
  ├─ Config file (engines, UI)
  ├─ Prompt DB (template, engine)
  └─ Environment variables (API keys)
  
  ↓
  
PROCESSING
  │
  ├─ Template engine (replace variables)
  ├─ Request builder (format for LLM)
  ├─ Router (choose engine)
  └─ Renderer (prepare output)
  
  ↓
  
OUTPUT
  │
  ├─ Streaming chunks
  ├─ Real-time display
  ├─ Error handling
  └─ Final result
```

---

## 🎪 UI Rendering

### Stdout Mode
```
$ ./prompt-agent run explain
🤖 AI: This is a simple Python function
that prints a greeting message...

✓ Done
```

### TUI Mode
```
┌─────────────────────────────────────────┐
│  🚀 Prompt: Explain Code                │
├─────────────────────────────────────────┤
│                                         │
│  🤖 AI:                                 │
│  ⟳ This is a simple Python function     │
│    that prints a greeting message...    │
│                                         │
│  [60% ▓▓▓▓▓░░░░░]                      │
│                                         │
│  ✓ Complete                            │
└─────────────────────────────────────────┘
```

---

## 🔌 API Integration

### OpenAI
```
Request → GET /v1/chat/completions
          (streaming: SSE)
          
Response ← data: {"choices":[{"delta":{"content":"..."}}]}
          ← data: {"choices":[{"finish_reason":"stop"}]}
```

### Gemini
```
Request → POST /v1beta/models/gemini-2.5-flash-lite:generateContent
          (single request)
          
Response ← {"candidates":[{"content":{"parts":[{"text":"..."}]}}]}
```

---

## 💾 Database Schema

```
PROMPTS TABLE
┌──────────┬──────────┬──────────┬──────────┬──────────┐
│ ID       │ NAME     │ TEMPLATE │ ENGINE   │ CREATED  │
├──────────┼──────────┼──────────┼──────────┼──────────┤
│ explain  │ Explain  │ Explain: │ openai   │ 2025-01-10
│ code     │ Code     │ {{sel}}  │          │          │
├──────────┼──────────┼──────────┼──────────┼──────────┤
│ translate│Translate │ Translate│ gemini   │ 2025-01-10
│ _en      │ to EN    │ {{sel}}  │          │          │
└──────────┴──────────┴──────────┴──────────┴──────────┘
```

---

## 🎯 Decision Tree

```
USER RUNS: ./prompt-agent run <id>
        │
        ├─ Does config exist?
        │  ├─ NO → Error: Create ~/.prompt-agent/config.yaml
        │  └─ YES ↓
        │
        ├─ Is API key set?
        │  ├─ NO → Error: export OPENAI_API_KEY=...
        │  └─ YES ↓
        │
        ├─ Is --dry-run flag?
        │  ├─ YES → Show template, exit
        │  └─ NO ↓
        │
        ├─ Make API request
        │  ├─ Success → Stream response ✓
        │  ├─ 401 → Error: Invalid API key
        │  ├─ 429 → Error: Rate limited
        │  └─ Other → Error: [details]
        │
        └─ DONE
```

---

## ✅ Setup Timeline

```
MINUTE 0-1: Preparation
  ├─ Get API key
  ├─ Set environment variable
  └─ Clone/locate project

MINUTE 1-2: Build
  ├─ go build ./cmd/prompt-agent
  └─ Verify binary exists

MINUTE 2-3: Configure
  ├─ Create ~/.prompt-agent directory
  ├─ Create config.yaml
  └─ Verify config

MINUTE 3-4: Test
  ├─ Test dry-run (no API)
  ├─ Test with API
  └─ Verify streaming

MINUTE 4-5: Ready!
  ├─ Create first custom prompt
  ├─ Run and see result
  └─ ✅ READY TO USE!
```

---

**Visual Guide Complete! 🎨**
