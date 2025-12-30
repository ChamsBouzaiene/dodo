# Feature: Slash Command System & Interactive Help Modal

**Type:** Feature
**Priority:** P1
**Effort:** M
**Owners:** TBD
**Target Release:** TBD

## Context
Users are unaware of the full capabilities of Dodo (both slash commands like `/configure` and agent instructions). We need a discoverable, interactive help system that guides users without cluttering the chat history.

## Problem Statement
- No centralized list of commands.
- Users rely on guessing what the agent can do.
- Existing "help" text in system prompts is invisible to the user.

## Goals
- [ ] Implement a **Modal/Overlay System** in the TUI.
- [ ] Create an interactive `/help` modal with navigation (arrow keys).
- [ ] Implement a Slash Command Interceptor.
- [ ] Improve discovery via input hints ("Type /help...").

## Non-Goals
- [ ] Fuzzy search within the help modal (simple list navigation for V1).

## Requirements
### Functional
1.  **Interceptor**:
    - Intercept inputs starting with known commands (e.g., `/help`, `/configure`, `/exit`).
    - Pass unknown slash commands (like `/bin/bash`) to the agent to avoid blocking valid path queries.
2.  **Help Modal**:
    - **Trigger**: `/help` command or `F1` key (optional).
    - **UI**: A centered box rendering on top of the conversation.
    - **Content**:
        - **System Commands**: `/configure`, `/exit`, `/clear`.
        - **Agent Capabilities**: "Run Tests", "Edit Files", "Search Codebase".
        - **Shortcuts**: `Ctrl+C` to cancel, `Up/Down` history.
    - **Navigation**: Use Up/Down arrow keys to scroll through the list. `Esc` to close.
3.  **Discovery**:
    - Update `Input` placeholder to: "Ask Dodo a question, or type /help..."

### Non-Functional
- **UX**: The modal must capture focus (keypresses) while open, preventing typing in the main input box.

## Impacted Areas
- **Frontend (`App.tsx`)**: Needs state `showHelpModal`.
- **Component (`HelpModal.tsx`)**: New component.
- **Input Handling**: Need to route keypresses to Modal when active (Priority Key Handler).

## Breaking Change Assessment
**Classification:** Non-breaking / Additive.
**Reasoning:** UI overlay only.

## Proposed Approach
1.  **Modal Infrastructure**:
    - Add `activeModal: 'none' | 'help' | 'config'` state to `App.tsx`.
    - Render `<HelpModal />` inside `Box` with absolute positioning (or z-index equivalent in Ink/Flexbox order) when active.
2.  **Command Handling**:
    - In `handleSubmit`:
        ```typescript
        if (input === '/help') { setActiveModal('help'); return; }
        ```
3.  **HelpModal Component**:
    - Uses `useInput` (or `useKeypress`) to handle Up/Down/Esc.
    - Renders a purely visual list of commands.

## Testing Strategy
### Unit
- Test `HelpModal` renders correctly.
- Test `Esc` calls `onClose`.

### Manual
- Open app, type `/help`. Verify modal appears. Use arrows. Press Esc. Verify modal closes and input focus returns.

## Risks & Mitigations
- **Risk**: Modal styling looks broken on small terminals.
  - **Mitigation**: Use `minHeight`/`minWidth` and check terminal size.
- **Risk**: Focus trapping (typing in input while modal is open).
  - **Mitigation**: Disable main `Input` component (`isDisabled={true}`) when modal is active.

## Task Breakdown
1.  Create `SlashCommandRegistry` (shared config).
2.  Implement `HelpModal.tsx` component (visuals + navigation).
3.  Add Modal state to `App.tsx` and integrate.
4.  Implement Slash Command Interceptor logic.
5.  Update Input placeholder.
