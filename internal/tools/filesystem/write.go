package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// writeFileImpl copies the implementation from tools/write.go
func writeFileImpl(fs FileSystem, repoRoot, path, content string) (string, error) {
	// Resolve and validate path
	filePath := filepath.Join(repoRoot, path)
	filePath = filepath.Clean(filePath)

	// Security: Ensure path is within repo root
	if !strings.HasPrefix(filePath, filepath.Clean(repoRoot)) {
		return "", fmt.Errorf("path %s is outside repository root", path)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := fs.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := fs.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	result := map[string]interface{}{
		"path":    path,
		"success": true,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewWriteFileTool creates an engine.Tool that wraps the write_file functionality.
func NewWriteFileTool(repoRoot string) engine.Tool {
	fs := NewOSFileSystem()
	return engine.Tool{
		Name:        "write_file",
		Description: "Writes content to a file. Creates the file if it doesn't exist, overwrites if it does.",
		SchemaJSON:  `{"type":"object","properties":{"path":{"type":"string","description":"Path to the file relative to the repository root"},"content":{"type":"string","description":"Content to write to the file"}},"required":["path","content"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path, ok := args["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			content, ok := args["content"].(string)
			if !ok {
				return "", fmt.Errorf("content must be a string")
			}
			return writeFileImpl(fs, repoRoot, path, content)
		},
		Retryable: false, // Writing files is not idempotent
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "filesystem",
			Tags:     []string{"write", "side-effect"},
		},
	}
}
