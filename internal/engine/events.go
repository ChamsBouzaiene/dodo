package engine

import (
	"context"
	"time"
)

type Event struct {
	Kind string // "step_start", "delta", "tool_start", "tool_output", "tool_done", "done", "retry_attempt", "retry_exhausted"
	Data any
}

// TUIHook bridges engine → TUI channel
type TUIHook struct{ Ch chan<- Event }

func (h TUIHook) OnStepStart(_ context.Context, st *State) {
	h.Ch <- Event{Kind: "step_start", Data: st.Step}
}
func (h TUIHook) OnBeforeLLM(_ context.Context, st *State, m []ChatMessage, schemas []ToolSchema) {
	h.Ch <- Event{Kind: "before_llm", Data: map[string]any{"messages": len(m), "tools": len(schemas)}}
}
func (h TUIHook) OnAfterLLM(_ context.Context, st *State, r LLMResponse) {
	h.Ch <- Event{Kind: "after_llm", Data: r.FinishReason}
}
func (h TUIHook) OnStreamDelta(_ context.Context, _ *State, d string) {
	h.Ch <- Event{Kind: "delta", Data: d}
}
func (h TUIHook) OnToolCall(_ context.Context, _ *State, c ToolCall) {
	h.Ch <- Event{Kind: "tool_start", Data: c.Name}
}
func (h TUIHook) OnToolResult(_ context.Context, _ *State, c ToolCall, _ string, _ error) {
	h.Ch <- Event{Kind: "tool_done", Data: c.Name}
}
func (h TUIHook) OnToolOutput(_ context.Context, _ *State, toolName string, output string) {
	h.Ch <- Event{Kind: "tool_output", Data: map[string]string{"tool": toolName, "output": output}}
}
func (h TUIHook) OnHistoryChanged(_ context.Context, st *State) {
	h.Ch <- Event{Kind: "history_changed", Data: len(st.History)}
}
func (h TUIHook) OnSummarize(_ context.Context, st *State, before, after []ChatMessage) {
	h.Ch <- Event{Kind: "summarize", Data: map[string]int{"before": len(before), "after": len(after)}}
}
func (h TUIHook) OnDone(_ context.Context, st *State) {
	h.Ch <- Event{Kind: "done", Data: st.Totals}
}
func (h TUIHook) OnRetryAttempt(_ context.Context, st *State, attempt int, maxAttempts int, delay time.Duration, err error) {
	h.Ch <- Event{Kind: "retry_attempt", Data: map[string]any{
		"attempt":     attempt,
		"maxAttempts": maxAttempts,
		"delay":       delay,
		"error":       err.Error(),
	}}
}
func (h TUIHook) OnRetryExhausted(_ context.Context, st *State, err error) {
	h.Ch <- Event{Kind: "retry_exhausted", Data: err.Error()}
}
func (h TUIHook) OnBudgetExceeded(_ context.Context, st *State, tokenCount int, softLimit int, hardLimit int) {
	h.Ch <- Event{Kind: "budget_exceeded", Data: map[string]int{
		"tokens":    tokenCount,
		"softLimit": softLimit,
		"hardLimit": hardLimit,
	}}
}
func (h TUIHook) OnBudgetCompression(_ context.Context, st *State, beforeTokens int, afterTokens int, strategy CompressionStrategy) {
	strategyName := []string{"truncate", "summarize", "aggressive_summarize", "remove"}[strategy]
	h.Ch <- Event{Kind: "budget_compression", Data: map[string]any{
		"before":   beforeTokens,
		"after":    afterTokens,
		"strategy": strategyName,
	}}
}
