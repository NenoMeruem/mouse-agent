# Phase 4 - Trigger System Setup Guide

## Overview

Phase 4 implements a trigger system that allows you to run prompts via hotkeys without opening the terminal.

## Architecture

```
Hotkey Press (skhd/sxhkd)
         ↓
  prompt-agent run <prompt-id>
         ↓
  Selection from clipboard
         ↓
  Prompt rendered
```

## macOS Setup (Recommended)

### 1. Install skhd (Hotkey Daemon)

```bash
brew install skhd
```

### 2. Create ~/.skhdrc configuration

```bash
# Create or edit the skhd config file
vim ~/.skhdrc
```

Add your hotkey mappings:

```text
# Explain code with Alt+Space
alt + space : prompt-agent run explain_code

# Code review with Alt+Shift+Space  
alt + shift + space : prompt-agent run code_review
```

### 3. Start skhd

```bash
# Start skhd (runs in background)
skhd

# Or with Homebrew services (recommended)
brew services start skhd

# Check status
brew services list | grep skhd
```

### 4. Grant Accessibility Permission

On macOS 10.15+, you need to grant Accessibility permission:

1. System Preferences → Security & Privacy → Accessibility
2. Add `/usr/local/bin/skhd` to the list
3. Or if using Homebrew Cellar location, find and add that path

### 5. Test Your Setup

```bash
# Copy some code to clipboard
echo 'func hello() { return "world" }' | pbcopy

# Now press Alt+Space and check that the prompt appears
# The formatted prompt should appear in your terminal
```

## Linux Setup (sxhkd)

### 1. Install sxhkd

```bash
sudo apt-get install sxhkd  # Ubuntu/Debian
sudo pacman -S sxhkd        # Arch
```

### 2. Create ~/.config/sxhkd/sxhkdrc

```bash
mkdir -p ~/.config/sxhkd
vim ~/.config/sxhkd/sxhkdrc
```

Add mappings:

```text
alt + space
  prompt-agent run explain_code

alt + shift + space
  prompt-agent run code_review
```

### 3. Start sxhkd

```bash
sxhkd &
```

## Configuration

Edit `~/.prompt-agent/config.yaml`:

```yaml
triggers:
  enabled: true
  default_prompt: explain_code
  hotkeys:
    alt+space: explain_code
    alt+shift+space: code_review
```

## Features

- ✅ Multiple hotkey triggers
- ✅ Per-hotkey prompt selection
- ✅ Clipboard selection support
- ✅ Template variable injection
- ✅ Cross-platform ready

## Troubleshooting

### Hotkey Not Working on macOS

1. Check skhd is running:
   ```bash
   brew services list | grep skhd
   ```

2. Check logs:
   ```bash
   log stream --predicate 'process == "skhd"'
   ```

3. Grant Accessibility permission (see above)

4. Restart skhd:
   ```bash
   brew services restart skhd
   ```

### Shell Not Found Error

If you see "command not found: prompt-agent", add to your shell profile:

```bash
export PATH="$PATH:/path/to/prompt-agent/directory"
```

## Future Plans (Phase 5+)

- [ ] Mouse middle-click trigger
- [ ] Context menu with prompt selection
- [ ] Accessibility API for better selection
- [ ] GUI indicator when trigger runs
- [ ] Log file for debugging
