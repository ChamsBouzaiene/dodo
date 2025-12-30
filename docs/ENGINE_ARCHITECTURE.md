# Engine Architecture - Complete Guide

## Overview

The engine (`internal/engine/`) is a **ReAct (Reasoning + Acting) loop** implementation that orchestrates LLM calls and tool execution. It's designed to be **provider-agnostic** (works with OpenAI, Anthropic, etc.) and **framework-independent** (no Eino dependency).

## Core Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    engine.Run()                         │
│  Main loop: steps until Done or MaxSteps reached      │
└─────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                  stepOnce()                             │
│  1. Detect phase                                        │
│  2. Build messages (with processors)                   │
│  3. Call LLM                                            │
│  4. Execute tools (parallel)                           │
│  5. Append results to history                          │
└─────────────────────────────────────────────────────────┘
```

## Key Components

### 1. **State** (`state.go`)

The `State` struct holds all execution context:

```go
type State struct {
    History   []ChatMessage  // Full conversation history
    Step      int            // Current step number
    Done      bool           // True when LLM returns no tool calls
    Phase     Phase          // Current phase (explore/discover/edit/validate)
    Model     string         // Model name (for provider)
    MaxSteps  int           // Maximum steps before stopping
    BudgetTok int           // Token budget (soft limit)
    Totals    Usage         // Accumulated token usage
}
```

**Key Methods:**
- `Append(msg ChatMessage)` - Adds message to history

**Design Notes:**
- State is **mutable** - modified in-place during execution
- History grows indefinitely (processors handle truncation)
- Phase is auto-detected from tool usage patterns

### 2. **LLMClient Interface** (`types.go`)

Abstracts provider-specific LLM calls:

```go
type LLMClient interface {
    Chat(ctx, model, messages, toolSchemas, opts) (LLMResponse, error)
    Stream(ctx, model, messages, toolSchemas, opts) (<-chan StreamEvent, <-chan error)
}
```

**Why This Design:**
- **Provider-agnostic**: Engine doesn't care if it's OpenAI or Anthropic
- **Simple contract**: Just messages in, response out
- **Tool support**: Schemas passed separately (provider converts to their format)

**Current Implementations:**
- `providers.OpenAIClient` - Direct OpenAI SDK calls
- `providers.AnthropicClient` - Direct Anthropic SDK calls

### 3. **Tool System** (`tools.go`)

Simple, direct tool execution:

```go
type ToolFunc func(ctx context.Context, args map[string]any) (string, error)

type Tool struct {
    Name        string
    Description string
    SchemaJSON  string  // JSON Schema for parameters
    Fn          ToolFunc
}

type ToolRegistry map[string]Tool
```

**Design Philosophy:**
- **Simple**: Just a function that takes args and returns a string
- **No abstraction**: Direct execution, no middleware
- **JSON Schema**: Standard format for all providers

**Tool Execution Flow:**
1. LLM returns `ToolCall` with name and args
2. Engine looks up tool in registry
3. Calls `Tool.Fn(ctx, args)`
4. Result appended to history as `tool` message

### 4. **ReAct Loop** (`run.go` + `step.go`)

The core execution loop:

```go
func Run(ctx, llm, reg, st, hooks, opts) error {
    for st.Step = 0; st.Step < st.MaxSteps && !st.Done; st.Step++ {
        if err := stepOnce(...); err != nil {
            return err
        }
    }
    if st.Done {
        hooks.OnDone(ctx, st)
    }
    return nil
}
```

**Step Execution (`stepOnce`):**

```
1. Detect Phase
   └─> Analyze history to determine current phase
   
2. Build Messages
   └─> Apply processors (summarize, truncate, etc.)
   
3. Call LLM
   └─> Convert engine.ChatMessage → provider format
   └─> Call provider SDK
   └─> Convert response → engine.LLMResponse
   
4. Update State
   └─> Append assistant message
   └─> Update token totals
   
5. Execute Tools (if any)
   └─> Run all tools in parallel (goroutines)
   └─> Append results in order
   
6. Continue Loop
   └─> If no tools → Done = true
   └─> Otherwise → next iteration
```

**Parallel Tool Execution:**
- All tools run concurrently using `sync.WaitGroup`
- Results collected in order-preserving slice
- Errors wrapped as `"ERROR: ..."` in content

### 5. **Message Processors** (`processors.go`)

Transform message history before LLM calls:

```go
type Processor func(ctx, st *State, msgs []ChatMessage) ([]ChatMessage, error)
```

**Built-in Processors:**

1. **`KeepLastN(n)`** - Keep only last N messages
   - Simple truncation
   - Used to limit context size

2. **`TruncateLongTools(maxChars)`** - Trim large tool outputs
   - Keeps head + tail, removes middle
   - Prevents huge tool outputs from bloating context

3. **`SummarizeOlderThanN(llm, keepLastN)`** - Compress old history
   - Uses LLM to summarize old messages
   - Keeps recent messages intact
   - Falls back to truncation if summarization fails

**Processor Pipeline:**
```go
msgs = ApplyProcessors(ctx, st, msgs,
    SummarizeOlderThanN(llm, 12),  // Summarize old
    KeepLastN(8),                   // Keep last 8
    TruncateLongTools(4000),        // Truncate large outputs
)
```

**Design Notes:**
- Processors are **composable** - chain multiple together
- Each processor receives full message list
- Processors can be stateful (access `st`)

### 6. **Hooks System** (`hooks.go` + `multi_hook.go`)

Observability and side effects:

```go
type Hook interface {
    OnStepStart(ctx, st *State)
    OnBeforeLLM(ctx, st, messages)
    OnAfterLLM(ctx, st, resp)
    OnToolCall(ctx, st, call)
    OnToolResult(ctx, st, call, result, err)
    OnHistoryChanged(ctx, st)
    OnSummarize(ctx, st, before, after)
    OnStreamDelta(ctx, st, delta)
    OnDone(ctx, st)
}
```

**Hook Lifecycle:**

```
Step Start
  └─> OnStepStart
  
Before LLM Call
  └─> OnBeforeLLM
  
After LLM Call
  └─> OnAfterLLM
  └─> OnHistoryChanged (assistant message added)
  
Tool Execution (for each tool)
  └─> OnToolCall
  └─> OnToolResult
  └─> OnHistoryChanged (tool result added)
  
Step Complete
  └─> (if Done) OnDone
```

**Multiple Hooks:**
- `Hooks []Hook` - Slice of hooks, all called in order
- Each hook method called for each registered hook
- Use `NopHook` to implement only needed methods

**Example Uses:**
- Logging (`LoggerHook`)
- Metrics collection
- TUI updates (`TUIHook`)
- Progress tracking

### 7. **Phase Detection** (`phase.go`)

Infers current workflow phase from tool usage:

```go
func DetectPhase(history []ChatMessage) Phase {
    // Look backwards through history for tool names
    // Return phase based on tool type
}
```

**Phases:**
- `PhaseExplore` - Initial exploration
- `PhaseDiscoverAndPlan` - Finding code, reading files
- `PhaseEdit` - Making changes (search_replace, write)
- `PhaseValidate` - Testing (run_tests, run_build, run_lint)

**Detection Logic:**
- Scans history backwards for `tool` messages
- Maps tool names to phases
- Returns first match found

**Limitations:**
- **Naive**: Only looks at tool names
- **No context**: Doesn't understand tool sequences
- **Last-wins**: Returns phase of most recent matching tool

### 8. **Budget Management** (`policy_budget.go`)

Token budget enforcement (soft limit):

```go
func BuildMessagesWithBudget(ctx, st, llm, estimate, processors) ([]ChatMessage, error) {
    // Estimate tokens
    // If over budget, summarize old messages
    // Return compressed messages
}
```

**Current State:**
- **Placeholder**: `estimate` function not implemented
- **Naive**: One-pass summarization attempt
- **Soft limit**: Doesn't hard-fail, just compresses

**Missing:**
- Real tokenizer integration
- Multi-pass compression
- Hard budget limits

## Data Flow

### Message Flow

```
User Input
  └─> ChatMessage{Role: "user", Content: "..."}
  └─> st.Append() → added to History

LLM Call
  └─> buildMessagesForCall()
      └─> Apply processors
      └─> Return processed messages
  └─> llm.Chat(messages, toolSchemas)
      └─> Provider converts to their format
      └─> SDK call
      └─> Provider converts response
  └─> LLMResponse{
          Assistant: ChatMessage{...},
          ToolCalls: [...],
          Usage: {...},
      }
  └─> st.Append(resp.Assistant)

Tool Execution
  └─> For each ToolCall:
      └─> reg[call.Name].Fn(ctx, call.Args)
      └─> Result string
  └─> ChatMessage{Role: "tool", Name: "...", Content: result}
  └─> st.Append(toolMessage)

Next Iteration
  └─> History now includes: user → assistant → tool → ...
  └─> LLM sees full context
```

### State Mutation

```
Initial State:
  History: [system, user]
  Step: 0
  Done: false

After Step 1:
  History: [system, user, assistant, tool]
  Step: 1
  Done: false (tools were called)

After Step 2:
  History: [system, user, assistant, tool, assistant]
  Step: 2
  Done: true (no tools called)
```

## Features

### ✅ What Works Well

1. **Simple API**
   - Single `Run()` function to execute
   - Clear state management
   - Easy to understand flow

2. **Provider Independence**
   - Works with any LLM via `LLMClient` interface
   - No framework dependencies
   - Easy to add new providers

3. **Parallel Tool Execution**
   - All tools run concurrently
   - Significant performance improvement
   - Order-preserving results

4. **Extensible Hooks**
   - Multiple hooks supported
   - Comprehensive lifecycle events
   - Easy to add logging/metrics

5. **Message Processing**
   - Composable processors
   - Built-in summarization
   - Tool output truncation

6. **Phase Detection**
   - Automatic phase tracking
   - Useful for UI/progress display

### ⚠️ Weaknesses & Limitations

1. **No Error Recovery**
   - Tool errors wrapped as strings
   - No retry logic
   - LLM errors stop execution immediately

2. **Naive Phase Detection**
   - Only looks at tool names
   - No understanding of sequences
   - Can misclassify phases

3. **Budget Management Incomplete**
   - No real tokenizer
   - Single-pass compression
   - No hard limits

4. **No Streaming Support**
   - `Stream()` method is placeholder
   - No incremental output
   - Can't show progress during generation

5. **State Mutability**
   - State modified in-place
   - Hard to test individual steps
   - No undo/redo capability

6. **Tool Registry Limitations**
   - Simple map lookup
   - No tool versioning
   - No tool dependencies
   - No tool validation

7. **Message Processing Issues**
   - Processors hardcoded in `buildMessagesForCall`
   - No dynamic processor selection
   - Processors can't access full context easily

8. **No Tool Call Validation**
   - Doesn't validate args against schema
   - No type checking
   - Errors only surface at execution

9. **Limited Error Context**
   - Errors don't include step number
   - No error categorization
   - Hard to debug failures

10. **No Cancellation Support**
    - Can't cancel mid-execution
    - No timeout per step
    - Context passed but not checked

## What's Missing

### Critical Missing Features

1. **Streaming Support**
   ```go
   // Current: Placeholder
   func Stream(...) (<-chan StreamEvent, <-chan error)
   
   // Needed: Real implementation
   - Incremental text output
   - Tool call streaming
   - Usage updates during generation
   ```

2. **Token Counting**
   ```go
   // Current: estimate function placeholder
   func estimate([]ChatMessage) int
   
   // Needed: Real tokenizer
   - Provider-specific tokenizers
   - Accurate counting
   - Budget enforcement
   ```

3. **Error Recovery**
   ```go
   // Needed:
   - Retry logic for transient errors
   - Tool error handling strategies
   - Graceful degradation
   ```

4. **Tool Validation**
   ```go
   // Needed:
   - Validate args against JSON schema
   - Type checking
   - Required field validation
   ```

5. **Cancellation & Timeouts**
   ```go
   // Needed:
   - Per-step timeouts
   - Cancellation support
   - Context propagation
   ```

### Nice-to-Have Features

1. **Tool Dependencies**
   - Tools that depend on other tools
   - Execution ordering
   - Dependency graph

2. **Tool Versioning**
   - Multiple versions of same tool
   - A/B testing
   - Gradual rollout

3. **State Snapshots**
   - Save/restore state
   - Checkpointing
   - Resume from failure

4. **Advanced Processors**
   - Context-aware summarization
   - Semantic compression
   - Relevance filtering

5. **Metrics & Observability**
   - Built-in metrics
   - Performance tracking
   - Cost estimation

6. **Tool Middleware**
   - Pre/post execution hooks
   - Rate limiting
   - Caching

## Current Bug: Empty Tool Content

**Symptom:** `Invalid value for 'content': expected a string, got null`

**Cause:** When a tool returns an empty string, OpenAI SDK may serialize it as `null` instead of `""`.

**Fix Needed:** In `providers/openai.go`, ensure tool messages always have non-empty content:

```go
case "tool":
    content := msg.Content
    if content == "" {
        content = "{}"  // Empty JSON object instead of empty string
    }
    openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
        Role:    openai.ChatMessageRoleTool,
        Name:    msg.Name,
        Content: content,
    })
```

## Common Issues & Solutions

### Issue 1: "Invalid value for 'content': expected a string, got null"

**Cause:** Tool result is empty or nil, provider expects string.

**Solution:** Ensure all tools return non-empty strings:
```go
Fn: func(ctx, args) (string, error) {
    result := doSomething()
    if result == "" {
        return "{}", nil  // Return empty JSON object, not empty string
    }
    return result, nil
}
```

### Issue 2: History Growing Too Large

**Cause:** No processors applied or insufficient compression.

**Solution:** Use processors:
```go
msgs, err := ApplyProcessors(ctx, st, msgs,
    SummarizeOlderThanN(llm, 10),
    KeepLastN(8),
    TruncateLongTools(2000),
)
```

### Issue 3: Tools Not Executing in Parallel

**Cause:** Already parallel, but check for blocking operations.

**Solution:** Ensure tools don't block each other:
- Use separate goroutines for I/O
- Don't share mutable state
- Use channels for coordination if needed

### Issue 4: Phase Detection Wrong

**Cause:** Naive detection based only on tool names.

**Solution:** Improve `DetectPhase()`:
- Look at tool sequences
- Consider context
- Use ML/pattern matching

## Extension Points

### Adding a New Provider

1. Implement `LLMClient` interface:
```go
type MyProvider struct { ... }

func (p *MyProvider) Chat(...) (LLMResponse, error) {
    // Convert engine.ChatMessage → provider format
    // Call provider SDK
    // Convert response → engine.LLMResponse
    return resp, nil
}
```

2. Add to factory:
```go
case "myprovider":
    return NewMyProviderClient(...)
```

### Adding a New Processor

```go
func MyProcessor(param int) Processor {
    return func(ctx, st *State, msgs []ChatMessage) ([]ChatMessage, error) {
        // Transform messages
        return transformed, nil
    }
}
```

### Adding a New Hook

```go
type MyHook struct { ... }

func (h MyHook) OnStepStart(ctx, st *State) {
    // Your logic
}

// Implement other methods or use NopHook
```

## Testing Strategy

### Unit Tests

- Test `stepOnce()` with mocked LLM
- Test processors independently
- Test tool execution
- Test phase detection

### Integration Tests

- Test full `Run()` loop
- Test with real providers (sandboxed)
- Test error scenarios
- Test parallel execution

### Performance Tests

- Measure tool parallelization speedup
- Test with large histories
- Test processor performance
- Test memory usage

## Future Improvements

1. **Streaming First**
   - Implement real streaming
   - Incremental updates
   - Better UX

2. **Better State Management**
   - Immutable state
   - State snapshots
   - Undo/redo

3. **Smarter Processors**
   - ML-based summarization
   - Relevance scoring
   - Context-aware compression

4. **Tool System Enhancements**
   - Tool dependencies
   - Tool versioning
   - Tool middleware

5. **Error Handling**
   - Retry strategies
   - Error recovery
   - Better error messages

6. **Observability**
   - Built-in metrics
   - Tracing support
   - Performance profiling

## Summary

The engine is a **simple, focused ReAct implementation** that:
- ✅ Works with any LLM provider
- ✅ Executes tools in parallel
- ✅ Provides extensibility via hooks
- ✅ Handles long conversations via processors

But it's missing:
- ❌ Real streaming support
- ❌ Token counting
- ❌ Error recovery
- ❌ Tool validation
- ❌ Advanced state management

The architecture is **clean and extensible**, making it easy to add these features incrementally.

