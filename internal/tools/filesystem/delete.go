package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// DeleteFileParams defines the input for the delete_file tool.
type DeleteFileParams struct {
	Path string `json:"path"`
}

// DeleteFileResult is the output of the delete_file tool.
type DeleteFileResult struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// deleteFileImpl implements the delete_file tool functionality.
// IMPORTANT: This is a destructive operation - use with caution.
func deleteFileImpl(fs FileSystem, repoRoot, relPath string) (string, error) {
	// Resolve and validate path
	absPath := filepath.Join(repoRoot, relPath)
	absPath = filepath.Clean(absPath)

	// Security: Ensure path is within repo root
	if !strings.HasPrefix(absPath, filepath.Clean(repoRoot)) {
		return "", fmt.Errorf("path %s is outside repository root", relPath)
	}

	// Check if file exists
	info, err := fs.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			result := DeleteFileResult{
				Path:    relPath,
				Success: true,
				Message: "File does not exist (already deleted)",
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), nil
		}
		return "", fmt.Errorf("failed to check file: %w", err)
	}

	// Prevent deleting directories
	if info.IsDir() {
		return "", fmt.Errorf("cannot delete directory %s (use delete_file only for files)", relPath)
	}

	// Delete the file
	if err := fs.Remove(absPath); err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	result := DeleteFileResult{
		Path:    relPath,
		Success: true,
		Message: "File deleted successfully",
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// NewDeleteFileTool creates an engine.Tool that wraps the delete_file functionality.
func NewDeleteFileTool(repoRoot string) engine.Tool {
	fs := NewOSFileSystem()
	return engine.Tool{
		Name:        "delete_file",
		Description: "Deletes a file from the repository. Use with caution - this is a destructive operation. Cannot delete directories. Use this to remove conflicting files or clean up temporary files.",
		SchemaJSON:  `{"type":"object","properties":{"path":{"type":"string","description":"Path to the file to delete, relative to the repository root"}},"required":["path"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path, ok := args["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			if path == "" {
				return "", fmt.Errorf("path cannot be empty")
			}
			return deleteFileImpl(fs, repoRoot, path)
		},
		Retryable: false, // Deleting files is NOT idempotent
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "filesystem",
			Tags:     []string{"delete", "destructive", "side-effect"},
		},
	}
}
