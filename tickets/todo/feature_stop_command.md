# Feature: Stop/Cancel Running Task

**Type:** Feature
**Priority:** P1
**Effort:** S
**Owners:** TBD
**Target Release:** TBD

## Context
Currently, once a user sends a prompt, the agent runs until completion. If the agent gets stuck, loops, or the user realizes they made a mistake, there is no way to abort the process without killing the entire application (losing all context).

## Problem Statement
- No "Stop" button or command.
- `Ctrl+C` kills the whole app, not just the current generation.
- Resources (LLM tokens, CPU) are wasted on tasks the user no longer wants.

## Goals
- [ ] Implement a `/stop` command.
- [ ] Intercept `Ctrl+C` during an active turn to trigger Stop instead of Exit.
- [ ] Gracefully terminate the Agent's run loop and any child processes.
- [ ] Return the Agent to a "Ready" state (awaiting input).

## Requirements
### Functional
1.  **Protocol**:
    - Add `cancel_request` command: `{"type": "cancel_request", "session_id": "uuid"}`.
2.  **Backend**:
    - Manage `context.CancelFunc` for the active `Run()` execution in `sessionState`.
    - Upon receiving `cancel_request`, invoke the cancel function immediately.
    - Ensure `Agent.Run` handles `context.Canceled` gracefully (returns error, but keeps session valid).
3.  **Frontend**:
    - Maps `/stop` command to `cancel_request`.
    - Maps `Ctrl+C` (whilst `isRunning === true`) to `cancel_request`.
    - UI updates status to "Stopping..." -> "Ready" (with "Interrupted by user" message).

### Non-Functional
- **Speed**: Cancellation must propagate instantly to subprocesses (e.g., stopping a long `grep`).

## Impacted Areas
- **Protocol**: `protocol.go`.
- **Backend**: `stdio_runner.go`.
- **Frontend**: `App.tsx`, `useEngineConnection.ts`.

## Breaking Change Assessment
**Classification:** Non-breaking / Additive.

## Proposed Approach
1.  **Backend (`stdio_runner.go`)**:
    - Add `cancelRequest context.CancelFunc` to `sessionState` struct.
    - In `HandleUserMessage`:
        ```go
        ctx, cancel := context.WithCancel(parentCtx)
        session.cancelRequest = cancel
        go func() { defer cancel(); agent.Run(ctx, ...) }()
        ```
    - In `HandleCancelRequest`: calls `session.cancelRequest()`.
2.  **Frontend**:
    - `useKeypress`: If `Ctrl+C` and `isRunning`, send Cancel. Else, `process.exit`.

## Risks & Mitigations
- **Risk**: Race conditions (Stopping just as it finishes).
  - **Mitigation**: Cancellation is idempotent. If it's done, cancel does nothing.
- **Risk**: Partial file edits.
  - **Mitigation**: Agent tools use atomic writes where possible, but stopping mid-write is inherently risky. We accept this risk for "Stop" functionality.

## Task Breakdown
1.  Define `CancelRequestCommand` in Protocol.
2.  Implement cancellation logic in `stdio_runner`.
3.  Wire up Frontend `Ctrl+C` interception and `/stop` handler.
