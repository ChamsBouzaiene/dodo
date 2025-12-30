package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// listFilesImpl copies the implementation from tools/list_files.go
func listFilesImpl(fileSys FileSystem, repoRoot, path string, recursive bool, maxDepth, limit int, ignorePatterns []string) (string, error) {
	// Resolve and validate path
	dirPath := filepath.Join(repoRoot, path)
	dirPath = filepath.Clean(dirPath)

	// Security: Ensure path is within repo root
	if !strings.HasPrefix(dirPath, filepath.Clean(repoRoot)) {
		return "", fmt.Errorf("path %s is outside repository root", path)
	}

	// Compile ignore matcher
	var matcher *gitignore.GitIgnore
	if len(ignorePatterns) > 0 {
		matcher = gitignore.CompileIgnoreLines(ignorePatterns...)
	}

	files := make([]string, 0)
	truncated := false

	// Helper to check if path matches ignore patterns
	shouldIgnore := func(relPath string, isDir bool) bool {
		// Always ignore .git
		if strings.Contains(relPath, ".git") {
			return true
		}
		if matcher != nil {
			return matcher.MatchesPath(relPath)
		}
		return false
	}

	if recursive {
		err := fileSys.WalkDir(dirPath, func(walkPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Calculate relative path from repo root for ignore checking
			relPathFromRoot, err := filepath.Rel(repoRoot, walkPath)
			if err != nil {
				return nil
			}

			// Check ignore patterns
			if shouldIgnore(relPathFromRoot, d.IsDir()) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Check max depth
			if maxDepth >= 0 {
				relPathFromStart, err := filepath.Rel(dirPath, walkPath)
				if err == nil {
					depth := strings.Count(relPathFromStart, string(filepath.Separator))
					if depth > maxDepth {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
			}

			// Skip the root directory itself
			if walkPath == dirPath {
				return nil
			}

			// Add to results
			relPath, err := filepath.Rel(repoRoot, walkPath)
			if err == nil {
				files = append(files, relPath)
			}

			// Check limit
			if len(files) >= limit {
				truncated = true
				return filepath.SkipAll // Stop walking
			}

			return nil
		})
		if err != nil {
			return "", err
		}
	} else {
		// Non-recursive (original behavior)
		entries, err := fileSys.ReadDir(dirPath)
		if err != nil {
			return "", err
		}

		for _, entry := range entries {
			// Skip hidden files (starting with .) unless explicitly requested?
			// For now, keep original behavior of skipping dotfiles if no ignore patterns provided
			// But if ignore patterns provided, respect them instead.
			name := entry.Name()
			if len(ignorePatterns) == 0 && strings.HasPrefix(name, ".") {
				continue
			}

			relPath := name
			if path != "" {
				relPath = filepath.Join(path, name)
			}

			if shouldIgnore(relPath, entry.IsDir()) {
				continue
			}

			files = append(files, relPath)
			if len(files) >= limit {
				truncated = true
				break
			}
		}
	}

	result := map[string]interface{}{
		"path":      path,
		"files":     files,
		"recursive": recursive,
		"truncated": truncated,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewListFilesTool creates an engine.Tool that wraps the list_files functionality.
func NewListFilesTool(repoRoot string) engine.Tool {
	fs := NewOSFileSystem()
	return engine.Tool{
		Name:        "list_files",
		Description: "Lists files in the repository. Use this to discover which files exist before reading them. Supports recursive listing and ignoring files.",
		SchemaJSON: `{"type":"object","properties":{
			"path":{"type":"string","description":"Optional: subdirectory path relative to repository root (empty string for root)"},
			"recursive":{"type":"boolean","description":"If true, list files recursively. Default: false"},
			"max_depth":{"type":"integer","description":"Maximum depth for recursive listing. Default: -1 (unlimited)"},
			"limit":{"type":"integer","description":"Maximum number of files to return. Default: 1000"},
			"ignore_patterns":{"type":"array","items":{"type":"string"},"description":"List of gitignore-style patterns to ignore. Default: ['.git', 'node_modules']"}
		},"required":[]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path := ""
			if p, ok := args["path"].(string); ok {
				path = p
			}
			recursive := false
			if r, ok := args["recursive"].(bool); ok {
				recursive = r
			}
			maxDepth := -1
			if d, ok := args["max_depth"].(float64); ok {
				maxDepth = int(d)
			}
			limit := 1000
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			var ignorePatterns []string
			if patterns, ok := args["ignore_patterns"].([]interface{}); ok {
				for _, p := range patterns {
					if s, ok := p.(string); ok {
						ignorePatterns = append(ignorePatterns, s)
					}
				}
			}
			// Add defaults if not provided
			if len(ignorePatterns) == 0 {
				ignorePatterns = []string{".git", "node_modules"}
			}

			return listFilesImpl(fs, repoRoot, path, recursive, maxDepth, limit, ignorePatterns)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.1.0",
			Category: "filesystem",
			Tags:     []string{"read-only", "idempotent"},
		},
	}
}
