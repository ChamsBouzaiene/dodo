package indexer

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// WorkspaceContext contains context information about the workspace.
type WorkspaceContext struct {
	UserInfo      UserInfo
	ProjectLayout string
	GitStatus     GitStatus
	ProjectType   string
}

// UserInfo contains information about the user's environment.
type UserInfo struct {
	OS    string
	Shell string
}

// GitStatus contains git repository status.
type GitStatus struct {
	Branch string
	Status string // "clean" or "dirty"
}

// GenerateWorkspaceContext generates workspace context for the agent.
func GenerateWorkspaceContext(ctx context.Context, repoRoot string, gitInfo GitInfo) (*WorkspaceContext, error) {
	startTime := time.Now()

	// 1. Get user info
	userInfo := UserInfo{
		OS:    runtime.GOOS,
		Shell: os.Getenv("SHELL"),
	}
	if userInfo.Shell == "" {
		userInfo.Shell = "/bin/sh" // fallback
	}

	// 2. Generate project layout (file tree)
	// Use maxDepth=4 to show files inside leaf directories like internal/adapter/terminal/
	projectLayout, err := generateProjectLayout(repoRoot, 4)
	if err != nil {
		return nil, fmt.Errorf("failed to generate project layout: %w", err)
	}

	// Debug: log the actual project layout
	log.Printf("📁 Generated project layout (%d bytes):\n%s", len(projectLayout), projectLayout)

	// 3. Get git status
	gitStatus := GitStatus{
		Branch: "unknown",
		Status: "unknown",
	}
	if gitInfo.IsGit {
		gitStatus = getGitStatus(ctx, gitInfo.GitRoot)
	}

	// 4. Detect project type (inline to avoid import cycle)
	projectType := detectProjectType(repoRoot)

	contextDuration := time.Since(startTime)

	// Create context and estimate tokens
	wc := &WorkspaceContext{
		UserInfo:      userInfo,
		ProjectLayout: projectLayout,
		GitStatus:     gitStatus,
		ProjectType:   projectType,
	}

	contextXML := wc.FormatAsXML()
	estimatedTokens := estimateTokens(contextXML)

	log.Printf("⏱️  Context preparation: %s | Size: %d bytes | Est. tokens: ~%d",
		contextDuration.Round(time.Millisecond),
		len(contextXML),
		estimatedTokens)

	return wc, nil
}

// detectProjectType detects the project type using manifest-first detection.
// This is a simplified version to avoid import cycles with the tools package.
func detectProjectType(repoRoot string) string {
	// Manifest-first detection: check for config files
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "package.json")); err == nil {
		return "node"
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "pyproject.toml")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "requirements.txt")); err == nil {
		return "python"
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "Cargo.toml")); err == nil {
		return "rust"
	}

	return "unknown"
}

// FormatAsXML formats the workspace context as XML for injection into system prompt.
func (wc *WorkspaceContext) FormatAsXML() string {
	return fmt.Sprintf(`<workspace_context>
<user_info>
  <os>%s</os>
  <shell>%s</shell>
</user_info>

<project_layout>
%s
</project_layout>

<git_status>
  <branch>%s</branch>
  <status>%s</status>
</git_status>

<project_type>%s</project_type>
</workspace_context>`,
		wc.UserInfo.OS,
		wc.UserInfo.Shell,
		wc.ProjectLayout,
		wc.GitStatus.Branch,
		wc.GitStatus.Status,
		wc.ProjectType,
	)
}

// generateProjectLayout creates a file tree representation of the project.
// maxDepth controls how deep to traverse (e.g., 3 levels).
func generateProjectLayout(repoRoot string, maxDepth int) (string, error) {
	const maxFilesPerDir = 15
	var sb strings.Builder

	// Write root path
	sb.WriteString(repoRoot + "/\n")

	// Load .gitignore patterns
	ignorePatterns := loadGitignorePatterns(repoRoot)

	// Build tree recursively
	if err := buildFileTree(&sb, repoRoot, "", 0, maxDepth, maxFilesPerDir, ignorePatterns); err != nil {
		return "", err
	}

	return sb.String(), nil
}

// buildFileTree recursively builds the file tree string.
func buildFileTree(sb *strings.Builder, currentPath, indent string, depth, maxDepth, maxFilesPerDir int, ignorePatterns []string) error {
	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	// Filter out ignored files and common junk
	var filteredEntries []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()
		relPath := filepath.Join(currentPath, name)

		// Skip common junk directories
		if isJunkPath(name) {
			continue
		}

		// Skip backup files (.orig, .bak, ~)
		if strings.HasSuffix(name, ".orig") || strings.HasSuffix(name, ".bak") || strings.HasSuffix(name, "~") {
			continue
		}

		// Skip .gitignore patterns
		if shouldIgnore(relPath, ignorePatterns) {
			continue
		}

		filteredEntries = append(filteredEntries, entry)
	}

	// Sort entries: directories first, then files
	sort.Slice(filteredEntries, func(i, j int) bool {
		if filteredEntries[i].IsDir() != filteredEntries[j].IsDir() {
			return filteredEntries[i].IsDir()
		}
		return filteredEntries[i].Name() < filteredEntries[j].Name()
	})

	// If too many files, show count instead
	if len(filteredEntries) > maxFilesPerDir {
		fileCount := 0
		dirCount := 0
		for _, entry := range filteredEntries {
			if entry.IsDir() {
				dirCount++
			} else {
				fileCount++
			}
		}

		// Still show directories, but summarize files
		shownDirs := 0
		for _, entry := range filteredEntries {
			if entry.IsDir() && shownDirs < maxFilesPerDir {
				sb.WriteString(fmt.Sprintf("%s  - %s/\n", indent, entry.Name()))
				shownDirs++

				// Recurse into directory
				nextIndent := indent + "    "
				nextPath := filepath.Join(currentPath, entry.Name())
				if err := buildFileTree(sb, nextPath, nextIndent, depth+1, maxDepth, maxFilesPerDir, ignorePatterns); err != nil {
					// Continue on error (permission denied, etc.)
					continue
				}
			}
		}

		if fileCount > 0 {
			sb.WriteString(fmt.Sprintf("%s  - [%d files]\n", indent, fileCount))
		}
		return nil
	}

	// Show all entries
	for _, entry := range filteredEntries {
		if entry.IsDir() {
			sb.WriteString(fmt.Sprintf("%s  - %s/\n", indent, entry.Name()))

			// Recurse into directory
			nextIndent := indent + "    "
			nextPath := filepath.Join(currentPath, entry.Name())
			if err := buildFileTree(sb, nextPath, nextIndent, depth+1, maxDepth, maxFilesPerDir, ignorePatterns); err != nil {
				// Continue on error (permission denied, etc.)
				continue
			}
		} else {
			// Add file size metadata for better decision making
			var sizeTag string
			lines := estimateLineCount(filepath.Join(currentPath, entry.Name()))
			if lines > 500 {
				sizeTag = " [LARGE: " + fmt.Sprintf("%d", lines) + " lines]"
			} else if lines > 200 {
				sizeTag = " [" + fmt.Sprintf("%d", lines) + " lines]"
			}
			sb.WriteString(fmt.Sprintf("%s  - %s%s\n", indent, entry.Name(), sizeTag))
		}
	}

	return nil
}

// estimateLineCount quickly estimates the number of lines in a file.
// Returns 0 if the file cannot be read or is binary.
func estimateLineCount(filePath string) int {
	// Skip binary files
	ext := filepath.Ext(filePath)
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".db": true, ".sqlite": true, ".bleve": true,
	}
	if binaryExts[ext] {
		return 0
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	// Quick line count
	return strings.Count(string(content), "\n") + 1
}

// isJunkPath checks if a path is a common junk directory/file to skip.
func isJunkPath(name string) bool {
	junkPaths := []string{
		".git",
		".dodo", // Dodo's own indexing directory
		"node_modules",
		"dist",
		"build",
		"venv",
		".venv",
		"__pycache__",
		".pytest_cache",
		".mypy_cache",
		"target", // Rust
		".idea",
		".vscode",
		".DS_Store",
	}

	for _, junk := range junkPaths {
		if name == junk {
			return true
		}
	}
	return false
}

// loadGitignorePatterns loads patterns from .gitignore for filtering.
func loadGitignorePatterns(repoRoot string) []string {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil
	}

	var patterns []string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// shouldIgnore checks if a path matches any .gitignore pattern.
// This is a simplified implementation - for production, use a proper gitignore library.
func shouldIgnore(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	base := filepath.Base(path)
	for _, pattern := range patterns {
		// Simple pattern matching (not full gitignore spec)
		if pattern == base {
			return true
		}
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(base, strings.TrimPrefix(pattern, "*")) {
			return true
		}
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(base, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

// estimateTokens provides a rough token count estimation.
// Uses a simple heuristic: ~4 characters per token for English/code.
func estimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}

	// Rough estimation: ~4 characters per token
	charCount := len([]rune(text))
	whitespaceCount := strings.Count(text, " ") + strings.Count(text, "\n") + strings.Count(text, "\t")

	// Rough formula: (characters / 4) + (whitespace / 6)
	estimated := (charCount / 4) + (whitespaceCount / 6)

	if estimated < 1 {
		return 1
	}

	return estimated
}

// getGitStatus gets the current git branch and clean/dirty status.
func getGitStatus(ctx context.Context, gitRoot string) GitStatus {
	status := GitStatus{
		Branch: "unknown",
		Status: "unknown",
	}

	// Get branch name
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gitRoot
	if output, err := cmd.Output(); err == nil {
		status.Branch = strings.TrimSpace(string(output))
	}

	// Check if repo is clean or dirty
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = gitRoot
	if output, err := cmd.Output(); err == nil {
		if len(strings.TrimSpace(string(output))) == 0 {
			status.Status = "clean"
		} else {
			status.Status = "dirty"
		}
	}

	return status
}
