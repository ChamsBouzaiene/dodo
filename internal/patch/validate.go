package patch

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ProposedDiff represents a proposed code change with rationale.
// This is a copy of agent.ProposedDiff to avoid import cycles.
type ProposedDiff struct {
	Target    string `json:"target"`    // File path (repo-relative path)
	Unified   string `json:"unified"`   // Unified diff text
	Rationale string `json:"rationale"` // Explanation of why this change is needed
}

// DiffBudget defines limits for a proposed diff.
type DiffBudget struct {
	MaxFiles        int      // Maximum number of files that can be changed
	MaxTotalLines   int      // Maximum total lines changed (additions + deletions)
	MaxLinesPerFile int      // Maximum lines changed per file (soft cap)
	AllowedPrefixes []string // Allowed path prefixes (e.g., ["src/", "internal/", "cmd/"])
}

// ForbiddenPaths are paths that should never be modified.
var ForbiddenPaths = []string{
	".env",
	".env.*",
	"config/secrets*",
	".git",
	".github",
	".idea",
	".vscode",
	".gitignore",
	".gitattributes",
	"go.mod",
	"go.sum",
	"package-lock.json",
	"yarn.lock",
	"node_modules",
	"dist",
	"build",
	"venv",
	".venv",
	".DS_Store",
}

// ValidateProposedDiff validates a ProposedDiff against a DiffBudget.
// It parses the unified diff, counts changed lines, and checks path constraints.
func ValidateProposedDiff(diff ProposedDiff, budget DiffBudget) error {
	if diff.Unified == "" {
		return fmt.Errorf("proposed diff has empty unified diff")
	}

	if diff.Target == "" {
		return fmt.Errorf("proposed diff has empty target path")
	}

	// 1) Parse unified diff into hunks and extract file paths
	files, totalLines, linesPerFile, err := parseUnifiedDiff(diff.Unified)
	if err != nil {
		return fmt.Errorf("failed to parse unified diff: %w", err)
	}

	// 2) Count files touched
	numFiles := len(files)
	if numFiles == 0 {
		// If parsing didn't find files, use the target from ProposedDiff
		numFiles = 1
		files = []string{diff.Target}
		// If we only have target, estimate lines from totalLines
		if len(linesPerFile) == 0 && totalLines > 0 {
			linesPerFile = map[string]int{diff.Target: totalLines}
		}
	}

	// 3) Validate file count
	if budget.MaxFiles > 0 && numFiles > budget.MaxFiles {
		return fmt.Errorf("patch touches %d files, max is %d", numFiles, budget.MaxFiles)
	}

	// 4) Validate total lines changed
	if budget.MaxTotalLines > 0 && totalLines > budget.MaxTotalLines {
		return fmt.Errorf("patch changes %d lines, max is %d", totalLines, budget.MaxTotalLines)
	}

	// 5) Validate lines per file
	if budget.MaxLinesPerFile > 0 {
		for file, lines := range linesPerFile {
			if lines > budget.MaxLinesPerFile {
				return fmt.Errorf("patch changes %d lines in file %s, max is %d per file", lines, file, budget.MaxLinesPerFile)
			}
		}
	}

	// 6) Validate file paths
	for _, file := range files {
		// Check for forbidden paths
		if err := isForbiddenPath(file); err != nil {
			return err
		}

		// Check allowed prefixes if specified
		if len(budget.AllowedPrefixes) > 0 {
			if !hasAllowedPrefix(file, budget.AllowedPrefixes) {
				return fmt.Errorf("path %s does not match any allowed prefix: %v", file, budget.AllowedPrefixes)
			}
		}
	}

	return nil
}

// parseUnifiedDiff parses a unified diff string and returns:
// - list of file paths touched
// - total number of lines changed (additions + deletions)
// - map of file path to lines changed in that file
func parseUnifiedDiff(diff string) ([]string, int, map[string]int, error) {
	lines := strings.Split(diff, "\n")
	var files []string
	totalChanged := 0
	linesPerFile := make(map[string]int)

	// Regex to match file paths in --- and +++ lines
	filePathRegex := regexp.MustCompile(`^(---|\+\+\+)\s+(?:a/|b/)?(.+)$`)
	// Regex to match hunk headers: @@ -start,count +start,count @@
	hunkHeaderRegex := regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@`)

	// Track seen files to avoid duplicates
	seenFiles := make(map[string]bool)
	currentFile := ""
	currentFileLines := 0

	for _, line := range lines {
		// Extract file paths from --- and +++ lines
		if matches := filePathRegex.FindStringSubmatch(line); matches != nil {
			filePath := matches[2]
			// Remove leading a/ or b/ if present
			filePath = strings.TrimPrefix(filePath, "a/")
			filePath = strings.TrimPrefix(filePath, "b/")
			if !seenFiles[filePath] {
				files = append(files, filePath)
				seenFiles[filePath] = true
			}
			// Update current file when we see a new file path
			if currentFile != filePath {
				// Save previous file's line count
				if currentFile != "" {
					linesPerFile[currentFile] = currentFileLines
				}
				currentFile = filePath
				currentFileLines = 0
			}
		}

		// Detect hunk headers to track which file we're in
		if hunkHeaderRegex.MatchString(line) {
			// We're in a hunk, continue counting for current file
		}

		// Count changed lines (lines starting with + or -)
		// Skip hunk headers (@@ lines) and context lines
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			totalChanged++
			if currentFile != "" {
				currentFileLines++
			}
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			totalChanged++
			if currentFile != "" {
				currentFileLines++
			}
		}
	}

	// Save last file's line count
	if currentFile != "" {
		linesPerFile[currentFile] = currentFileLines
	}

	return files, totalChanged, linesPerFile, nil
}

// isForbiddenPath checks if a path matches any forbidden pattern.
func isForbiddenPath(path string) error {
	// Normalize path (handle both forward and backslashes)
	normalized := filepath.ToSlash(path)
	normalizedLower := strings.ToLower(normalized)

	// Check if path contains any forbidden substring or matches patterns
	for _, forbidden := range ForbiddenPaths {
		forbiddenLower := strings.ToLower(forbidden)

		// Handle wildcard patterns (e.g., ".env.*")
		if strings.HasSuffix(forbiddenLower, "*") {
			prefix := strings.TrimSuffix(forbiddenLower, "*")
			if strings.HasPrefix(normalizedLower, prefix) {
				return fmt.Errorf("path %s matches forbidden pattern: %s", path, forbidden)
			}
		} else {
			// Exact match or substring match
			if strings.Contains(normalizedLower, forbiddenLower) {
				return fmt.Errorf("path %s contains forbidden pattern: %s", path, forbidden)
			}
		}
	}

	// Check for absolute paths (security risk)
	if filepath.IsAbs(path) {
		return fmt.Errorf("path %s is absolute, must be relative to repo root", path)
	}

	// Check for paths that try to escape repo root
	if strings.Contains(path, "..") {
		return fmt.Errorf("path %s contains '..', which is not allowed", path)
	}

	return nil
}

// hasAllowedPrefix checks if a path starts with any of the allowed prefixes.
func hasAllowedPrefix(path string, prefixes []string) bool {
	normalized := filepath.ToSlash(path)
	for _, prefix := range prefixes {
		normalizedPrefix := filepath.ToSlash(prefix)
		if strings.HasPrefix(normalized, normalizedPrefix) {
			return true
		}
	}
	return false
}
