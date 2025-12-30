// Package engine provides agent orchestration functionality.
// This file contains error classification and handling.

package engine

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RetryClass indicates whether an error should be retried.
// Used for intelligent retry decision-making.
type RetryClass string

const (
	RetryClassRetryable    RetryClass = "retryable"     // Definitely retry
	RetryClassMaybe        RetryClass = "maybe"         // Retry with caution (limited attempts)
	RetryClassNonRetryable RetryClass = "non_retryable" // Never retry
)

// EngineError wraps errors with classification metadata.
type EngineError struct {
	Err         error
	Class       RetryClass
	HTTPStatus  int    // HTTP status code if applicable
	RetryAfter  string // Retry-After header value if present
	IsRateLimit bool   // True if this is a rate limit error
	IsTimeout   bool   // True if this is a timeout error
	IsNetwork   bool   // True if this is a network error
	IsAuth      bool   // True if this is an authentication error
	IsQuota     bool   // True if this is a quota exhaustion error
}

func (e *EngineError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("engine error: %s", e.Class)
}

func (e *EngineError) Unwrap() error {
	return e.Err
}

// NewEngineError creates a new EngineError with classification.
func NewEngineError(err error, class RetryClass) *EngineError {
	return &EngineError{
		Err:   err,
		Class: class,
	}
}

// ClassifyLLMError classifies an error from an LLM provider call.
func ClassifyLLMError(err error) RetryClass {
	if err == nil {
		return RetryClassNonRetryable
	}

	// Check if it's already an EngineError
	var engineErr *EngineError
	if errors.As(err, &engineErr) {
		return engineErr.Class
	}

	errStr := strings.ToLower(err.Error())

	// Rate limit errors (429) - retryable, respect Retry-After
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") {
		return RetryClassRetryable
	}

	// Server errors (5xx) - retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "internal server error") ||
		strings.Contains(errStr, "bad gateway") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "gateway timeout") {
		return RetryClassRetryable
	}

	// Network/timeout errors - retryable
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dns") ||
		strings.Contains(errStr, "temporary failure") {
		return RetryClassRetryable
	}

	// Context deadline exceeded - maybe (limited retries)
	if strings.Contains(errStr, "context deadline exceeded") ||
		strings.Contains(errStr, "deadline exceeded") {
		return RetryClassMaybe
	}

	// Length/context overflow - maybe (after auto-shrink attempt)
	if strings.Contains(errStr, "context length") ||
		strings.Contains(errStr, "token limit") ||
		strings.Contains(errStr, "maximum context length") {
		return RetryClassMaybe
	}

	// Authentication errors (401, 403) - non-retryable
	if strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "invalid api key") ||
		strings.Contains(errStr, "authentication failed") {
		return RetryClassNonRetryable
	}

	// Bad request (400) - non-retryable
	if strings.Contains(errStr, "400") ||
		strings.Contains(errStr, "bad request") ||
		strings.Contains(errStr, "invalid request") ||
		strings.Contains(errStr, "malformed") {
		return RetryClassNonRetryable
	}

	// Quota exhausted (402) - non-retryable
	if strings.Contains(errStr, "402") ||
		strings.Contains(errStr, "quota") ||
		strings.Contains(errStr, "billing") ||
		strings.Contains(errStr, "payment required") {
		return RetryClassNonRetryable
	}

	// Safety/guardrail refusals - non-retryable
	if strings.Contains(errStr, "content filter") ||
		strings.Contains(errStr, "safety") ||
		strings.Contains(errStr, "guardrail") ||
		strings.Contains(errStr, "policy violation") {
		return RetryClassNonRetryable
	}

	// Default: non-retryable for unknown errors
	return RetryClassNonRetryable
}

// ClassifyToolError classifies an error from a tool execution.
func ClassifyToolError(err error, toolRetryable bool) RetryClass {
	if err == nil {
		return RetryClassNonRetryable
	}

	// If tool is marked as non-retryable, don't retry
	if !toolRetryable {
		return RetryClassNonRetryable
	}

	errStr := strings.ToLower(err.Error())

	// Network/timeout errors - retryable
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "temporary failure") {
		return RetryClassRetryable
	}

	// Server errors (5xx) - retryable
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "internal server error") ||
		strings.Contains(errStr, "service unavailable") {
		return RetryClassRetryable
	}

	// OS/file system errors - retryable (transient)
	if strings.Contains(errStr, "file locked") ||
		strings.Contains(errStr, "resource temporarily unavailable") ||
		strings.Contains(errStr, "spawn") ||
		strings.Contains(errStr, "temporary") {
		return RetryClassRetryable
	}

	// DB deadlocks/retryable errors - retryable
	if strings.Contains(errStr, "deadlock") ||
		strings.Contains(errStr, "retry") ||
		strings.Contains(errStr, "transaction") {
		return RetryClassRetryable
	}

	// Deterministic failures - non-retryable
	if strings.Contains(errStr, "file not found") ||
		strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "invalid input") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "not found") {
		return RetryClassNonRetryable
	}

	// Default: non-retryable for unknown errors
	return RetryClassNonRetryable
}

// ExtractRetryAfter extracts the Retry-After header value from an error.
// Returns 0 if not found or invalid.
func ExtractRetryAfter(err error) time.Duration {
	var engineErr *EngineError
	if errors.As(err, &engineErr) && engineErr.RetryAfter != "" {
		// Try parsing as seconds (integer)
		var seconds int
		if _, err := fmt.Sscanf(engineErr.RetryAfter, "%d", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
		// Try parsing as HTTP date (RFC 1123)
		if t, err := time.Parse(time.RFC1123, engineErr.RetryAfter); err == nil {
			now := time.Now()
			if t.After(now) {
				return t.Sub(now)
			}
		}
	}

	// Check error string for common patterns
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "retry after") {
		// Try to extract number from error message
		var seconds int
		if _, err := fmt.Sscanf(errStr, "retry after %d", &seconds); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}

	return 0
}

// WrapLLMError wraps an LLM provider error with classification metadata.
func WrapLLMError(err error, httpStatus int, retryAfter string) error {
	if err == nil {
		return nil
	}

	class := ClassifyLLMError(err)
	engineErr := &EngineError{
		Err:         err,
		Class:       class,
		HTTPStatus:  httpStatus,
		RetryAfter:  retryAfter,
		IsRateLimit: httpStatus == http.StatusTooManyRequests,
		IsTimeout:   httpStatus == http.StatusGatewayTimeout || httpStatus == http.StatusRequestTimeout,
		IsNetwork:   httpStatus == 0 || httpStatus >= 500,
		IsAuth:      httpStatus == http.StatusUnauthorized || httpStatus == http.StatusForbidden,
		IsQuota:     httpStatus == http.StatusPaymentRequired,
	}

	return engineErr
}

// WrapToolError wraps a tool execution error with classification metadata.
func WrapToolError(err error, toolRetryable bool) error {
	if err == nil {
		return nil
	}

	class := ClassifyToolError(err, toolRetryable)
	engineErr := &EngineError{
		Err:   err,
		Class: class,
	}

	return engineErr
}

// RetryExhaustedError indicates that all retry attempts have been exhausted.
type RetryExhaustedError struct {
	Err         error
	Attempts    int
	MaxAttempts int
	IsGuarded   bool // True if this was a "maybe" class error with limited retries
}

func (e *RetryExhaustedError) Error() string {
	if e.IsGuarded {
		return fmt.Sprintf("guarded retries exhausted after %d attempts: %v", e.Attempts, e.Err)
	}
	return fmt.Sprintf("retries exhausted after %d attempts: %v", e.Attempts, e.Err)
}

func (e *RetryExhaustedError) Unwrap() error {
	return e.Err
}

// NewRetryExhaustedError creates a new RetryExhaustedError.
func NewRetryExhaustedError(err error, attempts, maxAttempts int, isGuarded bool) *RetryExhaustedError {
	return &RetryExhaustedError{
		Err:         err,
		Attempts:    attempts,
		MaxAttempts: maxAttempts,
		IsGuarded:   isGuarded,
	}
}

// IsRetryExhausted checks if an error is a RetryExhaustedError.
func IsRetryExhausted(err error) bool {
	var retryExhausted *RetryExhaustedError
	return errors.As(err, &retryExhausted)
}

// ToolValidationError indicates that tool arguments failed JSON schema validation.
type ToolValidationError struct {
	ToolName string
	Errors   []string
}

func (e *ToolValidationError) Error() string {
	return fmt.Sprintf("tool %s validation failed: %s", e.ToolName, strings.Join(e.Errors, "; "))
}

// EngineContextError wraps errors with execution context (step, phase, tool, operation).
type EngineContextError struct {
	Err       error
	Step      int
	Phase     Phase
	ToolName  string // If error occurred during tool execution
	Operation string // "llm_call", "tool_execution", "compression", etc.
}

func (e *EngineContextError) Error() string {
	if e.ToolName != "" {
		return fmt.Sprintf("[step=%d phase=%s op=%s tool=%s] %v",
			e.Step, e.Phase, e.Operation, e.ToolName, e.Err)
	}
	return fmt.Sprintf("[step=%d phase=%s op=%s] %v",
		e.Step, e.Phase, e.Operation, e.Err)
}

func (e *EngineContextError) Unwrap() error {
	return e.Err
}

// WrapWithContext wraps an error with execution context for debugging.
func WrapWithContext(err error, st *State, operation string, toolName string) error {
	if err == nil {
		return nil
	}
	return &EngineContextError{
		Err:       err,
		Step:      st.Step,
		Phase:     st.Phase,
		ToolName:  toolName,
		Operation: operation,
	}
}
