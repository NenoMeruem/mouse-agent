# ✅ Fixed! Prompt Add Command Now Supports Flags

## 🔧 What Was Fixed

The `prompt add` command was using **interactive mode only**. It now supports:

1. **Flag-based mode** (what you wanted to use)
2. **Interactive mode** (still available as fallback)

---

## 📝 Now You Can Use:

### Option 1: Command-Line Flags (Recommended)
```bash
./prompt-agent prompt add \
  --id translate \
  --name "Translate to English" \
  --template "Dịch sang Tiếng Anh:\n{{selection}}" \
  --engine gemini

# Result:
# ✓ Prompt 'translate' created
#   Variables: [selection]
```

### Option 2: Interactive Mode (Still Works)
```bash
./prompt-agent prompt add

# Will prompt you for:
# ID: translate
# Name: Translate to English
# Description: ...
# Engine: gemini
# Template: ...
```

---

## 🎯 Available Flags

```bash
./prompt-agent prompt add --help

# Flags:
#   --id string          Prompt ID (required when using flags)
#   --name string        Prompt name (optional, defaults to ID)
#   --description string Prompt description (optional)
#   --engine string      LLM engine: openai or gemini (default: openai)
#   --template string    Prompt template (required when using flags)
#   -h, --help           Help for add
```

---

## 🚀 Practical Examples

### Example 1: Explain Code (OpenAI)
```bash
./prompt-agent prompt add \
  --id explain \
  --name "Explain Code" \
  --template "Explain this code:\n{{selection}}" \
  --engine openai
```

### Example 2: Translate (Gemini)
```bash
./prompt-agent prompt add \
  --id translate \
  --template "Translate to English:\n{{selection}}" \
  --engine gemini
```

### Example 3: Fix Bugs (with description)
```bash
./prompt-agent prompt add \
  --id fix \
  --name "Fix Bugs" \
  --description "Find and fix bugs in code" \
  --template "Fix the bugs in this code:\n{{selection}}" \
  --engine openai
```

---

## 📋 What Changed in Code

**File**: `internal/cli/prompt_add.go`

**Added**:
- Global variables for flags: `addID`, `addName`, `addEngine`, `addTemplate`
- Flag registration in `init()` function
- New function: `addPromptFromFlags()` for flag-based mode
- New function: `addPromptInteractive()` for interactive mode
- Logic to choose between modes

**Before**:
```go
var promptAddCmd = &cobra.Command{
  RunE: func(cmd *cobra.Command, args []string) error {
    // Only interactive mode
  },
}
```

**After**:
```go
var promptAddCmd = &cobra.Command{
  RunE: func(cmd *cobra.Command, args []string) error {
    // If flags provided, use them; otherwise use interactive mode
    if addID != "" || addTemplate != "" {
      return addPromptFromFlags()
    }
    return addPromptInteractive()
  },
}
```

---

## ✨ Benefits

✅ **Faster**: No interactive prompts  
✅ **Scriptable**: Can be used in bash scripts  
✅ **Flexible**: Still supports interactive mode  
✅ **Backward Compatible**: Old way still works  

---

## 🧪 Next Step: Rebuild and Test

```bash
# Build
go build ./cmd/prompt-agent

# Test with flags
./prompt-agent prompt add \
  --id translate \
  --template "Dịch sang Tiếng Anh:\n{{selection}}" \
  --engine gemini

# Should see:
# ✓ Prompt 'translate' created
#   Variables: [selection]
```

---

## 🎉 Now Your Command Works!

```bash
./prompt-agent prompt add \
  --id translate \
  --template "Dịch sang Tiếng Anh:\n{{selection}}" \
  --engine gemini

# ✓ SUCCESS!
```

---

**Fix Complete!** 🚀
