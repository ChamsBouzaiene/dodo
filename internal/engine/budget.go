// Package engine provides agent orchestration functionality.
// This file contains budget management and message compression logic.

package engine

import (
	"context"
	"fmt"
)

// CompressionStrategy represents a compression approach.
type CompressionStrategy int

const (
	CompressionTruncate CompressionStrategy = iota
	CompressionSummarize
	CompressionAggressiveSummarize
	CompressionRemove
)

// compressionResult tracks the result of a compression attempt.
type compressionResult struct {
	messages    []ChatMessage
	tokenCount  int
	strategy    CompressionStrategy
	success     bool
	description string
}

// compressWithTruncate truncates long tool outputs.
func compressWithTruncate(ctx context.Context, msgs []ChatMessage, maxChars int) ([]ChatMessage, error) {
	result := make([]ChatMessage, 0, len(msgs))
	modified := false
	
	for _, msg := range msgs {
		if msg.Role == RoleTool && len(msg.Content) > maxChars {
			head := msg.Content[:maxChars/2]
			tail := msg.Content[len(msg.Content)-maxChars/2:]
			msg.Content = head + "\n...\n" + tail
			modified = true
		}
		result = append(result, msg)
	}
	
	if !modified {
		return nil, fmt.Errorf("truncation did not reduce size")
	}
	
	return result, nil
}

// compressWithSummarize summarizes old messages while keeping recent ones.
func compressWithSummarize(ctx context.Context, llm LLMClient, st *State, msgs []ChatMessage, keepLastN int) ([]ChatMessage, error) {
	if len(msgs) <= keepLastN {
		return nil, fmt.Errorf("not enough messages to summarize")
	}
	
	oldMsgs := msgs[:len(msgs)-keepLastN]
	recentMsgs := msgs[len(msgs)-keepLastN:]
	
	summary, err := SummarizeOld(ctx, llm, st, oldMsgs)
	if err != nil {
		return nil, fmt.Errorf("summarization failed: %w", err)
	}
	
	return append([]ChatMessage{summary}, recentMsgs...), nil
}

// compressWithAggressiveSummarize applies more aggressive summarization.
func compressWithAggressiveSummarize(ctx context.Context, llm LLMClient, st *State, msgs []ChatMessage, keepLastN int) ([]ChatMessage, error) {
	if len(msgs) <= keepLastN {
		return nil, fmt.Errorf("not enough messages to summarize")
	}
	
	// Keep only system message (if present) and last N messages
	var systemMsg *ChatMessage
	var otherMsgs []ChatMessage
	
	for _, msg := range msgs {
		if msg.Role == RoleSystem {
			systemMsg = &msg
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}
	
	// Summarize everything except last N
	if len(otherMsgs) <= keepLastN {
		return nil, fmt.Errorf("not enough messages for aggressive summarization")
	}
	
	toSummarize := otherMsgs[:len(otherMsgs)-keepLastN]
	keepRecent := otherMsgs[len(otherMsgs)-keepLastN:]
	
	summary, err := SummarizeOld(ctx, llm, st, toSummarize)
	if err != nil {
		return nil, fmt.Errorf("aggressive summarization failed: %w", err)
	}
	
	result := []ChatMessage{summary}
	if systemMsg != nil {
		result = append(result, *systemMsg)
	}
	result = append(result, keepRecent...)
	
	return result, nil
}

// compressWithRemove removes non-essential messages, keeping only system and last few.
func compressWithRemove(ctx context.Context, msgs []ChatMessage, keepLastN int) ([]ChatMessage, error) {
	if len(msgs) <= keepLastN {
		return nil, fmt.Errorf("not enough messages to remove")
	}
	
	var systemMsg *ChatMessage
	var otherMsgs []ChatMessage
	
	for _, msg := range msgs {
		if msg.Role == RoleSystem {
			systemMsg = &msg
		} else {
			otherMsgs = append(otherMsgs, msg)
		}
	}
	
	// Keep only last N non-system messages
	if len(otherMsgs) <= keepLastN {
		return nil, fmt.Errorf("not enough messages to remove")
	}
	
	keepRecent := otherMsgs[len(otherMsgs)-keepLastN:]
	
	result := keepRecent
	if systemMsg != nil {
		result = append([]ChatMessage{*systemMsg}, result...)
	}
	
	return result, nil
}

// compressMessages applies a compression strategy to reduce message size.
func compressMessages(ctx context.Context, llm LLMClient, st *State, msgs []ChatMessage, strategy CompressionStrategy) ([]ChatMessage, error) {
	switch strategy {
	case CompressionTruncate:
		return compressWithTruncate(ctx, msgs, 2000) // Truncate to 2000 chars
	case CompressionSummarize:
		return compressWithSummarize(ctx, llm, st, msgs, 8)
	case CompressionAggressiveSummarize:
		return compressWithAggressiveSummarize(ctx, llm, st, msgs, 4)
	case CompressionRemove:
		return compressWithRemove(ctx, msgs, 4)
	default:
		return nil, fmt.Errorf("unknown compression strategy: %v", strategy)
	}
}

// compressUntilUnderBudget iteratively compresses messages until they fit within budget.
// Returns compressed messages and token count, or error if hard limit cannot be met.
// onCompression callback is called for each successful compression attempt.
func compressUntilUnderBudget(
	ctx context.Context,
	llm LLMClient,
	st *State,
	msgs []ChatMessage,
	budget BudgetConfig,
	tokenizer Tokenizer,
	onCompression func(beforeTokens, afterTokens int, strategy CompressionStrategy),
) ([]ChatMessage, int, error) {
	if budget.HardLimit <= 0 {
		// No budget limit, return as-is
		tokenCount, err := CountTokensForMessages(tokenizer, msgs, st.Model)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count tokens: %w", err)
		}
		return msgs, tokenCount, nil
	}
	
	// Calculate effective hard limit (accounting for reserved tokens)
	effectiveHardLimit := budget.HardLimit - budget.ReserveTokens
	if effectiveHardLimit <= 0 {
		return nil, 0, fmt.Errorf("hard limit too small after reserving %d tokens", budget.ReserveTokens)
	}
	
	currentMsgs := msgs
	attempts := 0
	attemptedStrategies := make(map[CompressionStrategy]bool)
	
	// Compression strategies in order of aggressiveness
	strategies := []CompressionStrategy{
		CompressionTruncate,
		CompressionSummarize,
		CompressionAggressiveSummarize,
		CompressionRemove,
	}
	
	for attempts < budget.MaxCompressionPasses {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}
		
		// Count tokens
		tokenCount, err := CountTokensForMessages(tokenizer, currentMsgs, st.Model)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count tokens: %w", err)
		}

		// Check if we're under the hard limit
		if tokenCount <= effectiveHardLimit {
			return currentMsgs, tokenCount, nil
		}
		
		// Try compression strategies in order
		compressed := false
		for _, strategy := range strategies {
			// Skip if we've already tried this strategy
			if attemptedStrategies[strategy] {
				continue
			}
			
			newMsgs, err := compressMessages(ctx, llm, st, currentMsgs, strategy)
			if err != nil {
				// Strategy failed, mark as attempted and try next
				attemptedStrategies[strategy] = true
				continue
			}
			
			// Check if compression actually reduced tokens
			newTokenCount, err := CountTokensForMessages(tokenizer, newMsgs, st.Model)
			if err != nil {
				attemptedStrategies[strategy] = true
				continue
			}
			
			if newTokenCount < tokenCount {
				currentMsgs = newMsgs
				compressed = true
				attemptedStrategies[strategy] = true
				// Call compression callback
				if onCompression != nil {
					onCompression(tokenCount, newTokenCount, strategy)
				}
				break
			} else {
				// Compression didn't help, mark as attempted
				attemptedStrategies[strategy] = true
			}
		}
		
		if !compressed {
			// No compression strategy helped, we're stuck
			return nil, tokenCount, &BudgetError{
				RequiredTokens: tokenCount,
				HardLimit:      budget.HardLimit,
				Attempts:       attempts,
			}
		}
		
		attempts++
	}
	
	// Max attempts reached, check final token count
	finalTokenCount, err := CountTokensForMessages(tokenizer, currentMsgs, st.Model)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count final tokens: %w", err)
	}
	
	if finalTokenCount > effectiveHardLimit {
		return nil, finalTokenCount, &BudgetError{
			RequiredTokens: finalTokenCount,
			HardLimit:      budget.HardLimit,
			Attempts:       attempts,
		}
	}
	
	return currentMsgs, finalTokenCount, nil
}

