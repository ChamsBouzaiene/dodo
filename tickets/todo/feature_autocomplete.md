# Feature: Command Autocomplete & Generic Triggers

**Type:** Feature (UX)
**Priority:** P2
**Effort:** M
**Owners:** TBD
**Target Release:** v0.1.0

## Context
Slash commands (`/help`) and future features (like `@file` context) rely on intuitive discovery. Users expect an IDE-like autocomplete experience where typing a trigger character (`/` or `@`) opens a suggestion menu.

## Problem Statement
- Commands are hidden; users must memorize them.
- No visual feedback when typing a command.

## Goals
- [ ] Create a `SuggestionBox` component (renders *above* the input).
- [ ] Modify `Input` component to support "Trigger characters".
- [ ] Implement Slash Command (`/`) autocomplete as the first use case.

## Requirements
### Functional
1.  **Triggers**:
    - `/` → Opens list of commands (`/help`, `/config`, `/stop`, `/clear`).
    - Future: `@` → Opens list of files.
2.  **Navigation**:
    - `ArrowUp` / `ArrowDown`: Move selection in the suggestion box.
    - `Tab` / `Enter`: Accept selection.
    - `Esc`: Close box.
3.  **Filtering**:
    - Typing after trigger filters the list (fuzzy or prefix).
    - Example: `/co` -> shows `config`, `context`.
4.  **UI Data**:
    - Description for each command (e.g., "/stop - Stop current generation").

### Non-Functional
- **Performance**: Zero latency on keypress.

## Impacted Areas
- `ink-ui/src/components/common/Input.tsx`: Needs to expose key events or accept an interceptor.
- `ink-ui/src/components/Footer.tsx`: Will host the `SuggestionBox`.

## Proposed Approach
1.  **Input Component**:
    - Add `onKeyDownCapture`: Before processing internal keys, check if it's a navigation key (Up/Down) AND if a menu is open. If so, call `onSelectNext/Prev` and `preventDefault`.
    - Add `onCursorChange`: Report cursor position and current "word" to parent.
2.  **Footer Controller**:
    - Detects if current word starts with `/`.
    - If yes, renders `<SuggestionBox items={filteredCommands} />` immediately above Input.

## Task Breakdown
1.  Implement `SuggestionBox` (Ink component).
2.  Refactor `Input` to allow external control of navigation keys.
3.  Implement Trigger logic in `Footer`.
