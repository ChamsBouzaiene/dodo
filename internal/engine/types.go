package engine

import (
	"context"
	"fmt"
)

// MessageRole represents the role of a chat message.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// ChatMessage is the provider-agnostic message we pass around.
type ChatMessage struct {
	Role    MessageRole // Role of the message sender
	Content string      // Message content
	Name    string      // Optional: tool name for tool messages
	// ToolCalls stores the actual tool calls made by this assistant message
	// This is needed when converting back to provider format (providers require tool_calls in assistant messages)
	ToolCalls []ToolCall // Tool calls made in this assistant message (if any)
}

// Validate checks if the ChatMessage is valid.
func (m ChatMessage) Validate() error {
	switch m.Role {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		// Valid roles
	default:
		return fmt.Errorf("invalid message role: %s", m.Role)
	}
	if m.Role == RoleTool && m.Name == "" {
		return fmt.Errorf("tool messages must have a Name field")
	}
	return nil
}

// Usage holds token accounting returned by providers.
type Usage struct {
	Prompt     int
	Completion int
	Total      int
}

// ToolCall represents a function/tool the assistant requested.
type ToolCall struct {
	ID    string // Provider-specific tool call ID (e.g., OpenAI's call_xxx)
	Name  string
	Args  map[string]any
	Error string // Set by provider if tool call is incomplete/invalid (e.g., stream ended prematurely)
}

// LLMResponse is a normalized result of one chat call.
type LLMResponse struct {
	Assistant    ChatMessage
	ToolCalls    []ToolCall // zero or more tool calls requested by the model
	Usage        Usage
	FinishReason string // "stop" | "length" | "tool_calls" | "content_filter" | "tool_error"
}

// LLMClient abstracts your chosen SDK (OpenAI, Anthropic, etc.)
type LLMClient interface {
	Chat(ctx context.Context, model string, messages []ChatMessage, toolSchemas []ToolSchema, opts ChatOptions) (LLMResponse, error)
	// Optional streaming variant (we’ll wire later):
	Stream(ctx context.Context, model string, messages []ChatMessage, toolSchemas []ToolSchema, opts ChatOptions) (<-chan StreamEvent, <-chan error)
}

// ChatOptions keeps knobs you'll forward to the SDK.
type ChatOptions struct {
	Temperature       float32
	MaxOutputTokens   int
	RetryConfig       *RetryConfig       // Optional retry configuration (nil = use defaults)
	Stream            bool               // Enable streaming mode (default: false, opt-in)
	CompressionConfig *CompressionConfig // Optional compression configuration (nil = use defaults)
	// You can add system prompt cache keys, top_p, etc.
}

// ToolSchema is the JSON schema (or similar) the provider expects for function calling.
type ToolSchema struct {
	Name        string
	Description string
	JSONSchema  string // keep as raw JSON string for simplicity
	Retryable   bool   // Whether this tool can be retried (default: true for idempotent tools)
}

// StreamEvent represents a streaming event from the LLM.
type StreamEvent struct {
	Type       string   // "text_delta" | "tool_call" | "tool_result" | "usage"
	Text       string   // for text_delta
	ToolCall   ToolCall // for tool_call
	ToolCallID string   // for tool_result (ID of the tool call this result belongs to)
	Content    string   // for tool_result (error message or result)
	Usage      Usage    // for usage
}

// BudgetConfig defines token budget limits and compression settings.
type BudgetConfig struct {
	SoftLimit            int // Target limit - warn if exceeded but allow
	HardLimit            int // Must not exceed - fail if cannot compress below this
	MaxCompressionPasses int // Maximum number of compression iterations (prevent infinite loops)
	ReserveTokens        int // Reserve tokens for response (e.g., 1000 tokens)
}

// DefaultBudgetConfig returns sensible default budget configuration.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		SoftLimit:            12000, // Increased from 4000 - modern models handle this fine
		HardLimit:            16000, // Increased from 8000 - avoid premature compression
		MaxCompressionPasses: 5,
		ReserveTokens:        2000, // Increased from 1000 for larger tool calls
	}
}

// BudgetError indicates that budget limits could not be met.
type BudgetError struct {
	RequiredTokens int
	HardLimit      int
	Attempts       int
}

func (e *BudgetError) Error() string {
	return fmt.Sprintf("budget exceeded: required %d tokens, hard limit %d (after %d compression attempts)", e.RequiredTokens, e.HardLimit, e.Attempts)
}

// ToolStreamer allows tools to stream output back to the engine.
type ToolStreamer interface {
	Stream(output string)
}

// ExecutionResult represents the standard format for execution tool results.
// All execution tools (run_cmd, run_tests, run_build) should return JSON
// that unmarshals to this structure. This provides a contract between tools
// and the protocol layer, preventing coupling to implementation details.
type ExecutionResult struct {
	Cmd            string `json:"cmd"`              // Command that was executed
	ExitCode       int    `json:"exit_code"`       // Exit code (0 = success)
	Stdout         string `json:"stdout"`          // Standard output
	Stderr         string `json:"stderr"`          // Standard error output
	TimedOut       bool   `json:"timed_out,omitempty"`       // Whether command timed out
	Status         string `json:"status,omitempty"`          // Status: "ok", "failed", "unavailable"
	Reason         string `json:"reason,omitempty"`          // Reason for status (e.g., "command_not_found")
	Passed         *bool  `json:"passed,omitempty"`         // For test tools: whether tests passed
	StdoutTruncated bool  `json:"stdout_truncated,omitempty"` // Whether stdout was truncated
	StderrTruncated bool  `json:"stderr_truncated,omitempty"` // Whether stderr was truncated
}
