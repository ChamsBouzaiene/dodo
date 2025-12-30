package editing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// writeImpl copies the implementation from tools/write.go
func writeImpl(repoRoot, path, content string) (string, error) {
	// Construct full path
	fullPath := filepath.Join(repoRoot, path)

	// Check if it's a text file
	if !isTextFile(fullPath) {
		result := map[string]interface{}{
			"path":   path,
			"status": "failed",
			"error":  "File type not allowed. write only works on text files.",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// Check if file exists to determine status
	fileExists := false
	if info, err := os.Stat(fullPath); err == nil {
		fileExists = true
		// Optimization: Check if content is identical
		if !info.IsDir() {
			existingContent, err := os.ReadFile(fullPath)
			if err == nil && string(existingContent) == content {
				result := map[string]interface{}{
					"path":   path,
					"status": "skipped",
					"lines":  strings.Count(content, "\n") + 1,
				}
				resultJSON, _ := json.Marshal(result)
				return string(resultJSON), nil
			}
		}
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result := map[string]interface{}{
			"path":   path,
			"status": "failed",
			"error":  fmt.Sprintf("Failed to create directory: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// Count lines
	lines := 1
	for _, c := range content {
		if c == '\n' {
			lines++
		}
	}

	// Write file atomically
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		result := map[string]interface{}{
			"path":   path,
			"status": "failed",
			"error":  fmt.Sprintf("Failed to write file: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// Determine status
	status := "created"
	if fileExists {
		status = "overwritten"
	}

	result := map[string]interface{}{
		"path":   path,
		"status": status,
		"lines":  lines,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// NewWriteTool creates an engine.Tool that wraps the write functionality.
func NewWriteTool(repoRoot string) engine.Tool {
	return engine.Tool{
		Name:        "write",
		Description: "Writes complete file contents to disk. Creates new files or OVERWRITES existing ones. Use search_replace for editing existing files.",
		SchemaJSON:  `{"type":"object","properties":{"path":{"type":"string","description":"The path to the file relative to the repo root"},"content":{"type":"string","description":"The content to write to the file"}},"required":["path","content"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path, ok := args["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			content, ok := args["content"].(string)
			if !ok {
				return "", fmt.Errorf("content must be a string")
			}
			return writeImpl(repoRoot, path, content)
		},
		Retryable: false, // Writing files is not idempotent
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "editing",
			Tags:     []string{"write", "side-effect"},
		},
	}
}
