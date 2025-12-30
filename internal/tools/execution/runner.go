package execution

import (
	"context"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/sandbox"
)

// Runner defines the interface for running commands.
// This allows mocking the sandbox runner for testing.
type Runner interface {
	RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error)
}

// SandboxRunner is the default implementation that uses the sandbox package.
// It automatically selects the appropriate runner (Docker or host) based on configuration.
type SandboxRunner struct {
	runner sandbox.Runner
}

// NewSandboxRunner creates a new SandboxRunner using the default sandbox configuration.
func NewSandboxRunner() *SandboxRunner {
	return &SandboxRunner{
		runner: sandbox.NewDefaultRunner(),
	}
}

// RunCmd calls the underlying sandbox runner.
func (r *SandboxRunner) RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (sandbox.Result, error) {
	return r.runner.RunCmd(ctx, repoDir, name, args, timeout)
}
