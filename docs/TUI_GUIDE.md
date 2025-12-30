# Dodo Interactive TUI Mode

## Overview

Dodo now features an interactive Terminal User Interface (TUI) powered by Bubble Tea, allowing you to run multiple tasks in a single session with real-time metrics and progress tracking.

## Features

✨ **Interactive Session** - Ask multiple tasks without restarting
📊 **Live Metrics** - Real-time token usage, steps, and cost tracking
🎨 **Beautiful UI** - Clean, colorful interface with animations
📝 **Task History** - View previous tasks and results
⚡ **Fast Workflow** - No need to restart for each task

## Usage

### Start TUI Mode

```bash
./dodo --repo /path/to/repo --tui
```

### With Indexing

```bash
./dodo --repo /path/to/repo --tui --index
```

### With Model Override

```bash
./dodo --repo /path/to/repo --tui --model gpt-4o
```

## Keyboard Shortcuts

- **Enter**: Submit task
- **Ctrl+U**: Clear input
- **Ctrl+C** (idle): Quit
- **Ctrl+C** (running): Cancel current task
- **Q** (idle): Quit

## Interface Layout

```
┌─────────────────────────────────────────────────────────┐
│ 🦤 DODO Interactive Agent          Tasks: 3             │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ✅ Task Completed                                       │
│                                                          │
│  Task: Add borders to snake game                        │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ I've successfully implemented borders...           │ │
│  │ - Updated Renderer interface                       │ │
│  │ - Implemented border drawing                       │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 📊 Metrics                                         │ │
│  │                                                     │ │
│  │ ⏱️  Duration: 23.5s                                 │ │
│  │ 🔢 Steps: 12                                        │ │
│  │ 🛠️  Tools: 18                                       │ │
│  │ 📥 Input Tokens: 15,234                            │ │
│  │ 📤 Output Tokens: 2,456                            │ │
│  │ 💰 Est. Cost: $0.0234                              │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
├─────────────────────────────────────────────────────────┤
│ ❯ █                                                     │
├─────────────────────────────────────────────────────────┤
│ enter: submit task  •  ctrl+u: clear  •  q: quit        │
└─────────────────────────────────────────────────────────┘
```

## Running State

When a task is running, you'll see:

```
┌────────────────────────────────────────────────────────┐
│ 🎯 Task: Fix the food spawning issue                   │
└────────────────────────────────────────────────────────┘

⏱️  12s  |  Step 8  |  Reading internal/game/actors.go

⠋ Working...
```

## Examples

### Example Session

```bash
$ ./dodo --repo ../myproject --tui

🦤 DODO Interactive Agent          Tasks: 0

👋 Welcome! Enter a task below to get started.

❯ add error handling to the API endpoints█

# Agent works on the task...

✅ Task Completed

Task: add error handling to the API endpoints

Result: I've added comprehensive error handling...

📊 Metrics
⏱️  Duration: 18.2s
🔢 Steps: 9
🛠️  Tools: 14
📥 Input Tokens: 12,456
📤 Output Tokens: 1,892
💰 Est. Cost: $0.0189

❯ now add input validation█

# Continue with more tasks...
```

## Benefits Over Single-Task Mode

| Feature | Single Task | TUI Mode |
|---------|-------------|----------|
| Multiple tasks | ❌ Restart each time | ✅ Continuous session |
| Metrics display | ✅ End only | ✅ Real-time + history |
| Progress tracking | ❌ Logs only | ✅ Live UI updates |
| Task history | ❌ None | ✅ Full history |
| User experience | 😐 CLI | 😍 Interactive TUI |

## Technical Details

- **Framework**: Bubble Tea (Go TUI framework)
- **Styling**: Lip Gloss (terminal styling)
- **State Management**: Bubble Tea's Elm architecture
- **Concurrency**: Each task runs in its own context
- **Cancellation**: Graceful task cancellation with Ctrl+C

## Tips

1. **Use TUI for iterative development** - Perfect for making multiple changes to a codebase
2. **Monitor token usage** - See real-time cost estimates for each task
3. **Review history** - Scroll through previous tasks and results
4. **Cancel long tasks** - Press Ctrl+C to cancel without losing session

## Troubleshooting

### TUI not displaying correctly

- Ensure your terminal supports ANSI colors and Unicode
- Try resizing your terminal window
- Use a modern terminal emulator (iTerm2, Alacritty, Windows Terminal)

### Task not starting

- Check that indexing completed successfully
- Verify your API keys are set (OPENAI_API_KEY, etc.)
- Look for error messages in the TUI

### Metrics not showing

- Metrics are estimated based on model pricing
- Some providers may not report token counts accurately
- Check that the model name is recognized

## Future Enhancements

- [ ] Task queue management
- [ ] Export task history
- [ ] Custom themes
- [ ] Split-pane view for code + results
- [ ] Real-time file watching integration
- [ ] Task templates/favorites

