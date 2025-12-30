package indexer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitInfo contains git repository information.
type GitInfo struct {
	IsGit   bool
	GitRoot string
}

// DetectGit detects if a directory is within a git repository.
// Returns git information or falls back to non-git mode.
func DetectGit(ctx context.Context, repoRoot string) GitInfo {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = repoRoot
	
	output, err := cmd.Output()
	if err != nil {
		// Not a git repo or git not available
		return GitInfo{IsGit: false}
	}
	
	gitRoot := strings.TrimSpace(string(output))
	return GitInfo{
		IsGit:   true,
		GitRoot: gitRoot,
	}
}

// GitFileChange represents a file change detected by git.
type GitFileChange struct {
	Path   string
	Status string // "A" (added), "M" (modified), "D" (deleted), "R" (renamed), etc.
}

// GetGitChanges uses `git status --porcelain` to detect file changes.
// This is much faster than walking the filesystem for git repos.
func GetGitChanges(ctx context.Context, gitRoot string) ([]GitFileChange, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = gitRoot
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}
	
	var changes []GitFileChange
	scanner := bufio.NewScanner(bytes.NewReader(output))
	
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 4 {
			continue
		}
		
		// git status --porcelain format:
		// XY filename
		// where X is the index status and Y is the working tree status
		status := strings.TrimSpace(line[0:2])
		path := strings.TrimSpace(line[3:])
		
		// Handle renames (e.g., "R  old -> new")
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			if len(parts) == 2 {
				path = parts[1] // Use the new path
			}
		}
		
		// Map git status codes to simpler categories
		statusCode := "M" // default to modified
		if strings.Contains(status, "A") {
			statusCode = "A" // added
		} else if strings.Contains(status, "D") {
			statusCode = "D" // deleted
		} else if strings.Contains(status, "R") {
			statusCode = "M" // renamed = modified
		} else if strings.Contains(status, "??") {
			statusCode = "A" // untracked = added
		}
		
		changes = append(changes, GitFileChange{
			Path:   path,
			Status: statusCode,
		})
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse git status: %w", err)
	}
	
	return changes, nil
}

// GetGitTrackedFiles returns the list of files tracked by git.
// This is used for initial indexing to get the baseline file set.
func GetGitTrackedFiles(ctx context.Context, gitRoot string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = gitRoot
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files failed: %w", err)
	}
	
	var files []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	
	for scanner.Scan() {
		path := strings.TrimSpace(scanner.Text())
		if path != "" {
			files = append(files, path)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse git ls-files: %w", err)
	}
	
	return files, nil
}

// IsGitInstalled checks if git is available on the system.
func IsGitInstalled(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "git", "--version")
	err := cmd.Run()
	return err == nil
}

