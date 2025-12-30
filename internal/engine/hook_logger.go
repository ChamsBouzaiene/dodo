// engine/hook_logger.go
package engine

import (
	"context"
	"log"
	"time"
)

type LoggerHook struct{ L *log.Logger }

func (h LoggerHook) OnStepStart(_ context.Context, st *State) {
	h.L.Printf("step=%d phase=%s", st.Step, st.Phase)
}
func (h LoggerHook) OnBeforeLLM(_ context.Context, st *State, msgs []ChatMessage, toolSchemas []ToolSchema) {
	// Count tokens in prompt (estimation)
	tokenizer := GetTokenizerForModel(st.Model)
	messageTokens, _ := CountTokensForMessages(tokenizer, msgs, st.Model)

	// Count tokens for tool schemas
	toolSchemaTokens := 0
	for _, schema := range toolSchemas {
		nameTokens, _ := tokenizer.CountTokens(schema.Name, st.Model)
		descTokens, _ := tokenizer.CountTokens(schema.Description, st.Model)
		schemaTokens, _ := tokenizer.CountTokens(schema.JSONSchema, st.Model)
		toolSchemaTokens += nameTokens + descTokens + schemaTokens + 10 // +10 for overhead per tool
	}

	totalTokens := messageTokens + toolSchemaTokens

	// Show message count and token breakdown only
	historyCount := len(st.History)
	sentCount := len(msgs)
	compressed := historyCount != sentCount

	if compressed {
		h.L.Printf("📤 step=%d: %d msgs (compressed from %d) | 💰 tokens: messages=~%d, tools=~%d, TOTAL=~%d (cumulative=~%d)",
			st.Step, sentCount, historyCount, messageTokens, toolSchemaTokens, totalTokens, st.Totals.Total)
	} else {
		h.L.Printf("📤 step=%d: %d msgs | 💰 tokens: messages=~%d, tools=~%d, TOTAL=~%d (cumulative=~%d)",
			st.Step, sentCount, messageTokens, toolSchemaTokens, totalTokens, st.Totals.Total)
	}
}
func (h LoggerHook) OnAfterLLM(_ context.Context, st *State, r LLMResponse) {
	// Show detailed token breakdown per call
	h.L.Printf("finish=%s tokens: prompt=%d completion=%d total=%d (cumulative=%d)",
		r.FinishReason, r.Usage.Prompt, r.Usage.Completion, r.Usage.Total, st.Totals.Total)
}
func (h LoggerHook) OnToolCall(_ context.Context, _ *State, c ToolCall) {
	// Log tool name and arguments
	h.L.Printf("tool → %s args=%v", c.Name, c.Args)
}
func (h LoggerHook) OnToolResult(_ context.Context, _ *State, c ToolCall, result string, err error) {
	if err != nil {
		h.L.Printf("tool %s error: %v", c.Name, err)
	} else {
		// Log tool result (truncate if too long for readability)
		resultPreview := result
		if len(resultPreview) > 100 {
			resultPreview = resultPreview[:100] + "..."
		}
		h.L.Printf("tool %s result: %s", c.Name, resultPreview)
	}
}
func (h LoggerHook) OnToolOutput(_ context.Context, _ *State, toolName string, output string) {
	// Log tool output (can be verbose, so optionally truncate)
	outputPreview := output
	if len(outputPreview) > 200 {
		outputPreview = outputPreview[:200] + "..."
	}
	h.L.Printf("tool %s output: %s", toolName, outputPreview)
}
func (h LoggerHook) OnStreamDelta(_ context.Context, _ *State, d string) { /* stream to TUI */ }
func (h LoggerHook) OnDone(_ context.Context, st *State) {
	h.L.Printf("done: steps=%d tokens=%d", st.Step, st.Totals.Total)
}
func (h LoggerHook) OnHistoryChanged(_ context.Context, _ *State)                {}
func (h LoggerHook) OnSummarize(_ context.Context, _ *State, _, _ []ChatMessage) {}
func (h LoggerHook) OnRetryAttempt(_ context.Context, st *State, attempt int, maxAttempts int, delay time.Duration, err error) {
	st.Retries++
	h.L.Printf("retry attempt=%d/%d delay=%v error=%v", attempt, maxAttempts, delay, err)
}
func (h LoggerHook) OnRetryExhausted(_ context.Context, st *State, err error) {
	h.L.Printf("retries exhausted: %v", err)
}
func (h LoggerHook) OnBudgetExceeded(_ context.Context, st *State, tokenCount int, softLimit int, hardLimit int) {
	h.L.Printf("budget exceeded: tokens=%d soft_limit=%d hard_limit=%d", tokenCount, softLimit, hardLimit)
}
func (h LoggerHook) OnBudgetCompression(_ context.Context, st *State, beforeTokens int, afterTokens int, strategy CompressionStrategy) {
	strategyName := []string{"truncate", "summarize", "aggressive_summarize", "remove"}[strategy]
	h.L.Printf("budget compression: %s before=%d after=%d reduction=%.1f%%", strategyName, beforeTokens, afterTokens, float64(beforeTokens-afterTokens)/float64(beforeTokens)*100)
}
func (h LoggerHook) OnSoftCapReached(_ context.Context, st *State, err error) {
	h.L.Printf("⚠️  soft cap reached: %v", err)
}

// For metrics, expose counters/gauges and plug into Prometheus later.
