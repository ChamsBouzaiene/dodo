package sandbox

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/ChamsBouzaiene/dodo/internal/workspace"
)

// Import image package for PullOptions

// DockerRunner runs commands in isolated Docker containers.
type DockerRunner struct {
	client *client.Client
	config Config
}

// NewDockerRunner creates a new Docker-based runner.
func NewDockerRunner(config Config) (*DockerRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify Docker daemon is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cli.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("Docker daemon not accessible: %w", err)
	}

	return &DockerRunner{
		client: cli,
		config: config,
	}, nil
}

// RunCmd runs a command in an isolated Docker container.
func (r *DockerRunner) RunCmd(ctx context.Context, repoDir, name string, args []string, timeout time.Duration) (Result, error) {
	if timeout <= 0 {
		// Use config timeout if available, otherwise fall back to default
		if r.config.CmdTimeout > 0 {
			timeout = r.config.CmdTimeout
		} else {
			timeout = defaultCmdTimeout
		}
	}

	// Detect project type to select appropriate Docker image
	projectType := workspace.DetectProjectType(repoDir)
	image := GetDockerImage(projectType, r.config)

	// Ensure image is available (pull if needed)
	if err := r.ensureImage(ctx, image); err != nil {
		return Result{}, fmt.Errorf("failed to ensure image %s: %w", image, err)
	}

	// Convert repoDir to absolute path for mounting
	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return Result{}, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create container configuration
	containerConfig := &container.Config{
		Image:      image,
		Cmd:        append([]string{name}, args...),
		WorkingDir: "/workspace",
		User:       "1000:1000",           // Non-root user
		Env:        []string{"HOME=/tmp"}, // Set HOME to writable location
		// Disable network access
		NetworkDisabled: true,
	}

	// Host configuration with security restrictions
	hostConfig := &container.HostConfig{
		// Mount repository directory
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: absRepoDir,
				Target: "/workspace",
			},
		},
		// Resource limits
		Resources: container.Resources{
			Memory:   parseMemory(r.config.Memory),
			NanoCPUs: parseCPU(r.config.CPU) * 1e9, // Convert to nanoseconds
			Ulimits: []*units.Ulimit{
				{
					Name: "nofile",
					Soft: 1024,
					Hard: 1024,
				},
			},
		},
		// Security options
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		ReadonlyRootfs: true,
		// Temporary filesystem for /tmp
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=100m",
		},
		// Auto-remove container after execution
		AutoRemove: true,
	}

	// Create container
	createResp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return Result{}, fmt.Errorf("failed to create container: %w", err)
	}

	containerID := createResp.ID

	// Ensure cleanup on exit
	defer func() {
		// Try to remove container if it still exists
		removeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = r.client.ContainerRemove(removeCtx, containerID, container.RemoveOptions{
			Force: true,
		})
	}()

	// Create timeout context for container execution
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start container
	if err := r.client.ContainerStart(execCtx, containerID, container.StartOptions{}); err != nil {
		return Result{}, fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for container to finish (with timeout)
	statusCh, errCh := r.client.ContainerWait(execCtx, containerID, container.WaitConditionNotRunning)

	var exitCode int64
	select {
	case <-execCtx.Done():
		// Timeout - kill container
		killCtx, killCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer killCancel()
		_ = r.client.ContainerKill(killCtx, containerID, "SIGKILL")
		return Result{
			Code:     1,
			TimedOut: true,
			Stderr:   "Command execution timed out",
		}, execCtx.Err()
	case err := <-errCh:
		if err != nil {
			return Result{}, fmt.Errorf("container wait error: %w", err)
		}
	case status := <-statusCh:
		exitCode = status.StatusCode
	}

	// Check if we timed out
	timedOut := false
	if execCtx.Err() == context.DeadlineExceeded {
		timedOut = true
	}

	// Read logs (stdout and stderr)
	logs, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "all",
	})
	if err != nil {
		return Result{}, fmt.Errorf("failed to read container logs: %w", err)
	}
	defer logs.Close()

	// Parse stdout and stderr from logs
	// Docker combines them with headers, we need to separate them
	stdout, stderr := parseDockerLogs(logs)

	return Result{
		Stdout:   stdout,
		Stderr:   stderr,
		Code:     int(exitCode),
		TimedOut: timedOut,
	}, nil
}

// ensureImage checks if the image exists locally, and pulls it if not.
func (r *DockerRunner) ensureImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, _, err := r.client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		// Image exists locally
		return nil
	}

	// Image doesn't exist, pull it
	reader, err := r.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Drain the pull output (required for pull to complete)
	_, _ = io.Copy(io.Discard, reader)

	return nil
}

// parseDockerLogs parses Docker container logs and separates stdout from stderr.
// Docker logs use a header format: [8 bytes header][payload]
// Header format: [STREAM_TYPE (1 byte)][RESERVED (3 bytes)][SIZE (4 bytes)]
func parseDockerLogs(reader io.Reader) (stdout, stderr string) {
	var stdoutParts, stderrParts []string

	for {
		// Read header (8 bytes)
		header := make([]byte, 8)
		n, err := reader.Read(header)
		if n < 8 || err == io.EOF {
			break
		}
		if err != nil {
			// If we can't read header, try reading as plain text
			break
		}

		// Extract stream type (first byte)
		streamType := header[0]
		// Extract size (last 4 bytes, big-endian)
		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])

		// Limit size to prevent excessive memory allocation
		if size <= 0 || size > 10*1024*1024 { // 10MB max per log line
			continue
		}

		// Read payload
		payload := make([]byte, size)
		n, err = reader.Read(payload)
		if n != size || err != nil {
			break
		}

		content := string(payload)
		// Remove trailing newline if present (Docker adds one)
		content = strings.TrimSuffix(content, "\n")

		// Route to appropriate stream
		if streamType == 1 {
			// stdout
			stdoutParts = append(stdoutParts, content)
		} else if streamType == 2 {
			// stderr
			stderrParts = append(stderrParts, content)
		}
	}

	return strings.Join(stdoutParts, "\n"), strings.Join(stderrParts, "\n")
}

// parseMemory parses memory string (e.g., "1g", "512m") to bytes.
func parseMemory(memStr string) int64 {
	memStr = strings.ToLower(strings.TrimSpace(memStr))
	if memStr == "" {
		return 1024 * 1024 * 1024 // Default 1GB
	}

	var multiplier int64 = 1
	if strings.HasSuffix(memStr, "g") {
		multiplier = 1024 * 1024 * 1024
		memStr = strings.TrimSuffix(memStr, "g")
	} else if strings.HasSuffix(memStr, "m") {
		multiplier = 1024 * 1024
		memStr = strings.TrimSuffix(memStr, "m")
	} else if strings.HasSuffix(memStr, "k") {
		multiplier = 1024
		memStr = strings.TrimSuffix(memStr, "k")
	}

	var value int64
	fmt.Sscanf(memStr, "%d", &value)
	return value * multiplier
}

// parseCPU parses CPU string (e.g., "2", "1.5") to float64.
func parseCPU(cpuStr string) int64 {
	cpuStr = strings.TrimSpace(cpuStr)
	if cpuStr == "" {
		return 2 // Default 2 CPUs
	}

	var value float64
	fmt.Sscanf(cpuStr, "%f", &value)
	if value <= 0 {
		return 2
	}
	return int64(value)
}
