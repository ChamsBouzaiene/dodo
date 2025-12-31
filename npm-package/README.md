<div align="center">
  <h1>Dodo AI</h1>
  <p><strong>The Developer-First AI Coding Agent</strong></p>
  <p>Local-First • Sandboxed • Model-Agnostic</p>
  <p>
    <a href="https://github.com/ChamsBouzaiene/dodo">GitHub Repository</a> • 
    <a href="https://github.com/ChamsBouzaiene/dodo/issues">Report Issues</a>
  </p>
</div>

---

**Dodo** is an open-source, terminal-based AI coding environment designed for developers who want the power of an autonomous agent without giving up control or privacy.

This package installs the CLI wrapper that automatically downloads and runs the high-performance Go engine for your operating system.

## 🚀 Quick Start

Install globally via npm:
```bash
npm install -g dodo-ai
```

Navigate to any project directory:
```bash
cd my-project
dodo
```

### 🔄 Keep Up to Date
To update to the latest version of Dodo, simply run:
```bash
npm install -g dodo-ai@latest
```

### ⚙️ Configuration & LLMs
Dodo works with your favorite LLM. You can configure it easily using the built-in wizard:

1. **Run the Setup Wizard**:
   Inside Dodo, type `/conf` (or select "Start New Session" -> "Configure").

2. **Choose your Provider**:
   - **OpenAI** (GPT-4o, o1-preview)
   - **Anthropic** (Claude 3.5 Sonnet)
   - **Google** (Gemini 1.5 Pro)
   - **Local Models** (Ollama, LM Studio via OpenAI-compatible endpoint)

3. **Enter API Keys**:
   The wizard will securely save your keys locally in `~/.dodo/config.yaml`.

## ✨ Key Features

- **🛡️ Sandboxed Execution**: All agent commands run in a secure, isolated Docker container (or controlled local environment).
- **🧠 Model Agnostic**: Bring your own LLM. Works with Claude 3.5 Sonnet, GPT-4o, Gemini, or local models via Ollama.
- **⚡ Fast & Native**: Core engine written in Go for speed, with a beautiful React-based terminal UI.
- **🔧 Developer First**: Full shell access, file editing, and project indexing capabilities.

## 🎮 CLI Commands

Once inside Dodo, you have a powerful set of tools at your disposal:

| Command | Description |
|---------|-------------|
| `/conf` | **Setup Wizard**: Configure LLM provider, API keys, and sandbox settings. |
| `/index` | **Project Indexing**: Scan your codebase to give the agent full context. |
| `/tools` | **Tool Viewer**: See exactly what capabilities the agent has enabled. |
| `/mode` | **Switch Modes**: Toggle between Planner, Code, and Architect modes. |
| `/clear` | **Clear Context**: Start fresh without losing your session settings. |
| `/exit` | **Quit**: Safely shutdown the session. |

## 📦 Supported Platforms

The installer automatically fetches the correct binary for:
- **macOS** (Apple Silicon + Intel)
- **Linux** (AMD64 + ARM64)
- **Windows** (AMD64)

## 🛠️ Troubleshooting

**"dodo: command not found"**
Ensure your npm global bin directory is in your PATH.
```bash
export PATH=$PATH:$(npm config get prefix)/bin
```

**"Permission denied"**
If you installed Node with root/sudo, you might need to fix npm permissions or run with sudo (not recommended).

## 📄 License
MIT © [Chams Bouzaiene](https://github.com/ChamsBouzaiene)
