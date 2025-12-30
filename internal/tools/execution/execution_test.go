package execution

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/sandbox"
)

// MockRunner is a mock implementation of the Runner interface.
type MockRunner struct {
	RunCmdFunc func(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error)
}

func (m *MockRunner) RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error) {
	if m.RunCmdFunc != nil {
		return m.RunCmdFunc(ctx, repoDir, name, args, timeout)
	}
	return sandbox.Result{}, nil
}

func TestRunCmdImpl(t *testing.T) {
	tests := []struct {
		name           string
		cmd            string
		args           string
		allowed        bool
		mockResult     sandbox.Result
		mockErr        error
		expectedStatus string
		expectedStdout string
	}{
		{
			name:           "Allowed command",
			cmd:            "go",
			args:           "version",
			allowed:        true,
			mockResult:     sandbox.Result{Stdout: "go version 1.20", Code: 0},
			expectedStatus: "", // Status is not explicitly returned in JSON for run_cmd, but exit_code is
			expectedStdout: "go version 1.20",
		},
		{
			name:           "Disallowed command",
			cmd:            "forbidden_cmd",
			args:           "--some-arg",
			allowed:        false,
			expectedStdout: "",
		},
		{
			name:           "Git command",
			cmd:            "git",
			args:           "status",
			allowed:        true,
			mockResult:     sandbox.Result{Stdout: "On branch main", Code: 0},
			expectedStdout: "On branch main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &MockRunner{
				RunCmdFunc: func(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error) {
					return tt.mockResult, tt.mockErr
				},
			}

			resultJSON, err := runCmdImpl(context.Background(), runner, "/tmp", tt.cmd, tt.args, 0, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var execResult engine.ExecutionResult
			if err := json.Unmarshal([]byte(resultJSON), &execResult); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if !tt.allowed {
				if execResult.Stderr == "" {
					t.Error("expected stderr for disallowed command")
				}
				return
			}

			if execResult.Stdout != tt.expectedStdout {
				t.Errorf("expected stdout %q, got %q", tt.expectedStdout, execResult.Stdout)
			}
		})
	}
}

func TestRunBuildImpl(t *testing.T) {
	// Note: Testing runBuildImpl requires mocking workspace.DetectProjectType which is hardcoded.
	// For now, we can only test the "unknown" project type path or if we had a way to mock the filesystem.
	// Since we can't easily mock the filesystem for workspace detection in this package without further refactoring,
	// we will skip detailed logic tests for build/test tools and focus on the fact that they call the runner.
	// A better approach would be to inject the project type detector as well.
}
