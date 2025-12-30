package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
)

// readSpanImpl copies the implementation from tools/read_span.go
func readSpanImpl(ctx context.Context, retrieval indexer.Retrieval, path string, start, end int) (string, error) {
	// Auto-correct invalid line numbers
	if start < 1 {
		start = 1
	}
	if end < 1 {
		end = 1
	}
	if end < start {
		start, end = end, start
	}

	// Read the span
	source, err := retrieval.ReadSpan(ctx, path, start, end)
	if err != nil {
		return "", err
	}

	result := map[string]interface{}{
		"path":   path,
		"start":  start,
		"end":    end,
		"source": source,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewReadSpanTool creates an engine.Tool that wraps the read_span functionality.
func NewReadSpanTool(retrieval indexer.Retrieval) engine.Tool {
	return engine.Tool{
		Name:        "read_span",
		Description: "Reads a specific line range (span) from a file in the repository. Use this after codebase_search to read the full context of a code location.",
		SchemaJSON:  `{"type":"object","properties":{"path":{"type":"string","description":"File path relative to repository root"},"start":{"type":"integer","description":"Start line number (1-indexed, inclusive)"},"end":{"type":"integer","description":"End line number (1-indexed, inclusive)"}},"required":["path","start","end"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path, ok := args["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			start := 0
			if s, ok := args["start"].(float64); ok {
				start = int(s)
			}
			end := 0
			if e, ok := args["end"].(float64); ok {
				end = int(e)
			}
			return readSpanImpl(ctx, retrieval, path, start, end)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "search",
			Tags:     []string{"read-only", "idempotent"},
		},
	}
}
