package execution

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/workspace"
)

// runBuildImpl copies the implementation from tools/run_build.go
func runBuildImpl(ctx context.Context, runner Runner, repoRoot string) (string, error) {
	// Detect project type
	projectType := workspace.DetectProjectType(repoRoot)
	if projectType == workspace.ProjectTypeUnknown {
		execResult := engine.ExecutionResult{
			Cmd:      "",
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "Could not detect project type",
			Status:   "unavailable",
			Reason:   "not_configured",
		}
		resultJSON, _ := json.Marshal(execResult)
		return string(resultJSON), nil
	}

	// Get build command
	cmdName, args := workspace.GetBuildCommand(projectType)
	if cmdName == "" {
		// Python usually doesn't have a build step
		if projectType == workspace.ProjectTypePython {
			execResult := engine.ExecutionResult{
				Cmd:      "",
				ExitCode: 0,
				Stdout:   "Python projects typically don't require a build step",
				Stderr:   "",
				Status:   "unavailable",
				Reason:   "not_configured",
			}
			resultJSON, _ := json.Marshal(execResult)
			return string(resultJSON), nil
		}

		execResult := engine.ExecutionResult{
			Cmd:      "",
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "No build command available for project type: " + string(projectType),
			Status:   "unavailable",
			Reason:   "not_configured",
		}
		resultJSON, _ := json.Marshal(execResult)
		return string(resultJSON), nil
	}

	// Run the command
	result, err := runner.RunCmd(ctx, repoRoot, cmdName, args, 0)
	if err != nil {
		// Log but continue
	}

	// Build full command string for output
	cmdStr := cmdName
	for _, arg := range args {
		cmdStr += " " + arg
	}

	execResult := engine.ExecutionResult{
		Cmd:      cmdStr,
		ExitCode: result.Code,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Status:   "ok",
	}
	if result.Code != 0 {
		execResult.Status = "failed"
		if err != nil && (strings.Contains(err.Error(), "executable file not found") || strings.Contains(result.Stdout, "command not found")) {
			execResult.Status = "unavailable"
			execResult.Reason = "command_not_found"
		}
	}

	resultJSON, err := json.Marshal(execResult)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewRunBuildTool creates an engine.Tool that wraps the run_build functionality.
func NewRunBuildTool(repoRoot string) engine.Tool {
	runner := NewSandboxRunner()
	return engine.Tool{
		Name:        "run_build",
		Description: "Runs the appropriate build command for the project type. Auto-detects project type (Go, Node, Python, Rust) and runs the corresponding build command.",
		SchemaJSON:  `{"type":"object","properties":{},"required":[]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			return runBuildImpl(ctx, runner, repoRoot)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "execution",
			Tags:     []string{"idempotent"},
		},
	}
}
