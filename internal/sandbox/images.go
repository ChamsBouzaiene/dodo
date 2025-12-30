package sandbox

import (
	"github.com/ChamsBouzaiene/dodo/internal/workspace"
)

// GetDockerImage returns the appropriate Docker image for a project type.
// If a custom image is specified in config, it takes precedence.
// Otherwise, returns a default lightweight image for the project type.
func GetDockerImage(projectType workspace.ProjectType, config Config) string {
	// Custom image override takes precedence
	if config.DockerImage != "" {
		return config.DockerImage
	}

	// Default images per project type (using lightweight alpine variants)
	switch projectType {
	case workspace.ProjectTypeGo:
		return "golang:alpine"
	case workspace.ProjectTypeNode:
		return "node:alpine"
	case workspace.ProjectTypePython:
		return "python:alpine"
	case workspace.ProjectTypeRust:
		return "rust:alpine"
	default:
		// Fallback to a generic image with common tools
		// Using alpine as base since it's lightweight
		return "alpine:latest"
	}
}






