package reasoning

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// RespondParams defines the input for the respond tool.
type RespondParams struct {
	Summary      string   `json:"summary"`
	FilesChanged []string `json:"files_changed,omitempty"`
	NextSteps    []string `json:"next_steps,omitempty"`
}

// RespondResult is the output of the respond tool.
type RespondResult struct {
	Status       string   `json:"status"`
	Summary      string   `json:"summary"`
	FilesChanged []string `json:"files_changed,omitempty"`
	NextSteps    []string `json:"next_steps,omitempty"`
}

// respondImpl implements the respond tool functionality.
// This allows the agent to signal task completion with a summary.
func respondImpl(summary string, filesChanged []string, nextSteps []string) (string, error) {
	if summary == "" {
		return "", fmt.Errorf("summary cannot be empty")
	}

	// Log the response with distinctive formatting
	log.Println("📋 ============== AGENT RESPONSE ==============")
	log.Printf("📝 Summary: %s", summary)

	if len(filesChanged) > 0 {
		log.Println("📁 Files Changed:")
		for _, file := range filesChanged {
			log.Printf("   - %s", file)
		}
	}

	if len(nextSteps) > 0 {
		log.Println("➡️  Next Steps:")
		for i, step := range nextSteps {
			log.Printf("   %d. %s", i+1, step)
		}
	}

	log.Println("📋 ============================================")

	result := RespondResult{
		Status:       "complete",
		Summary:      summary,
		FilesChanged: filesChanged,
		NextSteps:    nextSteps,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// NewRespondTool creates an engine.Tool that wraps the respond functionality.
func NewRespondTool() engine.Tool {
	return engine.Tool{
		Name:        "respond",
		Description: `Signal task completion with a summary. Use this when you've finished the user's request. Provide a concise summary of what was done, which files were changed, and optional next steps. This marks the task as COMPLETE.`,
		SchemaJSON:  `{"type":"object","properties":{"summary":{"type":"string","description":"Concise summary of what was accomplished (2-4 sentences)"},"files_changed":{"type":"array","items":{"type":"string"},"description":"List of files that were created or modified"},"next_steps":{"type":"array","items":{"type":"string"},"description":"Optional: 1-3 suggested next steps for the user"}},"required":["summary"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			summary, ok := args["summary"].(string)
			if !ok {
				return "", fmt.Errorf("summary must be a string")
			}

			// Parse optional arrays
			var filesChanged []string
			if fc, ok := args["files_changed"].([]interface{}); ok {
				for _, f := range fc {
					if s, ok := f.(string); ok {
						filesChanged = append(filesChanged, s)
					}
				}
			}

			var nextSteps []string
			if ns, ok := args["next_steps"].([]interface{}); ok {
				for _, n := range ns {
					if s, ok := n.(string); ok {
						nextSteps = append(nextSteps, s)
					}
				}
			}

			return respondImpl(summary, filesChanged, nextSteps)
		},
		Retryable: true, // Idempotent - can be called multiple times
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "meta",
			Tags:     []string{"completion", "summary", "communication"},
		},
	}
}
