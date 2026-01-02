# CLI Reference

Dodo's interface is designed to be fully controllable from the keyboard.

## Slash Commands

Type these commands into the input bar to control the session.

| Command | Description |
| :--- | :--- |
| `/help` | Show a list of available commands and their descriptions. |
| `/clear` | **Clear Context**: Wipe the conversation history but keep the current project/configuration. Useful when the context window gets full. |
| `/conf` | **Configure**: Open the interactive Setup Wizard to change LLM providers or API keys. |
| `/mode` | **Switch Mode**: Toggle between `Planner`, `Code`, and `Architect` modes. |
| `/tools` | **Tools Viewer**: List all enabled tools for the current session and their status. |
| `/index` | **Re-index**: Force a full re-scan of the codebase for the semantic search index. |
| `/debug` | **Diagnostics**: Dump the current internal state to a temp file (useful for bug reports). |
| `/exit` | **Quit**: Safely shutdown the session and cleanup any resources. |

## Keyboard Shortcuts

| Shortcut | Action |
| :--- | :--- |
| `Metx + C` | **Copy**: Copy selected text or code block. |
| `Ctrl + C` | **Cancel / Exit**: Interrupt the current agent action or exit the application. |
| `Up / Down` | **History**: Navigate through your previous inputs in the prompt bar. |
| `Enter` | **Submit**: Send your message to the agent. |

## Launch Arguments

When starting `dodo` from the terminal, you can pass flags:

```bash
dodo [flags]
```

- `--repo <path>`: Open a specific repository (defaults to current directory).
- `--debug`: Enable verbose debug logging to stderr.
- `--version`: Print the current version.
