package engine

import (
	"context"
	"time"
)

type Hooks []Hook

func (hs Hooks) OnStepStart(ctx context.Context, st *State) {
	for _, h := range hs {
		h.OnStepStart(ctx, st)
	}
}
func (hs Hooks) OnBeforeLLM(ctx context.Context, st *State, m []ChatMessage, schemas []ToolSchema) {
	for _, h := range hs {
		h.OnBeforeLLM(ctx, st, m, schemas)
	}
}
func (hs Hooks) OnAfterLLM(ctx context.Context, st *State, r LLMResponse) {
	for _, h := range hs {
		h.OnAfterLLM(ctx, st, r)
	}
}
func (hs Hooks) OnToolCall(ctx context.Context, st *State, c ToolCall) {
	for _, h := range hs {
		h.OnToolCall(ctx, st, c)
	}
}
func (hs Hooks) OnToolResult(ctx context.Context, st *State, c ToolCall, s string, e error) {
	for _, h := range hs {
		h.OnToolResult(ctx, st, c, s, e)
	}
}
func (hs Hooks) OnHistoryChanged(ctx context.Context, st *State) {
	for _, h := range hs {
		h.OnHistoryChanged(ctx, st)
	}
}
func (hs Hooks) OnSummarize(ctx context.Context, st *State, b, a []ChatMessage) {
	for _, h := range hs {
		h.OnSummarize(ctx, st, b, a)
	}
}
func (hs Hooks) OnStreamDelta(ctx context.Context, st *State, d string) {
	for _, h := range hs {
		h.OnStreamDelta(ctx, st, d)
	}
}
func (hs Hooks) OnDone(ctx context.Context, st *State) {
	for _, h := range hs {
		h.OnDone(ctx, st)
	}
}
func (hs Hooks) OnRetryAttempt(ctx context.Context, st *State, attempt int, maxAttempts int, delay time.Duration, err error) {
	for _, h := range hs {
		h.OnRetryAttempt(ctx, st, attempt, maxAttempts, delay, err)
	}
}
func (hs Hooks) OnRetryExhausted(ctx context.Context, st *State, err error) {
	for _, h := range hs {
		h.OnRetryExhausted(ctx, st, err)
	}
}
func (hs Hooks) OnBudgetExceeded(ctx context.Context, st *State, tokenCount int, softLimit int, hardLimit int) {
	for _, h := range hs {
		h.OnBudgetExceeded(ctx, st, tokenCount, softLimit, hardLimit)
	}
}
func (hs Hooks) OnBudgetCompression(ctx context.Context, st *State, beforeTokens int, afterTokens int, strategy CompressionStrategy) {
	for _, h := range hs {
		h.OnBudgetCompression(ctx, st, beforeTokens, afterTokens, strategy)
	}
}
func (hs Hooks) OnSoftCapReached(ctx context.Context, st *State, err error) {
	for _, h := range hs {
		h.OnSoftCapReached(ctx, st, err)
	}
}
