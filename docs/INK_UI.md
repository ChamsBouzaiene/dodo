# Ink-based CLI UI

The `ink-ui/` workspace contains a React + Ink client that talks to `dodo engine --stdio` using the NDJSON protocol defined in `docs/PROTO.md`.

## Prerequisites

1. Build or have access to the Go engine binary in the repo root:
   ```bash
   go build -o dodo ./cmd/repl
   ```
   (If the binary is missing, the UI falls back to `go run ./cmd/repl engine --stdio` automatically; building upfront avoids repeated compilation.)
2. Install the Ink UI dependencies once:
   ```bash
   cd ink-ui
   npm install
   ```

## Running the UI

From the `ink-ui/` directory:
```bash
npm run dev -- --repo /path/to/target/repo
```

Optional flags:

- `--session-id <id>` – ask the engine to reuse/label the session with a specific id.
- `--engine <path>` – path to a prebuilt `dodo` binary (defaults to `../dodo` if present, otherwise `go run ./cmd/repl`).
- `--engine-cwd <path>` – working directory for the engine command (defaults to the repository root).

The UI automatically sends a `start_session` command on boot and begins streaming events as soon as the engine emits `status=session_ready`.

## Layout Overview

- **Header** – Displays the repo name, current session id (shortened), overall status, and the engine command that was launched.
- **Main pane** – Shows the conversation history. User prompts appear in green; streaming assistant responses append in place. Summaries emitted via `respond.summary` are rendered beneath the assistant text.
- **Side pane** – Lists the most recent `tool_event` messages and the latest `files_changed` payload.
- **Footer** – Input box for the next user instruction plus a placeholder token counter. Input is disabled while a request is active or before the session becomes ready.

## Protocol Mapping

The `EngineClient` class mirrors the Go protocol types (see `internal/engine/protocol`):

- Commands sent upstream:
  - `start_session` (automatic on boot)
  - `user_message` (on Enter)
- Events handled downstream:
  - `status` → drives the status chip + info banner.
  - `assistant_text` → populates the streaming area (`delta`, `assistant`, `respond.summary`).
  - `tool_event` / `files_changed` → update the side pane.
  - `done` → marks the turn complete, re-enables input, caches summary + files.
  - `error` → shows an inline error banner and clears the pending state.

Refer to `docs/PROTO.md` for the canonical NDJSON contract.

## Manual Test Checklist

1. Start the UI against the repo itself:
   ```bash
   cd ink-ui
   npm run dev -- --repo ..
   ```
2. When the header shows `session_ready`, type a request (e.g., “Summarize main.go”).
3. Observe:
   - Streaming assistant output in the main pane.
   - Tool invocations appearing in the sidebar.
   - `files_changed` list updating after completion (if the agent reports file edits).
4. Send another request to confirm multi-turn behavior.
5. Exit with `Ctrl+C`; the UI forwards the signal to the engine and shuts down cleanly.
