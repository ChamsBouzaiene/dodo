// engine/hooks.go
package engine

import (
	"context"
	"time"
)

type Hook interface {
	OnStepStart(ctx context.Context, st *State)
	OnBeforeLLM(ctx context.Context, st *State, messages []ChatMessage, toolSchemas []ToolSchema)
	OnAfterLLM(ctx context.Context, st *State, resp LLMResponse)
	OnToolCall(ctx context.Context, st *State, call ToolCall)
	OnToolResult(ctx context.Context, st *State, call ToolCall, result string, err error)
	OnToolOutput(ctx context.Context, st *State, toolName string, output string)
	OnHistoryChanged(ctx context.Context, st *State)
	OnSummarize(ctx context.Context, st *State, before, after []ChatMessage)
	OnStreamDelta(ctx context.Context, st *State, delta string) // for streaming
	OnDone(ctx context.Context, st *State)
	// Retry hooks
	OnRetryAttempt(ctx context.Context, st *State, attempt int, maxAttempts int, delay time.Duration, err error)
	OnRetryExhausted(ctx context.Context, st *State, err error)
	// Budget hooks
	OnBudgetExceeded(ctx context.Context, st *State, tokenCount int, softLimit int, hardLimit int)
	OnBudgetCompression(ctx context.Context, st *State, beforeTokens, afterTokens int, strategy CompressionStrategy)
	// Soft cap hooks
	OnSoftCapReached(ctx context.Context, st *State, err error)
}

// NopHook lets you implement any hook you need.
type NopHook struct{}

func (NopHook) OnStepStart(context.Context, *State)                                        {}
func (NopHook) OnBeforeLLM(context.Context, *State, []ChatMessage, []ToolSchema)           {}
func (NopHook) OnAfterLLM(context.Context, *State, LLMResponse)                            {}
func (NopHook) OnToolCall(context.Context, *State, ToolCall)                               {}
func (NopHook) OnToolResult(context.Context, *State, ToolCall, string, error)              {}
func (NopHook) OnToolOutput(context.Context, *State, string, string)                       {}
func (NopHook) OnHistoryChanged(context.Context, *State)                                   {}
func (NopHook) OnSummarize(context.Context, *State, []ChatMessage, []ChatMessage)          {}
func (NopHook) OnStreamDelta(context.Context, *State, string)                              {}
func (NopHook) OnDone(context.Context, *State)                                             {}
func (NopHook) OnRetryAttempt(context.Context, *State, int, int, time.Duration, error)     {}
func (NopHook) OnRetryExhausted(context.Context, *State, error)                            {}
func (NopHook) OnBudgetExceeded(context.Context, *State, int, int, int)                    {}
func (NopHook) OnBudgetCompression(context.Context, *State, int, int, CompressionStrategy) {}
func (NopHook) OnSoftCapReached(context.Context, *State, error)                            {}
