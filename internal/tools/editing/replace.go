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

// searchReplaceImpl copies the implementation from tools/search_replace.go
func searchReplaceImpl(repoRoot, filePath, oldString, newString string, replaceAll bool) (string, error) {
	// Construct full path
	fullPath := filepath.Join(repoRoot, filePath)

	// 1. Check file type
	if !isTextFile(fullPath) {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  "File type not allowed. search_replace only works on text files.",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 2. Read entire file into memory
	contentBytes, err := os.ReadFile(fullPath)
	if err != nil {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  fmt.Sprintf("Failed to read file: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	content := string(contentBytes)

	// 3. Check for generated file
	if isGen, marker := isGeneratedFile(content); isGen {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  fmt.Sprintf("File appears to be generated (found: %q). Edit the generator instead.", marker),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 4. Check edit size
	if safe, warning := isSafeEditSize(oldString); !safe {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  warning,
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	} else if warning != "" {
		// Warning but proceed
	}

	// 5. Count occurrences of old_string
	count := strings.Count(content, oldString)

	// 6. Validate occurrence count
	if count == 0 {
		// Check for whitespace mismatch
		normalizedContent := strings.Join(strings.Fields(content), " ")
		normalizedOld := strings.Join(strings.Fields(oldString), " ")
		hint := ""
		if strings.Contains(normalizedContent, normalizedOld) {
			hint = "\n  - Whitespace mismatch detected. The text exists but with different whitespace/indentation."
		}

		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error": fmt.Sprintf("old_string not found in file. Common causes:\n"+
				"  - Indentation mismatch (file uses %s)\n"+
				"  - Whitespace differences\n"+
				"  - old_string needs more context for uniqueness\n"+
				"Solution: Read file again and copy EXACT text%s", detectIndentation(content), hint),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	if count > 1 && !replaceAll {
		// Find line numbers of occurrences
		var lineNums []int
		lines := strings.Split(content, "\n")
		oldLines := strings.Split(oldString, "\n")
		firstLineOld := strings.TrimSpace(oldLines[0])

		for i, line := range lines {
			if strings.Contains(line, firstLineOld) {
				// Simple heuristic: if the first line matches, it's a candidate
				// For exact multi-line matching, we'd need more complex logic,
				// but this is good enough for a hint
				lineNums = append(lineNums, i+1)
			}
		}

		lineNumStr := ""
		if len(lineNums) > 0 {
			// Limit to first 5
			if len(lineNums) > 5 {
				lineNums = lineNums[:5]
			}
			lineNumStr = fmt.Sprintf(" (lines: %v...)", lineNums)
		}

		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error": fmt.Sprintf("old_string appears %d times in the file%s. Solutions:\n"+
				"  1. Include more context in old_string to make it unique (e.g., include function signature)\n"+
				"  2. Use replace_all=true to replace all %d occurrences", count, lineNumStr, count),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 7. Check if old_string and new_string are identical
	if oldString == newString {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  "old_string and new_string are identical. No changes to make.",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 8. Perform the replacement
	var newContent string
	var replacements int

	if replaceAll {
		newContent = strings.ReplaceAll(content, oldString, newString)
		replacements = count
	} else {
		// Replace only the first (and only) occurrence
		newContent = strings.Replace(content, oldString, newString, 1)
		replacements = 1
	}

	// 9. Write the file back atomically
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		result := map[string]interface{}{
			"path":   filePath,
			"status": "failed",
			"error":  fmt.Sprintf("Failed to write file: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// Success!
	result := map[string]interface{}{
		"path":         filePath,
		"status":       "success",
		"replacements": replacements,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}

// Helper functions copied from tools/search_replace.go
func isTextFile(path string) bool {
	textExts := []string{
		".go", ".py", ".js", ".ts", ".jsx", ".tsx",
		".java", ".c", ".cpp", ".h", ".hpp",
		".rs", ".rb", ".php", ".html", ".css", ".scss",
		".md", ".txt", ".json", ".yaml", ".yml", ".toml",
		".sh", ".bash", ".zsh", ".sql", ".xml",
	}
	ext := filepath.Ext(path)
	for _, allowed := range textExts {
		if ext == allowed {
			return true
		}
	}
	return false
}

func isGeneratedFile(content string) (bool, string) {
	preview := content
	if len(content) > 500 {
		preview = content[:500]
	}

	markers := []string{
		"Code generated",
		"DO NOT EDIT",
		"Auto-generated",
		"automatically generated",
		"This file is generated",
	}

	for _, marker := range markers {
		if strings.Contains(preview, marker) {
			return true, marker
		}
	}
	return false, ""
}

func isSafeEditSize(oldString string) (bool, string) {
	lines := strings.Count(oldString, "\n")

	if lines > 500 {
		return false, fmt.Sprintf("old_string is %d lines (max recommended: 500). Consider breaking into smaller changes.", lines)
	}

	if lines > 200 {
		return true, fmt.Sprintf("Warning: old_string is %d lines. Smaller changes are safer.", lines)
	}

	return true, ""
}

func detectIndentation(content string) string {
	if strings.Contains(content, "\t") {
		return "tabs"
	}
	if strings.Contains(content, "    ") {
		return "4 spaces"
	}
	if strings.Contains(content, "  ") {
		return "2 spaces"
	}
	return "unknown"
}

// NewSearchReplaceTool creates an engine.Tool that wraps the search_replace functionality.
func NewSearchReplaceTool(repoRoot string) engine.Tool {
	return engine.Tool{
		Name:        "search_replace",
		Description: "Performs exact string search and replace in a file. This is the PRIMARY tool for editing files. ALWAYS read the file first with read_file to see exact content.",
		SchemaJSON:  `{"type":"object","properties":{"file_path":{"type":"string","description":"File path relative to the repository root"},"old_string":{"type":"string","description":"Exact string to find and replace"},"new_string":{"type":"string","description":"Replacement string"},"replace_all":{"type":"boolean","description":"If true, replace all occurrences"}},"required":["file_path","old_string","new_string"]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			filePath, ok := args["file_path"].(string)
			if !ok {
				return "", fmt.Errorf("file_path must be a string")
			}
			oldString, ok := args["old_string"].(string)
			if !ok {
				return "", fmt.Errorf("old_string must be a string")
			}
			newString, ok := args["new_string"].(string)
			if !ok {
				return "", fmt.Errorf("new_string must be a string")
			}
			replaceAll := false
			if ra, ok := args["replace_all"].(bool); ok {
				replaceAll = ra
			}
			return searchReplaceImpl(repoRoot, filePath, oldString, newString, replaceAll)
		},
		Retryable: false, // Editing files is not idempotent
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "editing",
			Tags:     []string{"write", "side-effect"},
		},
	}
}
