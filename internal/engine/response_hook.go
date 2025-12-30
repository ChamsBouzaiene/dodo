package engine

import (
	"context"
	"fmt"
	"os"
	"time"
)

// ResponseHook prints assistant responses to stdout.
// This is a generic solution for displaying responses in REPL/interactive mode.
type ResponseHook struct {
	Writer *os.File // Defaults to os.Stdout
}

// NewResponseHook creates a new response hook that prints to stdout.
func NewResponseHook() *ResponseHook {
	return &ResponseHook{Writer: os.Stdout}
}

func (h *ResponseHook) OnStepStart(context.Context, *State)                               {}
func (h *ResponseHook) OnBeforeLLM(context.Context, *State, []ChatMessage, []ToolSchema) {}
func (h *ResponseHook) OnToolCall(context.Context, *State, ToolCall)                      {}
func (h *ResponseHook) OnToolResult(context.Context, *State, ToolCall, string, error)     {}
func (h *ResponseHook) OnToolOutput(context.Context, *State, string, string)               {}
func (h *ResponseHook) OnHistoryChanged(context.Context, *State)                          {}
func (h *ResponseHook) OnSummarize(context.Context, *State, []ChatMessage, []ChatMessage) {}
// OnStreamDelta prints streaming text deltas incrementally to stdout.
func (h *ResponseHook) OnStreamDelta(_ context.Context, _ *State, delta string) {
	// Print delta immediately without newline (for incremental output)
	fmt.Fprint(h.Writer, delta)
	// Note: We don't flush here as it can cause performance issues
	// The terminal will display buffered output automatically
}

// OnAfterLLM prints the assistant's response when it's a final answer (no tool calls).
func (h *ResponseHook) OnAfterLLM(ctx context.Context, st *State, resp LLMResponse) {
	// Only print if this is a final response (no tool calls)
	if len(resp.ToolCalls) == 0 {
		content := resp.Assistant.Content
		// Skip empty or whitespace-only responses
		if content != "" && content != " " {
			fmt.Fprintf(h.Writer, "assistant> %s\n", content)
		}
	}
}

func (h *ResponseHook) OnDone(context.Context, *State)                                    {}
func (h *ResponseHook) OnRetryAttempt(context.Context, *State, int, int, time.Duration, error) {}
func (h *ResponseHook) OnRetryExhausted(context.Context, *State, error)                  {}
func (h *ResponseHook) OnBudgetExceeded(context.Context, *State, int, int, int)        {}
func (h *ResponseHook) OnBudgetCompression(context.Context, *State, int, int, CompressionStrategy) {}
func (h *ResponseHook) OnSoftCapReached(context.Context, *State, error)                  {}
