// engine/processors.go
package engine

import (
	"context"
	"fmt"
	"strings"
)

type Processor func(ctx context.Context, st *State, msgs []ChatMessage) ([]ChatMessage, error)

func ApplyProcessors(ctx context.Context, st *State, msgs []ChatMessage, ps ...Processor) ([]ChatMessage, error) {
	var err error
	for _, p := range ps {
		msgs, err = p(ctx, st, msgs)
		if err != nil {
			return msgs, err
		}
	}
	return msgs, nil
}

// KeepLastN keeps last N messages raw; summarize/truncate older ones later.
// IMPORTANT: Always preserves system message and first user message to maintain context.
func KeepLastN(n int) Processor {
	return func(ctx context.Context, st *State, msgs []ChatMessage) ([]ChatMessage, error) {
		if len(msgs) <= n {
			return msgs, nil
		}

		// Always preserve system message (first message) and first user message
		// This ensures the agent never loses track of the original task
		var preserved []ChatMessage
		var firstUserIdx = -1

		// Find system message (should be first) and first user message
		for i, msg := range msgs {
			if i == 0 && msg.Role == RoleSystem {
				preserved = append(preserved, msg)
			} else if msg.Role == RoleUser && firstUserIdx == -1 {
				firstUserIdx = i
				preserved = append(preserved, msg)
			}
		}

		// Get last N messages
		recentMsgs := msgs[len(msgs)-n:]

		// Merge preserved messages with recent messages, avoiding duplicates
		result := make([]ChatMessage, 0, len(preserved)+len(recentMsgs))

		// Add preserved messages first
		for _, msg := range preserved {
			result = append(result, msg)
		}

		// Add recent messages, skipping ones we already preserved
		for _, msg := range recentMsgs {
			// Check if this message is already in preserved (by comparing content)
			isPreserved := false
			for _, p := range preserved {
				if p.Role == msg.Role && p.Content == msg.Content {
					isPreserved = true
					break
				}
			}
			if !isPreserved {
				result = append(result, msg)
			}
		}

		return result, nil
	}
}

// TruncateLongTools trims huge tool outputs (keep head/tail lines).
func TruncateLongTools(maxChars int) Processor {
	return func(ctx context.Context, st *State, msgs []ChatMessage) ([]ChatMessage, error) {
		out := make([]ChatMessage, 0, len(msgs))
		for _, m := range msgs {
			if m.Role == RoleTool && len(m.Content) > maxChars {
				head := m.Content[:maxChars/2]
				tail := m.Content[len(m.Content)-maxChars/2:]
				m.Content = head + "\n...\n" + tail
			}
			out = append(out, m)
		}
		return out, nil
	}
}

// SummarizeOlderThanN summarizes messages older than the last N messages.
// It uses the LLM to compress old history while keeping recent messages intact.
// IMPORTANT: Always preserves system message and first user message.
func SummarizeOlderThanN(llm LLMClient, keepLastN int) Processor {
	return func(ctx context.Context, st *State, msgs []ChatMessage) ([]ChatMessage, error) {
		if len(msgs) <= keepLastN {
			return msgs, nil
		}

		// Always preserve system message (first) and first user message
		var preserved []ChatMessage
		var firstUserIdx = -1

		for i, msg := range msgs {
			if i == 0 && msg.Role == RoleSystem {
				preserved = append(preserved, msg)
			} else if msg.Role == RoleUser && firstUserIdx == -1 {
				firstUserIdx = i
				preserved = append(preserved, msg)
			}
		}

		// Split into old (to summarize) and recent (to keep)
		// But exclude preserved messages from summarization
		oldMsgs := msgs[:len(msgs)-keepLastN]
		recentMsgs := msgs[len(msgs)-keepLastN:]

		// Filter out preserved messages from oldMsgs (they'll be added back)
		oldMsgsToSummarize := make([]ChatMessage, 0, len(oldMsgs))
		for _, msg := range oldMsgs {
			isPreserved := false
			for _, p := range preserved {
				if p.Role == msg.Role && p.Content == msg.Content {
					isPreserved = true
					break
				}
			}
			if !isPreserved {
				oldMsgsToSummarize = append(oldMsgsToSummarize, msg)
			}
		}

		// Only summarize if there are messages to summarize
		if len(oldMsgsToSummarize) == 0 {
			// Nothing to summarize, just return preserved + recent
			result := make([]ChatMessage, 0, len(preserved)+len(recentMsgs))
			result = append(result, preserved...)
			for _, msg := range recentMsgs {
				isPreserved := false
				for _, p := range preserved {
					if p.Role == msg.Role && p.Content == msg.Content {
						isPreserved = true
						break
					}
				}
				if !isPreserved {
					result = append(result, msg)
				}
			}
			return result, nil
		}

		// Summarize old messages (excluding preserved ones)
		summary, err := SummarizeOld(ctx, llm, st, oldMsgsToSummarize)
		if err != nil {
			// If summarization fails, keep preserved + recent messages
			result := make([]ChatMessage, 0, len(preserved)+len(recentMsgs))
			result = append(result, preserved...)
			for _, msg := range recentMsgs {
				isPreserved := false
				for _, p := range preserved {
					if p.Role == msg.Role && p.Content == msg.Content {
						isPreserved = true
						break
					}
				}
				if !isPreserved {
					result = append(result, msg)
				}
			}
			return result, nil
		}

		// Combine: preserved messages + summary + recent messages
		result := make([]ChatMessage, 0, len(preserved)+1+len(recentMsgs))
		result = append(result, preserved...)
		result = append(result, summary)
		for _, msg := range recentMsgs {
			isPreserved := false
			for _, p := range preserved {
				if p.Role == msg.Role && p.Content == msg.Content {
					isPreserved = true
					break
				}
			}
			if !isPreserved {
				result = append(result, msg)
			}
		}
		return result, nil
	}
}

// KeepRecentToolCalls keeps the last N tool interaction cycles in full detail
// and compresses older tool calls into compact summaries.
// This is inspired by SWE-agent's history compression approach.
//
// A tool cycle consists of: assistant message with tool calls + tool result messages
func KeepRecentToolCalls(keepCount int) Processor {
	return func(ctx context.Context, st *State, msgs []ChatMessage) ([]ChatMessage, error) {
		if len(msgs) <= keepCount+5 { // +5 for system prompt and some buffer
			return msgs, nil // Too short to compress
		}

		// Identify system prompt (always keep)
		var systemMsg *ChatMessage
		if len(msgs) > 0 && msgs[0].Role == RoleSystem {
			systemMsg = &msgs[0]
		}

		// Group messages into tool cycles
		cycles := identifyToolCycles(msgs)

		if len(cycles) <= keepCount {
			return msgs, nil // Not enough cycles to compress
		}

		// Keep last keepCount cycles in full
		keepCycles := cycles[len(cycles)-keepCount:]
		compressCycles := cycles[:len(cycles)-keepCount]

		// Build compressed messages
		var result []ChatMessage

		// Add system prompt
		if systemMsg != nil {
			result = append(result, *systemMsg)
		}

		// Add compressed summary of old cycles
		if len(compressCycles) > 0 {
			summary := summarizeToolCycles(compressCycles)
			result = append(result, ChatMessage{
				Role:    RoleUser,
				Content: fmt.Sprintf("[HISTORY SUMMARY]\nPrevious %d tool interaction cycles:\n%s\n[/HISTORY SUMMARY]", len(compressCycles), summary),
			})
		}

		// Add full recent cycles
		for _, cycle := range keepCycles {
			result = append(result, cycle.Messages...)
		}

		return result, nil
	}
}

// ToolCycle represents a group of messages forming one tool interaction
type ToolCycle struct {
	Messages []ChatMessage
	Step     int
}

// identifyToolCycles groups messages into tool interaction cycles
func identifyToolCycles(msgs []ChatMessage) []ToolCycle {
	var cycles []ToolCycle
	var currentCycle []ChatMessage
	step := 0
	inToolCycle := false

	for _, msg := range msgs {
		if msg.Role == RoleSystem {
			continue // Skip system messages
		}

		// Start of a tool cycle: assistant message with tool calls
		if msg.Role == RoleAssistant && len(msg.ToolCalls) > 0 {
			// If we were in a previous cycle, close it
			if len(currentCycle) > 0 {
				cycles = append(cycles, ToolCycle{Messages: currentCycle, Step: step})
				currentCycle = nil
			}
			step++
			inToolCycle = true
			currentCycle = append(currentCycle, msg)
		} else if inToolCycle && msg.Role == RoleTool {
			// Tool results belong to current cycle
			currentCycle = append(currentCycle, msg)
		} else {
			// Other messages (user, assistant without tools)
			if len(currentCycle) > 0 {
				// Close current cycle
				cycles = append(cycles, ToolCycle{Messages: currentCycle, Step: step})
				currentCycle = nil
				inToolCycle = false
			}
			// Start a new cycle with this message
			currentCycle = []ChatMessage{msg}
		}
	}

	// Close final cycle if any
	if len(currentCycle) > 0 {
		cycles = append(cycles, ToolCycle{Messages: currentCycle, Step: step})
	}

	return cycles
}

// summarizeToolCycles creates compact summaries of tool cycles
func summarizeToolCycles(cycles []ToolCycle) string {
	var lines []string
	for _, cycle := range cycles {
		// Extract tool calls and summarize
		for _, msg := range cycle.Messages {
			if msg.Role == RoleAssistant && len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					// Find result for this tool call
					var result string
					for _, resultMsg := range cycle.Messages {
						if resultMsg.Role == RoleTool && resultMsg.Name == tc.ID {
							result = resultMsg.Content
							break
						}
					}
					summary := summarizeToolCall(tc, result)
					lines = append(lines, fmt.Sprintf("- Step %d: %s", cycle.Step, summary))
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

// summarizeToolCall creates a one-line summary of a tool call and its result
func summarizeToolCall(tc ToolCall, result string) string {
	switch tc.Name {
	case "read_file":
		if file, ok := tc.Args["file_path"].(string); ok {
			return fmt.Sprintf("read_file(%s) → %d bytes", file, len(result))
		}
	case "codebase_search":
		if query, ok := tc.Args["query"].(string); ok {
			return fmt.Sprintf("codebase_search('%s') → found results", truncateString(query, 40))
		}
	case "search_replace":
		if file, ok := tc.Args["file_path"].(string); ok {
			if strings.Contains(result, "ERROR") {
				return fmt.Sprintf("search_replace(%s) → FAILED", file)
			}
			return fmt.Sprintf("search_replace(%s) → success", file)
		}
	case "run_build":
		if strings.Contains(result, "exit_code: 0") || strings.Contains(result, "exit code 0") {
			return "run_build() → passed"
		}
		return "run_build() → failed"
	case "run_tests":
		if strings.Contains(result, "exit_code: 0") || strings.Contains(result, "PASS") {
			return "run_tests() → passed"
		}
		return "run_tests() → failed"
	case "grep":
		if strings.Contains(result, "Found") {
			return fmt.Sprintf("grep() → found matches")
		}
		return "grep() → no matches"
	case "think":
		return "think() → reasoning logged"
	default:
		if strings.Contains(result, "ERROR") {
			return fmt.Sprintf("%s() → error", tc.Name)
		}
		return fmt.Sprintf("%s() → completed", tc.Name)
	}
	return fmt.Sprintf("%s() → completed", tc.Name)
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
