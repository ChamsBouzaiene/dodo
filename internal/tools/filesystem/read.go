package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// readFileImpl copies the implementation from tools/read_file.go
func readFileImpl(fs FileSystem, repoRoot, path string) (string, error) {
	// Resolve and validate path
	filePath := filepath.Join(repoRoot, path)
	filePath = filepath.Clean(filePath)

	// Security: Ensure path is within repo root
	if !strings.HasPrefix(filePath, filepath.Clean(repoRoot)) {
		return "", fmt.Errorf("path %s is outside repository root", path)
	}

	contentBytes, err := fs.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(contentBytes)
	lineCount := strings.Count(content, "\n") + 1

	// Tier 1: Small files (<200 lines) - return full content
	if lineCount < 200 {
		result := map[string]interface{}{
			"path":         path,
			"content":      content,
			"line_count":   lineCount,
			"content_type": "full",
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(resultJSON), nil
	}

	// Tier 2: Medium files (200-400 lines) - return full content with warning
	if lineCount < 400 {
		warningHeader := fmt.Sprintf("⚠️  LARGE FILE WARNING: This file has %d lines.\n"+
			"For editing, consider using 'read_span' to focus on specific sections for better performance.\n"+
			"Example: read_span({\"path\": \"%s\", \"start\": 50, \"end\": 150})\n\n"+
			"=== FULL FILE CONTENT BELOW ===\n\n",
			lineCount, path)
		result := map[string]interface{}{
			"path":         path,
			"content":      warningHeader + content,
			"line_count":   lineCount,
			"content_type": "full",
		}
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(resultJSON), nil
	}

	// Tier 3: Large files (>400 lines) - return OUTLINE only
	outline := generateOutline(content, path, lineCount)
	result := map[string]interface{}{
		"path":         path,
		"content":      outline,
		"line_count":   lineCount,
		"content_type": "outline",
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(resultJSON), nil
}

// generateOutline copies the outline generation logic from tools/read_file.go
func generateOutline(content, path string, lineCount int) string {
	ext := filepath.Ext(path)
	var outline strings.Builder
	outline.WriteString("╔═══════════════════════════════════════════════════════════════╗\n")
	outline.WriteString("║  📋 OUTLINE MODE - This is NOT the full file content         ║\n")
	outline.WriteString("║  This file has " + fmt.Sprintf("%4d", lineCount) + " lines (too large for full context)      ║\n")
	outline.WriteString("║                                                               ║\n")
	outline.WriteString("║  ⚠️  DO NOT pass this outline to 'propose_diff'               ║\n")
	outline.WriteString("║                                                               ║\n")
	outline.WriteString("║  EFFICIENT READING STRATEGY:                                  ║\n")
	outline.WriteString("║  1. Review the outline below to find your target function    ║\n")
	outline.WriteString("║  2. Use 'read_span' to read that specific section            ║\n")
	outline.WriteString("║     Example: read_span({\"path\": \"" + path + "\", \"start\": 50, \"end\": 150})  ║\n")
	outline.WriteString("║  3. Pass the read_span result to 'propose_diff'              ║\n")
	outline.WriteString("║                                                               ║\n")
	outline.WriteString("║  💡 TIP: Use 'codebase_search' first to find relevant code,  ║\n")
	outline.WriteString("║  then use the 'read_span_hint' from results for exact lines  ║\n")
	outline.WriteString("╚═══════════════════════════════════════════════════════════════╝\n\n")

	switch ext {
	case ".go":
		outline.WriteString(generateGoOutline(content))
	case ".py":
		outline.WriteString(generatePythonOutline(content))
	case ".ts", ".tsx", ".js", ".jsx":
		outline.WriteString(generateJSOutline(content))
	default:
		outline.WriteString(generateGenericOutline(content, lineCount))
	}

	return outline.String()
}

func generateGoOutline(content string) string {
	var outline strings.Builder
	outline.WriteString("=== GO FILE STRUCTURE ===\n\n")
	lines := strings.Split(content, "\n")
	inMultilineComment := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "/*") {
			inMultilineComment = true
		}
		if strings.HasSuffix(trimmed, "*/") {
			inMultilineComment = false
			continue
		}
		if inMultilineComment || strings.HasPrefix(trimmed, "//") {
			continue
		}

		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "func ") ||
			strings.HasPrefix(trimmed, "const ") ||
			strings.HasPrefix(trimmed, "var ") {
			outline.WriteString(fmt.Sprintf("Line %4d: %s\n", i+1, trimmed))
		}
	}

	outline.WriteString("\nUse 'read_span' with the line numbers above to read specific functions.\n")
	return outline.String()
}

func generatePythonOutline(content string) string {
	var outline strings.Builder
	outline.WriteString("=== PYTHON FILE STRUCTURE ===\n\n")
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "from ") ||
			strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "@") {
			outline.WriteString(fmt.Sprintf("Line %4d: %s\n", i+1, trimmed))
		}
	}
	outline.WriteString("\nUse 'read_span' with the line numbers above to read specific functions.\n")
	return outline.String()
}

func generateJSOutline(content string) string {
	var outline strings.Builder
	outline.WriteString("=== JAVASCRIPT/TYPESCRIPT FILE STRUCTURE ===\n\n")
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "export ") ||
			strings.HasPrefix(trimmed, "class ") ||
			(strings.HasPrefix(trimmed, "function ") || (strings.HasPrefix(trimmed, "const ") && strings.Contains(trimmed, "=>"))) ||
			strings.HasPrefix(trimmed, "interface ") ||
			strings.HasPrefix(trimmed, "type ") {
			outline.WriteString(fmt.Sprintf("Line %4d: %s\n", i+1, trimmed))
		}
	}
	outline.WriteString("\nUse 'read_span' with the line numbers above to read specific functions.\n")
	return outline.String()
}

func generateGenericOutline(content string, lineCount int) string {
	lines := strings.Split(content, "\n")
	var outline strings.Builder
	outline.WriteString("=== FILE STRUCTURE (Generic) ===\n\n")
	outline.WriteString("=== FIRST 30 LINES ===\n")
	for i := 0; i < 30 && i < len(lines); i++ {
		outline.WriteString(fmt.Sprintf("Line %4d: %s\n", i+1, lines[i]))
	}
	if lineCount > 60 {
		outline.WriteString(fmt.Sprintf("\n... %d lines omitted ...\n\n", lineCount-60))
		outline.WriteString("=== LAST 30 LINES ===\n")
		start := len(lines) - 30
		if start < 0 {
			start = 0
		}
		for i := start; i < len(lines); i++ {
			outline.WriteString(fmt.Sprintf("Line %4d: %s\n", i+1, lines[i]))
		}
	}
	outline.WriteString("\nUse 'read_span' to read specific line ranges from this file.\n")
	return outline.String()
}

// NewReadFileTool creates an engine.Tool that wraps the read_file functionality.
func NewReadFileTool(repoRoot string) engine.Tool {
	fs := NewOSFileSystem()
	return engine.Tool{
		Name:        "read_file",
		Description: "Reads the content of a file from the repository. Provide the file path relative to the repo root.",
		SchemaJSON:  `{"type":"object","properties":{"path":{"type":"string","description":"Path to the file relative to the repository root"}},"required":["path"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			path, ok := args["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}
			return readFileImpl(fs, repoRoot, path)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "filesystem",
			Tags:     []string{"read-only", "idempotent"},
		},
	}
}
