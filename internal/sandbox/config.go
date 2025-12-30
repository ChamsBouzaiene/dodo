package sandbox

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Mode represents the sandbox execution mode.
type Mode string

const (
	// ModeDocker uses Docker containers for isolation.
	ModeDocker Mode = "docker"
	// ModeHost runs commands directly on the host (no isolation).
	ModeHost Mode = "host"
	// ModeAuto automatically selects Docker if available, otherwise falls back to host.
	ModeAuto Mode = "auto"
)

// Config holds configuration for sandbox execution.
type Config struct {
	Mode        Mode
	DockerImage string        // Custom Docker image override
	CPU         string        // CPU limit (e.g., "2")
	Memory      string        // Memory limit (e.g., "1g")
	CmdTimeout  time.Duration // Default command timeout (0 = use default)
}

// DefaultConfig returns the default configuration based on environment variables.
func DefaultConfig() Config {
	modeStr := strings.ToLower(os.Getenv("DODO_SANDBOX_MODE"))
	if modeStr == "" {
		modeStr = "auto"
	}

	var mode Mode
	switch modeStr {
	case "docker":
		mode = ModeDocker
	case "host":
		mode = ModeHost
	case "auto":
		mode = ModeAuto
	default:
		log.Printf("WARNING: Unknown DODO_SANDBOX_MODE value '%s', defaulting to 'auto'", modeStr)
		mode = ModeAuto
	}

	// Parse command timeout from environment (in seconds)
	cmdTimeout := 2 * time.Minute // Default: 2 minutes
	if timeoutStr := os.Getenv("DODO_CMD_TIMEOUT"); timeoutStr != "" {
		if seconds, err := time.ParseDuration(timeoutStr); err == nil && seconds > 0 {
			cmdTimeout = seconds
		} else {
			log.Printf("WARNING: Invalid DODO_CMD_TIMEOUT value '%s', using default 2m", timeoutStr)
		}
	}

	return Config{
		Mode:        mode,
		DockerImage: os.Getenv("DODO_DOCKER_IMAGE"),
		CPU:         getEnvOrDefault("DODO_DOCKER_CPU", "2"),
		Memory:      getEnvOrDefault("DODO_DOCKER_MEMORY", "1g"),
		CmdTimeout:  cmdTimeout,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// IsDockerAvailable checks if Docker is available and accessible.
func IsDockerAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "ps")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}

// NewDefaultRunner creates a runner based on the configuration and Docker availability.
// It respects the DODO_SANDBOX_MODE environment variable:
// - "docker": Use Docker (fails if unavailable)
// - "host": Use host executor (no isolation)
// - "auto": Use Docker if available, fallback to host
func NewDefaultRunner() Runner {
	config := DefaultConfig()
	ctx := context.Background()

	switch config.Mode {
	case ModeDocker:
		if !IsDockerAvailable(ctx) {
			log.Printf("WARNING: Docker mode requested but Docker is not available. Falling back to host executor.")
			return &HostRunner{config: config}
		}
		dockerRunner, err := NewDockerRunner(config)
		if err != nil {
			log.Printf("WARNING: Failed to create Docker runner: %v. Falling back to host executor.", err)
			return &HostRunner{config: config}
		}
		return dockerRunner

	case ModeHost:
		log.Printf("WARNING: Using host executor (no sandboxing). This is insecure and should only be used for development.")
		return &HostRunner{config: config}

	case ModeAuto:
		if IsDockerAvailable(ctx) {
			dockerRunner, err := NewDockerRunner(config)
			if err != nil {
				log.Printf("WARNING: Docker available but failed to create runner: %v. Falling back to host executor.", err)
				return &HostRunner{config: config}
			}
			return dockerRunner
		}
		log.Printf("WARNING: Docker not available. Using host executor (no sandboxing). This is insecure.")
		return &HostRunner{config: config}

	default:
		log.Printf("WARNING: Unknown sandbox mode, defaulting to host executor.")
		return &HostRunner{config: config}
	}
}

// NewRunner creates a specific runner implementation.
func NewRunner(mode Mode, config Config) (Runner, error) {
	switch mode {
	case ModeDocker:
		return NewDockerRunner(config)
	case ModeHost:
		return &HostRunner{config: config}, nil
	default:
		return nil, fmt.Errorf("unknown runner mode: %s", mode)
	}
}
