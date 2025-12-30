// Package engine provides agent orchestration functionality.
// This file contains token counting interfaces and implementations.

package engine

import (
	"fmt"
	"strings"
)

// Tokenizer provides token counting for text.
// Different models use different tokenization schemes, so the model name is required.
type Tokenizer interface {
	// CountTokens returns the number of tokens in the given text for the specified model.
	// Returns an error if tokenization fails or the model is not supported.
	CountTokens(text string, model string) (int, error)
}

// EstimateTokens provides a rough token count estimation.
// Uses a simple heuristic: ~4 characters per token for English/code.
// This is approximate but useful for logging and analysis.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}

	// Rough estimation: ~4 characters per token
	// This is a conservative estimate for English/code
	// Actual tokenization varies by model, but this gives a reasonable approximation
	charCount := len([]rune(text))

	// Account for whitespace and special characters
	// Code tends to have more tokens per character due to syntax
	whitespaceCount := strings.Count(text, " ") + strings.Count(text, "\n") + strings.Count(text, "\t")

	// Rough formula: (characters / 4) + (whitespace / 6)
	// This accounts for the fact that whitespace-heavy text has fewer tokens
	estimated := (charCount / 4) + (whitespaceCount / 6)

	// Minimum of 1 token for non-empty text
	if estimated < 1 {
		return 1
	}

	return estimated
}

// DefaultTokenizer uses estimation as a fallback when no specific tokenizer is available.
type DefaultTokenizer struct{}

// CountTokens implements Tokenizer using estimation.
func (t DefaultTokenizer) CountTokens(text string, model string) (int, error) {
	return EstimateTokens(text), nil
}

// CountTokensForMessages counts tokens for a slice of messages.
// It includes formatting overhead (role names, separators) in the count.
func CountTokensForMessages(tokenizer Tokenizer, messages []ChatMessage, model string) (int, error) {
	total := 0

	for _, msg := range messages {
		// Count role token (e.g., "assistant", "user", "system")
		roleTokens, err := tokenizer.CountTokens(string(msg.Role), model)
		if err != nil {
			return 0, fmt.Errorf("failed to count role tokens: %w", err)
		}
		total += roleTokens

		// Count content tokens
		contentTokens, err := tokenizer.CountTokens(msg.Content, model)
		if err != nil {
			return 0, fmt.Errorf("failed to count content tokens: %w", err)
		}
		total += contentTokens

		// Count tool calls if present (rough estimate: tool name + args)
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				nameTokens, err := tokenizer.CountTokens(tc.Name, model)
				if err != nil {
					return 0, fmt.Errorf("failed to count tool call name tokens: %w", err)
				}
				total += nameTokens

				// Estimate args tokens (convert to JSON string for counting)
				argsStr := fmt.Sprintf("%v", tc.Args)
				argsTokens, err := tokenizer.CountTokens(argsStr, model)
				if err != nil {
					return 0, fmt.Errorf("failed to count tool call args tokens: %w", err)
				}
				total += argsTokens
			}
		}

		// Add overhead for message formatting (approximately 4 tokens per message)
		total += 4
	}

	return total, nil
}

// GetTokenizerForModel returns an appropriate tokenizer for the given model.
// Currently returns DefaultTokenizer (estimation), but can be extended to support
// provider-specific tokenizers like tiktoken for OpenAI models.
func GetTokenizerForModel(model string) Tokenizer {
	// Check if model is OpenAI (can add tiktoken support later)
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") {
		// TODO: Return TikTokenTokenizer when implemented
		return DefaultTokenizer{}
	}

	// Default to estimation for all other models
	return DefaultTokenizer{}
}
