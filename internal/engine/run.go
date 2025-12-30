package engine

import (
	"context"
	"fmt"
)

// Run executes the ReAct loop until completion, max steps reached, or an error occurs.
// It orchestrates multiple reasoning/acting cycles, handling retries internally.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - llm: LLM client for making chat completion calls
//   - reg: Registry of available tools
//   - st: Engine state (history, step count, etc.) - modified in place
//   - hooks: Observability hooks for monitoring execution
//   - opts: Chat options including retry configuration
//
// Returns:
//   - error: Non-nil if execution fails (retries exhausted, non-retryable error, etc.)
//
// Step counting: Steps increment only on successful completion. Retries are tracked separately.
func Run(ctx context.Context, llm LLMClient, reg ToolRegistry, st *State, hooks Hooks, opts ChatOptions) error {
	// Initialize step counter
	st.Step = 0

	for st.Step < st.MaxSteps && !st.Done {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}
		
		// Check soft caps BEFORE step
		if err := checkSoftCaps(st); err != nil {
			// Gracefully stop with explanation
			hooks.OnSoftCapReached(ctx, st, err)
			return err // Return error to stop execution
		}
		
		err := stepOnce(ctx, llm, reg, st, hooks, opts)
		if err != nil {
			// stepOnce handles retries internally, so if we get here:
			// - Retries were exhausted (RetryExhaustedError)
			// - Non-retryable error occurred
			// In either case, return the error without incrementing step
			return err
		}
		// Success: increment step only after successful completion
		st.Step++
	}
	if st.Done {
		hooks.OnDone(ctx, st)
	}
	return nil
}

// RunStream executes the ReAct loop with streaming support.
// It uses stepOnceStream instead of stepOnce to enable incremental output.
// This is opt-in - existing code using Run() is unaffected.
//
// Parameters:
//   - ctx: Context for cancellation and timeouts
//   - llm: LLM client for making streaming chat completion calls
//   - reg: Registry of available tools
//   - st: Engine state (history, step count, etc.) - modified in place
//   - hooks: Observability hooks for monitoring execution (OnStreamDelta will be called)
//   - opts: Chat options including retry configuration
//
// Returns:
//   - error: Non-nil if execution fails (retries exhausted, non-retryable error, etc.)
//
// Step counting: Steps increment only on successful completion. Retries are tracked separately.
func RunStream(ctx context.Context, llm LLMClient, reg ToolRegistry, st *State, hooks Hooks, opts ChatOptions) error {
	// Initialize step counter
	st.Step = 0

	for st.Step < st.MaxSteps && !st.Done {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("execution cancelled: %w", ctx.Err())
		default:
		}
		
		// Check soft caps BEFORE step
		if err := checkSoftCaps(st); err != nil {
			// Gracefully stop with explanation
			hooks.OnSoftCapReached(ctx, st, err)
			return err // Return error to stop execution
		}
		
		err := stepOnceStream(ctx, llm, reg, st, hooks, opts)
		if err != nil {
			// stepOnceStream handles retries internally, so if we get here:
			// - Retries were exhausted (RetryExhaustedError)
			// - Non-retryable error occurred
			// In either case, return the error without incrementing step
			return err
		}
		// Success: increment step only after successful completion
		st.Step++
	}
	if st.Done {
		hooks.OnDone(ctx, st)
	}
	return nil
}
