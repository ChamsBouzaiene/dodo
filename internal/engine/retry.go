package engine

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy defines retry behavior for a specific operation type.
type RetryPolicy struct {
	MaxRetries   int           // Maximum number of retry attempts (0 = no retries)
	InitialDelay time.Duration // Initial delay before first retry
	MaxDelay     time.Duration // Maximum delay cap
	Multiplier   float64       // Exponential backoff multiplier (e.g., 2.0)
	Jitter       bool          // Whether to add random jitter to delays
}

// RetryConfig holds separate retry policies for LLM and tool calls.
type RetryConfig struct {
	LLMPolicy  RetryPolicy // Policy for LLM API calls
	ToolPolicy RetryPolicy // Policy for tool executions
}

// DefaultRetryConfig is defined in config.go for centralized configuration management.

// RetryableFunc is a function that can be retried.
type RetryableFunc[T any] func(ctx context.Context) (T, error)

// RetryWithPolicy executes a function with retry logic based on the policy.
// Returns the result on success, or the last error if all retries are exhausted.
func RetryWithPolicy[T any](
	ctx context.Context,
	policy RetryPolicy,
	fn RetryableFunc[T],
	classifyError func(error) RetryClass,
	onRetry func(attempt int, delay time.Duration, err error),
) (T, error) {
	var zero T

	attempt := 0

	for {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		// Classify the error
		class := classifyError(err)
		if class == RetryClassNonRetryable {
			return zero, err
		}

		// Check if we've exhausted retries
		if attempt >= policy.MaxRetries {
			return zero, NewRetryExhaustedError(err, attempt, policy.MaxRetries, false)
		}

		// For "maybe" class, limit to 1-2 retries
		if class == RetryClassMaybe && attempt >= 2 {
			return zero, NewRetryExhaustedError(err, attempt, 2, true)
		}

		// Calculate delay
		delay := calculateDelay(policy, attempt, err)

		// Call retry hook if provided
		if onRetry != nil {
			onRetry(attempt+1, delay, err)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("context cancelled during retry: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}

		attempt++
	}
}

// calculateDelay computes the delay for a retry attempt.
func calculateDelay(policy RetryPolicy, attempt int, err error) time.Duration {
	// Check for Retry-After header
	retryAfter := ExtractRetryAfter(err)
	if retryAfter > 0 {
		// Use Retry-After if present, but cap at MaxDelay
		if retryAfter > policy.MaxDelay {
			return policy.MaxDelay
		}
		return retryAfter
	}

	// Exponential backoff: initialDelay * (multiplier ^ attempt)
	delay := float64(policy.InitialDelay) * math.Pow(policy.Multiplier, float64(attempt))

	// Cap at MaxDelay
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}

	// Add jitter if enabled (0-20% random variation)
	if policy.Jitter {
		jitter := rand.Float64() * 0.2 * delay // 0-20% jitter
		delay += jitter
	}

	return time.Duration(delay)
}

// ShouldRetry determines if an error should be retried based on classification and policy.
func ShouldRetry(err error, policy RetryPolicy, classifyError func(error) RetryClass) bool {
	if err == nil {
		return false
	}

	class := classifyError(err)
	if class == RetryClassNonRetryable {
		return false
	}

	// For "maybe" class, we'll retry but with limited attempts (handled in RetryWithPolicy)
	return true
}

// RetryLLMCall wraps an LLM call with retry logic.
func RetryLLMCall(
	ctx context.Context,
	policy RetryPolicy,
	llm LLMClient,
	model string,
	messages []ChatMessage,
	toolSchemas []ToolSchema,
	opts ChatOptions,
	onRetry func(attempt int, delay time.Duration, err error),
) (LLMResponse, error) {
	return RetryWithPolicy(
		ctx,
		policy,
		func(ctx context.Context) (LLMResponse, error) {
			return llm.Chat(ctx, model, messages, toolSchemas, opts)
		},
		ClassifyLLMError,
		onRetry,
	)
}

// RetryToolCall wraps a tool call with retry logic.
func RetryToolCall(
	ctx context.Context,
	policy RetryPolicy,
	call ToolCall,
	reg ToolRegistry,
	onRetry func(attempt int, delay time.Duration, err error),
) (string, error) {
	// Check if tool is retryable
	tool, ok := reg[call.Name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", call.Name)
	}

	// Check if tool is retryable
	// Note: Since Retryable is a bool, zero value is false.
	// We default to retryable (true) for most tools since they're idempotent.
	// Tools should explicitly set Retryable=false for non-idempotent operations.
	toolRetryable := tool.Retryable
	// If Retryable is false (either unset or explicitly false), we still allow retries
	// by default since most tools are idempotent. Only disable if explicitly set to false
	// and the tool author knows it's non-idempotent.
	// For now, we respect the value: if false, don't retry.
	if !toolRetryable {
		// Tool is marked as non-retryable, use a policy with 0 retries
		policy = RetryPolicy{MaxRetries: 0}
	}

	return RetryWithPolicy(
		ctx,
		policy,
		func(ctx context.Context) (string, error) {
			return executeTool(ctx, call, reg)
		},
		func(err error) RetryClass {
			return ClassifyToolError(err, toolRetryable)
		},
		onRetry,
	)
}
