package search

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/tools/execution"
)

// grepImpl uses ripgrep to search for patterns
func grepImpl(ctx context.Context, runner execution.Runner, repoRoot, pattern, path, globs string, caseInsensitive bool) (string, error) {
	// Build rg command
	// --json: output in JSON format
	// --line-number: show line numbers (implied by json but good to be explicit)
	// --with-filename: show file names (implied by json)
	// --no-heading: (implied by json)
	args := []string{"--json"}

	if caseInsensitive {
		args = append(args, "-i")
	}

	// Handle globs
	if globs != "" {
		parts := strings.Split(globs, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				args = append(args, "-g", trimmed)
			}
		}
	}

	// Pattern
	args = append(args, "-e", pattern)

	// Path (default to current directory if empty)
	if path != "" {
		args = append(args, path)
	} else {
		args = append(args, ".")
	}

	// Run command
	// 10 second timeout for search seems reasonable
	res, err := runner.RunCmd(ctx, repoRoot, "rg", args, 10*time.Second)
	if err != nil {
		// Check if it's just "no matches found" (exit code 1)
		if res.Code == 1 {
			return `{"pattern": "` + pattern + `", "results": [], "count": 0}`, nil
		}
		return "", fmt.Errorf("grep failed: %v, stderr: %s", err, res.Stderr)
	}

	// Parse JSON output
	// rg --json outputs one JSON object per line
	type RgMatch struct {
		Type string `json:"type"`
		Data struct {
			Path struct {
				Text string `json:"text"`
			} `json:"path"`
			Lines struct {
				Text string `json:"text"`
			} `json:"lines"`
			LineNumber int `json:"line_number"`
			Submatches []struct {
				Match struct {
					Text string `json:"text"`
				} `json:"match"`
				Start int `json:"start"`
				End   int `json:"end"`
			} `json:"submatches"`
		} `json:"data"`
	}

	type GrepResult struct {
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Content string `json:"content"`
	}

	results := make([]GrepResult, 0)
	lines := strings.Split(res.Stdout, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var rgMsg RgMatch
		if err := json.Unmarshal([]byte(line), &rgMsg); err != nil {
			// Skip non-match lines or parse errors
			continue
		}

		if rgMsg.Type == "match" {
			results = append(results, GrepResult{
				Path:    rgMsg.Data.Path.Text,
				Line:    rgMsg.Data.LineNumber,
				Content: strings.TrimSpace(rgMsg.Data.Lines.Text),
			})
		}
	}

	// Limit results to prevent context overflow
	maxResults := 100
	truncated := false
	if len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}

	response := map[string]interface{}{
		"pattern":   pattern,
		"results":   results,
		"count":     len(results),
		"truncated": truncated,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(responseJSON), nil
}

// NewGrepTool creates an engine.Tool that wraps the grep functionality.
func NewGrepTool(repoRoot string) engine.Tool {
	runner := execution.NewSandboxRunner()
	return engine.Tool{
		Name:        "grep",
		Description: "Fast, regex-based code search using ripgrep. Use this to find code patterns, function definitions, or references. Supports case-insensitive search and glob patterns.",
		SchemaJSON:  `{"type":"object","properties":{"pattern":{"type":"string","description":"Regex pattern to search for"},"path":{"type":"string","description":"Optional: specific file or directory path"},"globs":{"type":"string","description":"Optional: comma-separated file patterns"},"case_insensitive":{"type":"boolean","description":"Optional: case-insensitive search"}},"required":["pattern"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			pattern, ok := args["pattern"].(string)
			if !ok {
				return "", fmt.Errorf("pattern must be a string")
			}
			path := ""
			if p, ok := args["path"].(string); ok {
				path = p
			}
			globs := ""
			if g, ok := args["globs"].(string); ok {
				globs = g
			}
			caseInsensitive := false
			if ci, ok := args["case_insensitive"].(bool); ok {
				caseInsensitive = ci
			}
			return grepImpl(ctx, runner, repoRoot, pattern, path, globs, caseInsensitive)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "2.0.0",
			Category: "search",
			Tags:     []string{"read-only", "idempotent"},
		},
	}
}
