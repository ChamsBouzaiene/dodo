//go:build !windows
// +build !windows

package sandbox

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"
)

const defaultCmdTimeout = 2 * time.Minute

// HostRunner runs commands directly on the host machine without isolation.
// This is the original implementation that provides no security sandboxing.
// It should only be used when Docker is unavailable or explicitly requested.
type HostRunner struct {
	config Config // Optional config for timeout override
}

// RunCmd runs a command in the given repo directory with a timeout.
// - ctx: base context
// - repoDir: path to repository root on disk
// - name: executable name, e.g. "go"
// - args: arguments, e.g. []string{"test", "./..."}
// - timeout: optional timeout (<=0 uses default)
func (r *HostRunner) RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (Result, error) {
	if timeout <= 0 {
		// Use config timeout if available, otherwise fall back to default
		if r.config.CmdTimeout > 0 {
			timeout = r.config.CmdTimeout
		} else {
			timeout = defaultCmdTimeout
		}
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(name, args...)
	cmd.Dir = repoDir
	// Create a new process group so we can kill all child processes on cancel
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Start()
	if err != nil {
		return Result{}, err
	}

	// Goroutine to kill the entire process group on context cancellation
	done := make(chan struct{})
	go func() {
		select {
		case <-cctx.Done():
			// Kill the entire process group (negative PID)
			if cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
		case <-done:
			// Process finished normally, nothing to do
		}
	}()

	waitErr := cmd.Wait()
	close(done) // Signal the goroutine to exit

	res := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
		Code:   0,
	}

	if waitErr != nil {
		res.Code = 1
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			res.Code = exitErr.ExitCode()
		}
		if errors.Is(cctx.Err(), context.DeadlineExceeded) || errors.Is(cctx.Err(), context.Canceled) {
			res.TimedOut = true
		}
		return res, waitErr
	}

	if errors.Is(cctx.Err(), context.DeadlineExceeded) || errors.Is(cctx.Err(), context.Canceled) {
		res.TimedOut = true
	}

	return res, nil
}
