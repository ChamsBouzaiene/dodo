# E2E Testing Framework: Learnings & Guide

## Overview
The `dodo-ink-ui` End-to-End (E2E) framework provides a robust environment for verifying the full application lifecycle. It treats the CLI application as a black box, spawning it as a real child process and interacting with it via standard input/output, while simulating the backend "Engine" using a mock TCP server.

### Architecture
- **Subject**: The real `dodo-ink-ui` application (spawned via `node-pty`).
- **Control**: `CliWrapper` (manages process spawning, input writing, output capturing).
- **Backend Simulation**: `MockEngineServer` (TCP server acting as the Dodo Engine).
- **Test Runner**: `Vitest` (manages test execution and assertions).

---

## Key Learnings & "Gotchas"

### 1. Protocol Fidelity is Critical
The mock server must behave **exactly** like the real engine. Small deviations cause test failures or false positives.
- **Activities vs. Timeline**: Do not use deprecated events like `timeline_update`. Use `activity` events (type `tool`, `reasoning`, `edit`) with `started`/`completed` statuses.
- **Event Types**: Ensure event types match `src/protocol.ts`. For example, `setup_required` is its own event type, not a `status` event value.
- **Thinking State**: To show "Thinking...", send an `activity` of type `reasoning` or `thinking`.

### 2. Timing & Cooldowns
- **Component Cooldowns**: Some UI components (e.g., `SetupWizard`) have built-in input cooldowns (e.g., 500ms on mount) to prevent accidental double-presses. Tests **must** respect these by adding `await new Promise(r => setTimeout(r, 600))` before sending input.
- **Asynchronous Output**: Always use `await cli.waitForOutput('Expected Text')` rather than fixed sleeps where possible. This is faster and more reliable.

### 3. Terminal Constraints
- **Vertical Space**: `node-pty` simulates a real terminal window. If the output scrolls off-screen, it might disappear from the immediate buffer (depending on implementation).
- **Recommendation**: Initialize `CliWrapper` with `rows: 60` to ensure enough vertical space for long lists or conversation history.
- **Input Nuances**: In `ink`, `\r` (Carriage Return) usually acts as Enter. However, in some specific input states or components, `\n` (Newline) might be safer or required.

### 4. Process Management
- **Graceful Shutdown**: Always ensure `cli.stop()` and `mockServer.stop()` are called in `afterAll` to prevent zombie processes.
- **Status Checks**: verifying `cli.isRunning()` is useful to check for unexpected crashes (e.g., `process.exit(1)`).

---

## Test Catalog

| Test File | Focus Area | Key Scenarios |
|-----------|------------|---------------|
| `startup.test.ts` | Initialization | Connection handshake, "READY" status display. |
| `interaction.test.ts` | Chat Flow | User input, "Thinking..." indicator, Assistant text response. |
| `tool_execution.test.ts` | Activities | Tool started/completed, Spinner rendering, output display. |
| `file_updates.test.ts` | Code Rendering | `edit` activities, `CodeDiff` component, syntax verification. |
| `error_handling.test.ts` | Robustness | Backend error events, UI Red status indicator. |
| `session_management.test.ts` | state | Session reloading, context events ("Reading file..."). |
| `commands.test.ts` | Input Handling | Slash commands (`/help` modal, `/clear` history). |
| `setup_wizard.test.ts` | First-Run | `setup_required` trigger, keyboard nav, config saving. |
| `project_plan.test.ts` | UI Panels | `project_plan` event, Side panel rendering. |

---

## How to Add New E2E Tests

### Step 1: Define the Scenario
Identify exactly what user journey you want to test. Is it a new event type? A complex UI interaction?

### Step 2: Scaffold the Test
Copy the boilerplate from an existing test (e.g., `interaction.test.ts`).
```typescript
describe('My New Feature', () => {
    // ... setup mockServer and cli ...
});
```

### Step 3: Implement Mock Protocol
In the `mockServer.on('connection')` handler, define how the backend should behave.
- **Trigger**: Does the backend send data first (e.g., `project_plan`)?
- **Response**: Does it wait for user input (`socket.on('data')`) before responding?

**Example:**
```typescript
socket.on('data', (data) => {
    if (data.toString().includes('my command')) {
        socket.write(JSON.stringify({ type: 'my_event', ... }) + '\n');
    }
});
```

### Step 4: Write Assertions
Interact with the CLI and assert on the visual output.
```typescript
cli.write('my command\r');
await cli.waitForOutput('Expected UI Response');
```

### Step 5: Run and Debug
Run your specific test file:
```bash
npx vitest run src/tests/e2e/my_new_test.test.ts
```
**Debugging Tips:**
- enable `DODO_DEBUG=true` in `CliWrapper` options to see internal UI logs.
- Use `console.log(cli.getBuffer())` in your test if assertions are failing to see what the terminal actually looks like.
