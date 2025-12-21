# Phase 5 - Selection Text & Run Pipeline Guide

## Overview

Phase 5 implements the complete end-to-end run pipeline:
- Selection text from clipboard
- Variable injection into prompts
- Flexible output rendering

This is the phase where prompts become truly dynamic and contextual.

## Architecture

```
Selection Provider (OS-specific)
         ↓
    Selection Manager
         ↓
     Run Command
         ↓
   Variable Collection
  (env vars + interactive)
         ↓
   Variable Validation
         ↓
   Prompt Builder
  (template injection)
         ↓
   Output Renderer
  (formatted or raw)
```

## Core Components

### 1. Selection Providers

**macOS** - Uses `pbpaste` command
```bash
echo 'some code' | pbcopy
./prompt-agent run explain_code  # Gets from clipboard
```

**Linux** - Uses `xclip` (stub in Phase 5)
```bash
# Future: echo 'code' | xclip -selection primary
```

**Windows** - Uses PowerShell (stub in Phase 5)
```bash
# Future: Get-Clipboard
```

### 2. Selection Manager

Provides unified interface for all OS providers:
```go
manager := selection.NewManager(provider)
text, err := manager.GetSelection()  // Error if empty
```

### 3. Run Command Pipeline

#### Basic Usage

```bash
./prompt-agent run <prompt-id>
```

This command:
1. Loads prompt from database
2. Gets selection from clipboard
3. Collects required variables
4. Builds and renders prompt

#### With Environment Variables

```bash
PROMPT_FILENAME="main.go" PROMPT_LANGUAGE="Go" \
  ./prompt-agent run code_review
```

Pattern: `PROMPT_<VARIABLE_NAME>`

#### Output Options

Raw output (no formatting):
```bash
./prompt-agent run explain_code --raw
```

Formatted output (default):
```bash
./prompt-agent run explain_code
```

Skip selection (use defaults only):
```bash
./prompt-agent run code_review --no-select
```

### 4. Variable Validation

Validates all required variables are present:
- From selection
- From environment variables
- From interactive input

Error example:
```
❌ missing variables: [filename, language]
```

## Usage Examples

### Example 1: Code Explanation

```bash
# Copy code to clipboard
cat main.go | pbcopy

# Run prompt
./prompt-agent run explain_code
```

Output:
```
============================
  PROMPT: Explain Code
============================
You are an expert programmer. Explain this code:
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
============================

📋 Prompt ID: explain_code | Engine: openai | Variables: [selection]
```

### Example 2: Code Review with Context

```bash
# Copy code
cat feature.py | pbcopy

# Run with additional variables
PROMPT_FILENAME="feature.py" PROMPT_LANGUAGE="Python" \
  ./prompt-agent run code_review
```

### Example 3: Raw Output (Integration)

```bash
# Use raw output for piping/scripting
./prompt-agent run explain_code --raw | \
  pbcopy  # Copy result to clipboard
```

## Configuration

Edit `~/.prompt-agent/config.yaml`:

```yaml
selection:
  provider: auto              # auto, macos, linux, windows
  fail_on_empty_selection: true
  trim_whitespace: true
```

## Error Handling

All errors are user-friendly:

```
❌ prompt not found: explain_code
❌ selection is empty
❌ missing variable: filename
❌ cannot build prompt: invalid template
```

## Testing

### Manual Test Checklist

1. **Basic Run**
   ```bash
   echo "some code" | pbcopy
   ./prompt-agent run explain_code
   ```

2. **With Variables**
   ```bash
   PROMPT_FILENAME="test.go" PROMPT_LANGUAGE="Go" \
     ./prompt-agent run code_review
   ```

3. **Empty Selection**
   ```bash
   echo "" | pbcopy
   ./prompt-agent run explain_code  # Should error gracefully
   ```

4. **Raw Output**
   ```bash
   ./prompt-agent run explain_code --raw
   ```

5. **Help**
   ```bash
   ./prompt-agent run --help
   ```

## Integration with Triggers

Phase 5 makes triggers from Phase 4 actually functional:

```yaml
# ~/.skhdrc (macOS)
alt + space : prompt-agent run explain_code

# Now hotkey works end-to-end:
# 1. Select text in any app
# 2. Press Alt+Space
# 3. Prompt appears in terminal
```

## Features

- ✅ Clipboard selection (macOS fully working)
- ✅ OS-specific providers
- ✅ Variable substitution
- ✅ Environment variable support
- ✅ Interactive variable input
- ✅ Raw and formatted output
- ✅ Graceful error handling
- ✅ Cross-platform architecture

## Limitations & Future

- **Current**: Selection from clipboard only
- **Phase 6+**: Native Accessibility API (better selection capture)
- **Phase 6+**: LLM integration (actual API calls)
- **Phase 7+**: Mouse trigger with context menu

## Next Steps

Phase 6 will add:
- [ ] LLM Gateway (OpenAI / Gemini)
- [ ] Streaming response
- [ ] TUI output with Bubbletea
- [ ] Request cancellation

See `PHASE6_PREVIEW.md` for details.
