# Development Guide

Want to contribute to Dodo? This guide will help you set up your development environment.

## Prerequisites

-   **Go 1.21+**: Required to build the engine.
-   **Node.js 18+**: Required to build the Ink CLI.
-   **Docker**: Optional (but recommended) for testing sandboxed execution.

## Monorepo Structure

Dodo is a monorepo containing both the backend and frontend:

```text
dodo/
├── cmd/            # Go entrypoints
├── internal/       # Go engine source code
├── ink-ui/         # React/Ink CLI source code
├── npm-package/    # The publishing artifacts for npm
└── docs/           # This website
```

## Running Locally

### 1. Build the Engine

The React UI needs the Go binary to be built first.

```bash
# In the root 'dodo/' directory
go build -o repl ./cmd/repl
```

This creates a `repl` binary in your root folder.

### 2. Run the UI in Dev Mode

Navigate to the UI directory and start the dev server. We pass the `--engine` flag to tell it where to find the binary we just built.

```bash
cd ink-ui
npm install

# Run dev mode (with hot reloading for UI components)
npm run dev -- --engine ../repl
```

### 3. Debugging

You can enable verbose debug logs to see exactly what JSON triggers are being sent back and forth.

```bash
export DODO_DEBUG=true
npm run dev -- --engine ../repl
```

Logs will be written to `/tmp/dodo_debug.log`.

## Running Tests

### Go Tests (Engine)
```bash
go test ./...
```

### TypeScript Tests (UI)
```bash
cd ink-ui
npm test
```
