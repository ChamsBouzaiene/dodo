package execution

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/workspace"
)

// runTestsImpl copies the implementation from tools/run_tests.go
func runTestsImpl(ctx context.Context, runner Runner, repoRoot string) (string, error) {
	// Detect project type
	projectType := workspace.DetectProjectType(repoRoot)
	if projectType == workspace.ProjectTypeUnknown {
		passed := false
		execResult := engine.ExecutionResult{
			Cmd:      "",
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "Could not detect project type",
			Passed:   &passed,
			Status:   "unavailable",
			Reason:   "not_configured",
		}
		resultJSON, _ := json.Marshal(execResult)
		return string(resultJSON), nil
	}

	// Get test command
	cmdName, args := workspace.GetTestCommand(projectType)
	if cmdName == "" {
		passed := false
		execResult := engine.ExecutionResult{
			Cmd:      "",
			ExitCode: 1,
			Stdout:   "",
			Stderr:   "No test command available for project type: " + string(projectType),
			Passed:   &passed,
			Status:   "unavailable",
			Reason:   "not_configured",
		}
		resultJSON, _ := json.Marshal(execResult)
		return string(resultJSON), nil
	}

	// Run the command
	res, err := runner.RunCmd(ctx, repoRoot, cmdName, args, 0)
	if err != nil {
		// Even if there's an error, return the result
	}

	// Build full command string for output
	cmdStr := cmdName
	for _, arg := range args {
		cmdStr += " " + arg
	}

	// Check if tests passed
	passed := (err == nil && res.Code == 0)

	execResult := engine.ExecutionResult{
		Cmd:      cmdStr,
		ExitCode: res.Code,
		Stdout:   res.Stdout,
		Stderr:   res.Stderr,
		Passed:   &passed,
		Status:   "ok",
	}
	if !passed {
		execResult.Status = "failed"
		if err != nil && (strings.Contains(err.Error(), "executable file not found") || strings.Contains(res.Stdout, "command not found")) {
			execResult.Status = "unavailable"
			execResult.Reason = "command_not_found"
		}
		if strings.Contains(res.Stdout, "no tests found") || strings.Contains(res.Stdout, "no tests") {
			execResult.Status = "unavailable"
			execResult.Reason = "not_configured"
		}
	}

	resultJSON, err := json.Marshal(execResult)
	if err != nil {
		return "", err
	}

	return string(resultJSON), nil
}

// NewRunTestsTool creates an engine.Tool that wraps the run_tests functionality.
func NewRunTestsTool(repoRoot string) engine.Tool {
	runner := NewSandboxRunner()
	return engine.Tool{
		Name:        "run_tests",
		Description: "Runs the appropriate test command for the project type. Auto-detects project type (Go, Node, Python, Rust) and runs the corresponding test command.",
		SchemaJSON:  `{"type":"object","properties":{},"required":[]}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			return runTestsImpl(ctx, runner, repoRoot)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "execution",
			Tags:     []string{"idempotent"},
		},
	}
}
