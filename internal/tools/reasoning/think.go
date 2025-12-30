package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// thinkImpl implements the think tool functionality.
// This allows the agent to log its reasoning and thought process.
func thinkImpl(reasoning string) (string, error) {
	// Log the reasoning with a distinctive emoji for visibility
	log.Printf("🧠 Agent reasoning: %s", reasoning)

	result := map[string]interface{}{
		"status": "noted",
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewThinkTool creates an engine.Tool that wraps the think functionality.
// This tool allows the agent to record its reasoning and thought process,
// providing transparency into the agent's decision-making.
func NewThinkTool() engine.Tool {
	return engine.Tool{
		Name: "think",
		Description: `Record your reasoning and thought process. Use this to make your thinking transparent.

When to use:
- After understanding the task, explain your high-level approach
- Before making changes, explain what you're about to do and why
- When you discover something important, note it
- When choosing between options, explain your decision

Example:
think({"reasoning": "I understand the task: add a title 'Snake Game' below the score. I'll modify internal/adapter/terminal/terminal.go by adding a new PaintTitle() method and calling it from PaintScore(). This is a simple addition with no signature changes needed."})

Your reasoning will be logged and visible to the user, helping them understand your approach.`,
		SchemaJSON: `{"type":"object","properties":{"reasoning":{"type":"string","description":"Your reasoning, thought process, or plan. Be specific about what you understand, what you'll do, and why. Include file names and function names when relevant."},"reason":{"type":"string","description":"Alias for 'reasoning' (deprecated, use 'reasoning' instead)"}},"required":[]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			// Accept both 'reasoning' and 'reason' for backwards compatibility
			var reasoning string
			if r, ok := args["reasoning"].(string); ok {
				reasoning = r
			} else if r, ok := args["reason"].(string); ok {
				reasoning = r
			} else {
				return "", fmt.Errorf("either 'reasoning' or 'reason' must be provided as a string")
			}

			if reasoning == "" {
				return "", fmt.Errorf("reasoning cannot be empty")
			}

			return thinkImpl(reasoning)
		},
		Retryable: true, // Thinking is idempotent
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "meta",
			Tags:     []string{"reasoning", "idempotent", "logging"},
		},
	}
}
