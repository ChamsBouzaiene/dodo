package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/workspace"
)

// codebaseSearchImpl copies the implementation from tools/codebase_search.go
func codebaseSearchImpl(ctx context.Context, retrieval indexer.Retrieval, query, globs string, k int) (string, error) {
	// Default k
	if k <= 0 {
		k = 10
	}

	// Parse globs
	var globList []string
	if globs != "" {
		parts := strings.Split(globs, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				globList = append(globList, trimmed)
			}
		}
	}

	// If no globs provided, default by project type
	if len(globList) == 0 {
		switch workspace.DetectProjectType(".") {
		case workspace.ProjectTypeGo:
			globList = []string{"*.go"}
		case workspace.ProjectTypeNode:
			globList = []string{"*.ts", "*.tsx", "*.js", "*.jsx"}
		case workspace.ProjectTypePython:
			globList = []string{"*.py"}
		case workspace.ProjectTypeRust:
			globList = []string{"*.rs"}
		}
	}

	// Perform search
	spans, err := retrieval.Search(ctx, query, globList, k)
	if err != nil {
		return "", err
	}

	// Convert spans to results with read_span recommendations
	// IMPORTANT: Fields are ordered by importance - LLM reads top-to-bottom
	type CodebaseSearchResult struct {
		Priority   string  `json:"priority"`    // "high", "medium", or "low" - FIRST for visibility
		NextAction string  `json:"next_action"` // Human-readable instruction - SECOND
		Command    string  `json:"command"`     // Copy-paste ready command - THIRD
		Path       string  `json:"path"`        // File path
		Lines      string  `json:"lines"`       // Simplified line range
		Snippet    string  `json:"snippet"`     // Code preview
		Reason     string  `json:"reason"`      // Why this is relevant
		Score      float64 `json:"score"`       // Relevance score (for debugging)
	}
	results := make([]CodebaseSearchResult, len(spans))
	for i, span := range spans {
		// Calculate context window (add 10 lines buffer on each side)
		contextStart := span.Start - 10
		if contextStart < 1 {
			contextStart = 1
		}
		contextEnd := span.End + 10

		// Calculate priority: higher score + smaller span = higher priority
		spanSize := contextEnd - contextStart
		priority := "high"
		if span.Score < 0.7 {
			priority = "medium"
		}
		if span.Score < 0.5 || spanSize > 200 {
			priority = "low"
		}

		// Generate actionable instruction
		nextAction := fmt.Sprintf("Read lines %d-%d from %s", contextStart, contextEnd, span.Path)
		command := fmt.Sprintf("read_span({\"path\": \"%s\", \"start\": %d, \"end\": %d})",
			span.Path, contextStart, contextEnd)
		linesStr := fmt.Sprintf("%d-%d", contextStart, contextEnd)

		results[i] = CodebaseSearchResult{
			Priority:   priority,
			NextAction: nextAction,
			Command:    command,
			Path:       span.Path,
			Lines:      linesStr,
			Snippet:    span.Snippet,
			Reason:     span.Reason,
			Score:      span.Score,
		}
	}

	// Build workflow reminder
	workflowReminder := `
╔════════════════════════════════════════════════════════════════╗
║                    RECOMMENDED WORKFLOW                        ║
╠════════════════════════════════════════════════════════════════╣
║ 1. Review the 'priority' field (high/medium/low)              ║
║ 2. Start with HIGH priority results                           ║
║ 3. Copy the 'command' field and call it directly              ║
║ 4. Use read_span for ALL results - it's faster than read_file ║
║                                                                ║
║ ⚠️  DO NOT call read_file on large files                      ║
║ ✅  DO call multiple read_span in parallel                    ║
╚════════════════════════════════════════════════════════════════╝`

	response := map[string]interface{}{
		"results":  results,
		"query":    query,
		"count":    len(results),
		"workflow": workflowReminder,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(responseJSON), nil
}

// NewCodebaseSearchTool creates an engine.Tool that wraps the codebase_search functionality.
func NewCodebaseSearchTool(retrieval indexer.Retrieval) engine.Tool {
	return engine.Tool{
		Name:        "codebase_search",
		Description: "Performs semantic search across the codebase to find relevant code spans. Use this to find code related to your query by meaning, not just exact text matches.",
		SchemaJSON:  `{"type":"object","properties":{"query":{"type":"string","description":"Natural language query describing what code you're looking for"},"globs":{"type":"string","description":"Optional comma-separated file patterns"},"k":{"type":"integer","description":"Maximum number of results (default: 10)"}},"required":["query"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			query, ok := args["query"].(string)
			if !ok {
				return "", fmt.Errorf("query must be a string")
			}
			globs := ""
			if g, ok := args["globs"].(string); ok {
				globs = g
			}
			k := 10
			if kVal, ok := args["k"].(float64); ok {
				k = int(kVal)
			}
			return codebaseSearchImpl(ctx, retrieval, query, globs, k)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "search",
			Tags:     []string{"read-only", "idempotent", "semantic"},
		},
	}
}
