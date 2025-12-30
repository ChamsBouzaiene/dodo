# Dodo Development Guide

## Build Engine
```bash
cd /Users/chamseddinebouzaiene/Documents/dodo
go build -o repl ./cmd/repl
```

## Run in Dev Mode
```bash
cd /Users/chamseddinebouzaiene/Documents/dodo/ink-ui
npm run dev
```

## ZSH Alias (add to ~/.zshrc)
```bash
export DODO_ENGINE_PATH="/Users/chamseddinebouzaiene/Documents/dodo/repl"
alias dodo="node /Users/chamseddinebouzaiene/Documents/dodo/ink-ui/scripts/cli.js"
```

## Build UI for Production
```bash
cd /Users/chamseddinebouzaiene/Documents/dodo/ink-ui
npm run build
```

## Project Structure
- `cmd/repl/` - Go backend engine
- `ink-ui/` - React/Ink terminal UI
- `internal/` - Go packages (providers, tools, prompts)
