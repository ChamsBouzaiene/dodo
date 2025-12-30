# Engine Package

The `engine` package provides a ReAct-style agent orchestration engine for managing LLM interactions, tool execution, and conversation flow.

## Architecture Overview

The engine follows a layered architecture with clear separation of concerns:

### Core Components

1. **State Management** (`state.go`)
   - `State`: Tracks conversation history, step count, phase, and token usage
   - Manages conversation context and execution state

2. **Execution Loop** (`run.go`, `step.go`)
   - `Run()`: Main execution loop that orchestrates the ReAct cycle
   - `stepOnce()`: Executes one reasoning/acting cycle
   - Broken into focused functions: `prepareMessages()`, `callLLMWithRetry()`, `processLLMResponse()`, `executeToolCalls()`

3. **LLM Client Interface** (`types.go`)
   - `LLMClient`: Provider-agnostic interface for LLM interactions
   - Supports both chat completion and streaming
   - Implementations in `providers/` package (OpenAI, Anthropic)

4. **Tool System** (`tools.go`)
   - `ToolRegistry`: Map of tool names to tool definitions
   - `Tool`: Defines tool schema, description, and execution function
   - Tools can be marked as retryable or non-retryable

5. **Error Handling & Retry** (`errors.go`, `retry.go`)
   - `RetryExhaustedError`: Type-safe error for exhausted retries
   - `RetryConfig`: Separate policies for LLM and tool retries
   - Exponential backoff with jitter and Retry-After header support
   - Error classification: Retryable, Maybe, Non-retryable

6. **Hooks System** (`hooks.go`, `multi_hook.go`)
   - `Hook`: Interface for observing engine events
   - `Hooks`: Multi-hook dispatcher
   - Events: step start, LLM calls, tool execution, retries, completion

7. **Message Processing** (`processors.go`, `summarizer.go`)
   - Processors: Transform message history (summarize, truncate, filter)
   - `SummarizeOlderThanN`: Compresses old history using LLM
   - `KeepLastN`: Keeps recent messages intact
   - `TruncateLongTools`: Trims large tool outputs

8. **Configuration** (`config.go`)
   - `EngineConfig`: Centralized configuration
   - `DefaultRetryConfig()`: Sensible defaults for retry policies

## Key Features

### ReAct Loop
The engine implements a Reasoning-Acting loop:
1. **Reason**: Call LLM with conversation history
2. **Act**: Execute requested tools (if any)
3. **Observe**: Append tool results to history
4. **Repeat**: Continue until LLM provides final answer

### Error Recovery
- Automatic retry with exponential backoff
- Separate retry policies for LLM calls and tool execution
- Respects `Retry-After` headers from rate-limited APIs
- Type-safe error classification

### Step Counting
- `Step`: Counts successful reasoning cycles (increments only on success)
- `Retries`: Tracks retry attempts separately
- Failed steps don't consume step budget

### Type Safety
- `MessageRole`: Typed constants for message roles (RoleSystem, RoleUser, RoleAssistant, RoleTool)
- `ChatMessage.Validate()`: Validates message structure
- Prevents invalid role strings

## Usage Example

```go
import (
    "context"
    "github.com/yourorg/dodo/internal/engine"
    "github.com/yourorg/dodo/internal/engine/providers"
)

// Create LLM client
llm, modelName, err := providers.NewLLMClientFromEnv(ctx)

// Define tools
reg := engine.ToolRegistry{
    "echo": {
        Name:        "echo",
        Description: "Echo the provided text",
        SchemaJSON:  `{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`,
        Fn: func(ctx context.Context, args map[string]any) (string, error) {
            return args["text"].(string), nil
        },
    },
}

// Configure state
st := &engine.State{
    History: []engine.ChatMessage{
        {Role: engine.RoleSystem, Content: "You are a helpful assistant."},
    },
    Model:     modelName,
    MaxSteps:  30,
    BudgetTok: 4000,
}

// Configure hooks
logger := engine.LoggerHook{L: log.Default()}
responseHook := engine.NewResponseHook()

// Configure options
retryConfig := engine.DefaultRetryConfig()
opts := engine.ChatOptions{
    MaxOutputTokens: 512,
    RetryConfig:     &retryConfig,
}

// Run the engine
err = engine.Run(ctx, llm, reg, st, engine.Hooks{logger, responseHook}, opts)
```

## File Organization

- **Core**: `run.go`, `step.go`, `state.go`, `types.go`
- **Error Handling**: `errors.go`, `retry.go`
- **Tools**: `tools.go`
- **Hooks**: `hooks.go`, `multi_hook.go`, `hook_logger.go`, `response_hook.go`, `events.go`
- **Processing**: `processors.go`, `summarizer.go`, `phase.go`
- **Configuration**: `config.go`
- **Utilities**: `utils.go`
- **Providers**: `providers/` (OpenAI, Anthropic implementations)
- **Streaming**: `step_stream.go` (streaming variant of step execution)
- **Legacy**: `model_adapter.go` (deprecated, use providers instead)

## Design Principles

1. **Separation of Concerns**: Business logic separated from I/O (HTTP, DB, external APIs)
2. **Dependency Injection**: Dependencies passed as parameters, not globals
3. **Testability**: Pure functions and interfaces enable easy testing
4. **Type Safety**: Typed constants and validation prevent runtime errors
5. **Error Recovery**: Configurable retry policies with intelligent error classification
6. **Observability**: Hook system for monitoring and logging

## Error Handling

Errors are classified into three categories:
- **Retryable**: Rate limits (429), server errors (5xx), network timeouts
- **Maybe**: Context deadline exceeded, length overflow (limited retries)
- **Non-retryable**: Auth failures (401/403), bad requests (400), quota exhausted (402)

Retry policies are configurable per operation type (LLM vs tools).

## Extension Points

- **Hooks**: Implement `Hook` interface to observe engine events
- **Processors**: Create custom `Processor` functions to transform message history
- **Providers**: Implement `LLMClient` interface to add new LLM providers
- **Tools**: Register tools in `ToolRegistry` with execution functions

