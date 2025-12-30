package sandbox

import (
	"context"
	"time"
)

// Result captures output of a command.
type Result struct {
	Stdout   string
	Stderr   string
	Code     int
	TimedOut bool
}

// Runner defines the interface for running commands in a sandboxed environment.
// Implementations should provide isolation from the host system to prevent
// malicious commands from affecting the host.
type Runner interface {
	// RunCmd runs a command in the given repo directory with a timeout.
	// - ctx: base context for cancellation
	// - repoDir: path to repository root on disk
	// - name: executable name, e.g. "go"
	// - args: arguments, e.g. []string{"test", "./..."}
	// - timeout: optional timeout (<=0 uses default)
	RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (Result, error)
}

// RunCmd is a convenience function that uses the default runner.
// It will attempt to use Docker if available, falling back to host execution.
// For explicit control, use NewRunner() to get a specific runner implementation.
func RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (Result, error) {
	runner := NewDefaultRunner()
	return runner.RunCmd(ctx, repoDir, name, args, timeout)
}
