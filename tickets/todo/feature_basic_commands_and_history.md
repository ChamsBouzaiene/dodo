# Feature: Basic Commands & History Navigation

**Type:** Feature
**Priority:** P1
**Effort:** S
**Owners:** TBD
**Target Release:** v0.1.0

## Context
Users expect standard shell-like behavior in a CLI tool. This includes command history navigation (Up/Down arrows) and standard lifecycle commands.

## Goals
- [ ] Implement command history persistence (in-memory or local storage).
- [ ] Implement **Up/Down arrow key navigation** in the input field to cycle through history.
- [ ] Implement **/exit** command to cleanly shut down the app.
- [ ] Implement **/clear** command to clear the conversation display.

## Requirements
### Functional
1.  **History Navigation**:
    - `Up`: Show previous command.
    - `Down`: Show next command (or empty if at end).
    - History should be preserved during the session.
2.  **Commands**:
    - `/exit`: Call app exit handler (same as Ctrl+C).
    - `/clear`: Clear the `turns` state in `AppLayout` or `useEngineConnection`.

## Impacted Areas
- **Frontend (`App.tsx`)**: History state management.
- **Component (`Input.tsx` / `Footer.tsx`)**: Keypress handling for arrows.
- **Hooks (`useEngineConnection.ts`)**: `clearTurns` method needed.

## Breaking Change Assessment
**Classification:** Non-breaking.
