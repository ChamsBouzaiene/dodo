# Feature: Persistent Sessions & Context Memory

**Type:** Feature
**Priority:** P1
**Effort:** L
**Owners:** TBD
**Target Release:** TBD

## Context
Dodo currently operates with "amnesia" - restart the app, and all context is lost. This forces users to repeatedly explain context or prevents them from resuming complex tasks. We want to persist sessions globally (to keep repos clean) and provide intelligent context carry-over between sessions.

## Problem Statement
- **Data Loss**: Closing the terminal loses the entire conversation history.
- **Context Loss**: Starting a new session forces the user to manually re-brief the agent on recent work.
- **Repo Pollution**: Storing logs/sessions in the repo is messy; users prefer a global store.

## Goals
- [ ] Persist all sessions to a global user directory (e.g., `~/.dodo/sessions/`).
- [ ] Implement "Resume Session" capability (exact state restoration).
- [ ] Implement "New Session with Context" (auto-summarization of previous work).
- [ ] Auto-generate human-readable session names (e.g., "Refactoring Auth - 2h ago").
- [ ] Provide a TUI Session Picker on startup.

## Non-Goals
- [ ] Storing sessions in the local git repo (explicit user request to avoid pollution).
- [ ] Editing history granularity (v1 is load entire session).

## Requirements
### Functional
1.  **Storage**:
    - Path: `~/.dodo/sessions/<repo_hash>/<session_id>.json`
    - Structure: We group sessions by Repository (using a hash of the repo path) so the Session Picker only shows *relevant* sessions for the current project.
2.  **Session Management**:
    - **Auto-Save**: Save state asynchronously every N turns.
    - **Auto-Naming**: Use a small LLM call (or heuristic) after the first few turns to generate a Title (e.g., "Fixing API Bug").
    - **Schema**:
        ```json
        {
          "id": "uuid",
          "repo_path": "/path/to/project",
          "created_at": "timestamp",
          "updated_at": "timestamp",
          "title": "Auto Generated Name",
          "history": [...],
          "summary": "User fixed X. Next steps: Y." // Generated on close
        }
        ```
3.  **Startup Flow**:
    - When running `dodo` in a repo:
    - Check global store for sessions matching this repo.
    - **UI**: Show "Select Session":
        - `[+] Start New Session` (Injects summary from last session if available)
        - `[>] Resume: Fixing API Bug (2h ago)`
        - `[>] Resume: Refactor (Yesterday)`
4.  **Context Carry-over**:
    - When a session is "Closed" (or when loading a "New" one), we take the `summary` from the *most recent* session.
    - We inject this into the **System Prompt** of the new session:
      > "Previous Session Context: The user was working on [Summary]. The last known state was [Status]."

### Non-Functional
- **Privacy**: Files stored strictly locally in user home dir.
- **Performance**: Summary generation happens in background (don't block exit).

## Impacted Areas
- **Backend**: `internal/session` package.
- **Frontend**: New `SessionPicker` screen in `ink-ui`.
- **Engine**: `Agent` needs `ExportState()` and `ImportState()` methods.

## Breaking Change Assessment
**Classification:** Non-breaking / Additive.
**Reasoning:** Start-up flow changes, but doesn't break existing agent logic.

## Proposed Approach
1.  **Session Store (`internal/session/store.go`)**:
    - `Save(session)`: Writes JSON.
    - `GetRecent(repoPath)`: Returns list of metadata.
2.  **Summarizer (`internal/session/summarizer.go`)**:
    - Function `Summarize(history)` -> returns text.
    - Triggered on `SIGINT` (Exit) or periodically.
3.  **Frontend**:
    - App start checking for sessions.
    - Render `SessionList` component.
    - On selection, pass `sessionId` to Backend.

## Testing Strategy
### Unit
- Test serialization/deserialization of `State`.
- Test session filtering by repo path.

## Risks & Mitigations
- **Risk**: Global store gets huge/cluttered.
  - **Mitigation**: Auto-rotate/delete sessions older than 30 days (or keep max 10 per repo).
- **Risk**: "Summary" is hallucinated or inaccurate.
  - **Mitigation**: Prompt engineering for the summarizer ("Be concise, list ONLY facts and TODOs").

## Task Breakdown
1.  Implement `session` package (Store, Model).
2.  Add `RepoHash` logic to link global sessions to local repos.
3.  Implement `Summarizer` (LLM call).
4.  Implement `SessionPicker` UI in Ink.
5.  Wire up "New Session" to inject summary context.
