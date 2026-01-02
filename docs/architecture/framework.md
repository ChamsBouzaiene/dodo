# The Dodo Framework

Dodo is not just an application; it is built on top of a powerful, custom-built agentic framework written in pure Go.

## Why a Custom Framework?

When we started Dodo, we evaluated existing ecosystem options like LangChain (Python/Go) and various "Agent" libraries. We found them to be:
1.  **Too Abstraacted**: Hiding prompt logic and control flow behind layers of complexity.
2.  **Heavy Dependencies**: Importing massive dependency trees just to make an API call.
3.  **Hard to Debug**: When an agent loops or fails, understanding *why* inside a black-box framework is painful.

**We chose to build from scratch.**

## Core Philosophy: "Zero Dependencies"

The Dodo Engine (`internal/engine`) is built with **zero external functional dependencies**. It relies only on the Go standard library and the official SDKs for LLM providers (OpenAI/Anthropic).

This gives us:
-   **Binary Size**: The entire compiled engine is ~15MB.
-   **Performance**: Sub-millisecond overhead for agent reasoning steps.
-   **Stability**: No "dependency hell" or breaking changes from upstream frameworks.

## Architecture Internals

The framework is Event-Driven and Hook-Based.

### The Agent Loop

Instead of a complex graph, Dodo uses a modified ReAct loop with lifecycle hooks:

```go
type Agent struct {
    // ...
    Hooks AgentHooks
}

type AgentHooks struct {
    OnStep       func(ctx Context, step Step)
    OnToolStart  func(ctx Context, tool ToolCall)
    OnToolFinish func(ctx Context, result ToolResult)
}
```

This architecture allows us to attach:
-   **Real-time Reporting**: Hooks stream events to the UI immediately.
-   **Safety Checks**: A hook can intercept a `run_cmd` tool call and block it if it looks dangerous.
-   **Middleware**: We can inject "thoughts" or memories into the context before the LLM sees them.

### Interfaces

Everything is an interface, making the system highly testable and extensible.

**The Tool Interface:**
```go
type Tool interface {
    Name() string
    Description() string
    Schema() jsonschema.Definition
    Execute(ctx context.Context, args json.RawMessage) (any, error)
}
```

**The Provider Interface:**
```go
type LLMProvider interface {
    Stream(ctx context.Context, messages []Message) (<-chan Event, error)
}
```

## Future: Standalone Library

We believe this "thin, fast, and transparent" approach to AI agents is valuable beyond Dodo.

**Roadmap Item**: We plan to extract `internal/engine` into a standalone library (`github.com/dodo-ai/framework`) so other Go developers can build:
-   Custom coding assistants.
-   DevOps bots.
-   Customer support agents.

All with the same performance and safety guarantees as Dodo.
