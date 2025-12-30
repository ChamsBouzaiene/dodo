# Feature: Background Command Execution & Process Management

**Type:** Feature
**Priority:** P1
**Effort:** L
**Owners:** TBD
**Target Release:** TBD

## Context
Currently, the agent interacts with the system using synchronous commands (`run_cmd`). This blocks the agent while a command runs, making it impossible to manage long-running processes like development servers, databases, or file watchers. To effectively debug and develop apps, the agent needs to "multi-task": start a server, observe it, and run tests against it simultaneously.

## Problem Statement
- The agent cannot start a development server (e.g., `npm start`) without blocking its entire execution loop.
- There is no way to perform health checks or integration tests against a running local server.
- `run_test` and `run_build` consume the main thread, disallowing "watch mode" testing or background builds.

## Goals
- [ ] Enable `run_background` tool for long-running processes.
- [ ] Add background/async capability to `run_test` and `run_build`.
- [ ] Provide observability mechanism for background processes (logs, health).
- [ ] Support Docker port forwarding so the agent (and user) can access sandboxed services.
- [ ] Ensure robust lifecycle management (cleanup on exit).

## Non-Goals
- [ ] Interactive TTY (sending input to background processes) is out of scope for V1.
- [ ] Persistence of processes after the Dodo application closes.

## Requirements
### Functional
1.  **Process Management**:
    - `start_process(cmd, args, options)`: Returns `process_id`. Options include `ports` (for Docker), `cwd`.
    - `stop_process(process_id)`: Terminate specific process.
    - `list_processes()`: Show active processes, their status, exit codes, and resource usage.
2.  **Tool Integration**:
    - Update `run_test` and `run_build` to accept a `background: boolean` flag.
3.  **Observability (The "Clever Solution")**:
    - **Log Peeking**: `read_process_output(id, lines=50)` to check recent output.
    - **Port Wait**: `wait_for_port(port, timeout)` to block *smartly* until a server is ready, avoiding "sleep 10s" guessing.
    - **Event Stream** (Optional V2): Inject vital process events (crashes) into the chat context automatically.

### Non-Functional
- **Resource Limits**: Background processes must honor the sandbox CPU/Memory limits.
- **Cleanup**: All background processes MUST terminate when the main Dodo session ends. Note: User specifically requested *not* to persist them.

## Impacted Areas
- **Backend (`internal/sandbox`)**:
    - Major refactor of `Runner` interface to support async operations.
    - `DockerRunner` needs to handle `detached` containers and dynamic port binding.
    - `HostRunner` needs to track `exec.Cmd` objects in a map.
- **Agent Tools (`internal/tools`)**:
    - New `tools/background` package.
    - Modifications to `tools/execution/cmd.go`, `test.go`, `build.go`.

## Proposed Approach

### 1. Sandbox Architecture Update
Introduce a `ProcessManager` inside the `Runner`.
```go
type ProcessManager interface {
    Start(ctx, cmd string, opts ProcessOpts) (string, error)
    Stop(id string) error
    List() []ProcessInfo
    GetLogs(id string, tail int) string
}
```

### 2. Docker Implementation
- **Start**: Use `ContainerCreate` -> `ContainerStart` (no wait).
- **Network**: Parse `opts.Ports` (e.g., `["3000"]`). Bind container port 3000 to a free ephemeral port on host (or same port if host mode).
- **Logs**: `ContainerLogs` API.

### 3. Agent Tooling ("Clever Observability")
Instead of spamming the chat with logs, we give the agent precise instruments:
- **`run_background`**: Starts the job.
- **`wait_for_service`**: New tool. Accepts `{ port: 1234, output_regex: "Ready on" }`. This effectively pauses the *agent's reasoning* (efficiently) until the condition is met, so it doesn't need to loop `read_logs` 100 times.

### 4. Updates to Existing Tools
- `run_test(..., background=true)`: Instead of streaming output, it returns a `job_id` and runs the test runner in a detached process. State is tracked in `ProcessManager`.

## Breaking Change Assessment
**Classification:** Non-breaking / Additive.
**Reasoning:** Existing `run_cmd` remains synchronous. New capabilities are opt-in via new tools or new arguments.

## Risks & Mitigations
- **Risk:** Agent starts too many heavy processes (e.g., 5 React servers).
  - **Mitigation:** Hard limit on concurrent background (e.g., max 3) in V1.
- **Risk:** Docker port conflicts.
  - **Mitigation:** Auto-assign host ports if the requested port is taken, and report the *mapped* port back to the agent.

## Task Breakdown
1.  Refactor `internal/sandbox` to include `ProcessManager` interface.
2.  Implement `ProcessManager` for `HostRunner` (using `os/exec`).
3.  Implement `ProcessManager` for `DockerRunner` (using detached containers + port mapping).
4.  Implement `run_background`, `stop_process`, `list_processes`, `read_process_output` tools.
5.  Implement `wait_for_service` tool (the observability helper).
6.  Migrate `run_test` and `run_build` to use the shared `ProcessManager` when `background: true`.
7.  Add cleanup hook in `cmd/dodo/main.go` to ensure `StopAll()` is called on shutdown.
