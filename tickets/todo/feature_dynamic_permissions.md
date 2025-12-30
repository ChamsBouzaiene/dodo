# Feature: Expanded Allowlist & Interactive Command Permissions

**Type:** Feature (Security/UX)
**Priority:** P1
**Effort:** L
**Owners:** TBD
**Target Release:** v0.1.0

## Context
The current `run_cmd` tool blocks any command not in a hardcoded allowlist (`go`, `npm`, etc.). This prevents the agent from performing useful tasks like fetching data (`curl`), managing archives (`zip`), or using system tools (`grep`, `awk`). Users need a way to approve these commands dynamically.

## Problem Statement
- **Missing Tools**: Essential tools (`curl`, `wget`, `tar`, `unzip`) are blocked.
- **Rigid Security**: Users cannot authorize "safe" commands that aren't on the list.
- **UX Dead End**: When a command is blocked, the tool just fails. The agent can't ask for permission.

## Goals
- [ ] **Expand Defaults**: Add `curl`, `wget`, `tar`, `zip/unzip`, `grep`, `awk`, `sed` to the built-in allowlist.
- [ ] **Interactive Permission Flow**:
    - When a blocked command is attempted, **Suspend** the agent.
    - Prompt the user in the UI: "Allow `curl example.com`? [Once] [Always] [Deny]".
    - **Resume** based on the decision.
- [ ] **Config**: Persist "Always" decisions to `~/.dodo/config.json`.

## Requirements
### Functional
1.  **Protocol**:
    - New Event: `request_permission` (`{"id": "...", "kind": "command", "data": "curl ..."}`).
    - New Command: `submit_permission` (`{"request_id": "...", "granted": true, "scope": "session"|"global"}`).
2.  **Backend**:
    - `runCmdImpl` checks allowlist. If blocked -> return `ErrPermissionRequired`.
    - `step.go`: Catch `ErrPermissionRequired`.
        - Emit `request_permission`.
        - Wait for `submit_permission` response (channel/mutex).
        - If Granted: Update session/config allowlist and Retry the tool.
        - If Denied: Return "User denied permission" error to Agent.
3.  **Frontend**:
    - `PermissionModal` component.
    - Appears when `request_permission` event is received.
    - Blocks input until resolved.

### Non-Functional
- **Security**: "Global" scope updates `~/.dodo/config.json`, effectively whitelisting that command for *all* future sessions.

## Impacted Areas
- `internal/tools/execution/cmd.go` (Validator)
- `internal/engine/step.go` (Suspend/Resume/Wait Logic)
- `internal/engine/protocol/protocol.go` (New messages)
- `ink-ui/src/ui/app.tsx` (Modal)

## Task Breakdown
1.  **Quick Win**: PR to expand the hardcoded allowlist in `cmd.go`.
2.  **Protocol**: Define Permission events.
3.  **Backend Core**: Implement the "Wait for Permission" state mechanism in `step.go`.
4.  **Frontend**: Build the Permission UI.

## Risks & Mitigations
- **Risk**: Agent hangs indefinitely waiting for permission.
  - **Mitigation**: Add a timeout (e.g., 5 mins) -> Auto-Deny.
