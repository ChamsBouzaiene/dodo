package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents the type of project.
type ProjectType string

const (
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeUnknown ProjectType = "unknown"
)

// DetectProjectType detects the project type using manifest-first detection with extension fallback.
func DetectProjectType(repoRoot string) ProjectType {
	// Manifest-first detection: check for config files
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return ProjectTypeGo
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "package.json")); err == nil {
		return ProjectTypeNode
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "pyproject.toml")); err == nil {
		return ProjectTypePython
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "requirements.txt")); err == nil {
		return ProjectTypePython
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "Cargo.toml")); err == nil {
		return ProjectTypeRust
	}

	// Extension fallback: scan repo root for common file extensions
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return ProjectTypeUnknown
	}

	extCounts := make(map[string]int)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != "" {
			extCounts[ext]++
		}
	}

	// Count extensions to determine project type
	goCount := extCounts[".go"]
	nodeCount := extCounts[".ts"] + extCounts[".tsx"] + extCounts[".js"] + extCounts[".jsx"]
	pythonCount := extCounts[".py"]
	rustCount := extCounts[".rs"]

	maxCount := 0
	var detectedType ProjectType = ProjectTypeUnknown

	if goCount > maxCount {
		maxCount = goCount
		detectedType = ProjectTypeGo
	}
	if nodeCount > maxCount {
		maxCount = nodeCount
		detectedType = ProjectTypeNode
	}
	if pythonCount > maxCount {
		maxCount = pythonCount
		detectedType = ProjectTypePython
	}
	if rustCount > maxCount {
		maxCount = rustCount
		detectedType = ProjectTypeRust
	}

	// Only return detected type if we found a reasonable number of files
	if maxCount >= 3 {
		return detectedType
	}

	return ProjectTypeUnknown
}

// GetLintCommand returns the lint command for a project type.
func GetLintCommand(projectType ProjectType) (string, []string) {
	switch projectType {
	case ProjectTypeGo:
		return "gofmt", []string{"-l", "."}
	case ProjectTypeNode:
		// Try npm run lint first, fallback to eslint
		return "npm", []string{"run", "lint"}
	case ProjectTypePython:
		return "ruff", []string{"check", "."}
	case ProjectTypeRust:
		return "cargo", []string{"clippy", "--", "-D", "warnings"}
	default:
		return "", nil
	}
}

// GetBuildCommand returns the build command for a project type.
func GetBuildCommand(projectType ProjectType) (string, []string) {
	switch projectType {
	case ProjectTypeGo:
		return "go", []string{"build", "./..."}
	case ProjectTypeNode:
		// Try npm run build first, fallback to tsc
		return "npm", []string{"run", "build"}
	case ProjectTypePython:
		// Python usually doesn't have a build step
		return "", nil
	case ProjectTypeRust:
		return "cargo", []string{"build"}
	default:
		return "", nil
	}
}

// GetTestCommand returns the test command for a project type.
func GetTestCommand(projectType ProjectType) (string, []string) {
	switch projectType {
	case ProjectTypeGo:
		return "go", []string{"test", "./..."}
	case ProjectTypeNode:
		return "npm", []string{"test"}
	case ProjectTypePython:
		return "pytest", []string{}
	case ProjectTypeRust:
		return "cargo", []string{"test"}
	default:
		return "", nil
	}
}
