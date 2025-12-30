## Engine ↔ CLI NDJSON Protocol

This document captures the minimal contract used by `dodo engine --stdio`. Each line written to stdin/stdout is a standalone JSON object with a `type` field.

### Commands (CLI ➜ Engine)

| Type | JSON shape | Notes |
| ---- | ---------- | ----- |
| `start_session` | `{"type":"start_session","session_id":"optional","repo_root":"/path","meta":{...}}` | If `session_id` is omitted the engine generates one. When this command succeeds, the first event referencing the session is `status=session_ready`, which contains the canonical `session_id`. Treat that event as the authoritative ID even if the client proposed another value. |
| `user_message` | `{"type":"user_message","session_id":"abc123","message":"..."}` | Adds a user turn to an existing session. |

### Events (Engine ➜ CLI)

All events include the session id when they relate to a specific session (`"session_id":"abc123"`). Global events (e.g. engine startup) omit it.

| Type | Fields | Semantics |
| ---- | ------ | --------- |
| `status` | `status`, `detail` | `status="engine_ready"` is emitted once when the stdio bridge is ready to accept commands. `status="session_ready"` is emitted after a successful `start_session` and signals that the engine recognized the session; clients should treat this event as the session acknowledgment. Other statuses (`thinking`, `step_start`, `retry`, `budget_exceeded`, `done`, etc.) track lifecycle progress. |
| `assistant_text` | `content`, `source`, `final?` | The `source` field indicates how the text should be presented: `delta` = incremental streaming tokens, `assistant` = non-streaming assistant response, `respond.summary` = structured summary returned by the `respond` tool (safe to show in a summary pane). |
| `tool_event` | `tool`, `phase`, `success?`, `details?` | Tool lifecycle notifications (`phase="start"`/`"end"`). |
| `files_changed` | `files[]` | Emitted when the agent reports file modifications (typically sourced from the `respond` tool payload). |
| `done` | `summary`, `files_changed[]` | Session request completed. |
| `error` | `message`, `kind`, `details?` | Protocol or engine errors that the client should surface. |

Future transports (Ink UI, IDE integration, WebSocket) should reuse these structures for consistency.

