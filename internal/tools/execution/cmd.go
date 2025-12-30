package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

const (
	defaultRunCmdTimeout = 60 * time.Second
	maxRunCmdTimeout     = 5 * time.Minute
	minRunCmdTimeout     = 5 * time.Second
	defaultRunCmdLines   = 40
	minRunCmdLines       = 5
	maxRunCmdLines       = 200
	maxRunCmdChars       = 4000
)

var runCmdAllowedCommands = []string{
	// Build tools
	"go", "gofmt", "goimports",
	"npm", "npx", "yarn", "pnpm", "bun",
	"python", "python3", "pip", "pip3", "pytest", "uv",
	"cargo", "rustc", "rustfmt",
	"make", "cmake",
	"gradle", "mvn",

	// Linters & formatters
	"eslint", "prettier", "biome",
	"ruff", "black", "isort", "mypy", "flake8",
	"tsc", "node",
	"golangci-lint",
	"shellcheck",

	// File operations (safe)
	"mkdir", "touch", "rm", "cp", "mv",
	"cat", "head", "tail", "less",
	"ls", "find", "tree",
	"wc", "grep", "awk", "sed", "sort", "uniq", "diff",

	// Version control
	"git",

	// Network (read-only / safe)
	"curl", "wget",

	// Shell
	"sh", "bash", "zsh",

	// Utilities
	"echo", "printf", "date", "which", "env",
	"tar", "zip", "unzip", "gzip", "gunzip",
	"jq", "yq",
}

// runCmdImpl copies the implementation from tools/run_cmd.go
func runCmdImpl(ctx context.Context, runner Runner, repoRoot, cmd, argsStr string, timeout time.Duration, maxLines int) (string, error) {
	isAllowed := false
	for _, allowed := range runCmdAllowedCommands {
		if cmd == allowed {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		execResult := engine.ExecutionResult{
			Cmd:      cmd,
			ExitCode: 1,
			Stdout:   "",
			Stderr:   fmt.Sprintf("Command '%s' is not in allowlist. Allowed commands: %s", cmd, strings.Join(runCmdAllowedCommands, ", ")),
			Status:   "failed",
		}
		resultJSON, _ := json.Marshal(execResult)
		return string(resultJSON), nil
	}

	var args []string
	if argsStr != "" {
		args = parseArgs(argsStr)
	}

	cmdCtx := ctx
	if timeout <= 0 {
		timeout = defaultRunCmdTimeout
	}
	if timeout > maxRunCmdTimeout {
		timeout = maxRunCmdTimeout
	}

	result, err := runner.RunCmd(cmdCtx, repoRoot, cmd, args, timeout)

	cmdStr := cmd
	if len(args) > 0 {
		cmdStr += " " + strings.Join(args, " ")
	}

	if maxLines <= 0 {
		maxLines = defaultRunCmdLines
	} else if maxLines > maxRunCmdLines {
		maxLines = maxRunCmdLines
	}

	stdout, stdoutTruncated := truncateOutput(result.Stdout, maxLines)
	stderr, stderrTruncated := truncateOutput(result.Stderr, maxLines)

	execResult := engine.ExecutionResult{
		Cmd:             cmdStr,
		ExitCode:        result.Code,
		Stdout:          stdout,
		Stderr:          stderr,
		StdoutTruncated: stdoutTruncated,
		StderrTruncated: stderrTruncated,
		Status:          "ok",
	}
	if result.TimedOut || errors.Is(err, context.DeadlineExceeded) {
		execResult.TimedOut = true
		execResult.Status = "failed"
	}
	if result.Code != 0 {
		execResult.Status = "failed"
	}

	resultJSON, marshalErr := json.Marshal(execResult)
	if marshalErr != nil {
		return "", marshalErr
	}

	return string(resultJSON), nil
}

// parseArgs parses a space-separated argument string into a slice of strings.
func parseArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(argsStr); i++ {
		char := argsStr[i]

		if char == '"' || char == '\'' {
			if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteByte(char)
			}
		} else if char == ' ' && !inQuotes {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func parseTimeoutArg(value any) time.Duration {
	if value == nil {
		return defaultRunCmdTimeout
	}
	var seconds float64
	switch v := value.(type) {
	case float64:
		seconds = v
	case int:
		seconds = float64(v)
	default:
		return defaultRunCmdTimeout
	}
	if seconds <= 0 {
		return defaultRunCmdTimeout
	}
	timeout := time.Duration(seconds) * time.Second
	if timeout < minRunCmdTimeout {
		timeout = minRunCmdTimeout
	}
	if timeout > maxRunCmdTimeout {
		timeout = maxRunCmdTimeout
	}
	return timeout
}

func parseMaxOutputLinesArg(value any) int {
	if value == nil {
		return defaultRunCmdLines
	}
	var lines int
	switch v := value.(type) {
	case float64:
		lines = int(v)
	case int:
		lines = v
	default:
		return defaultRunCmdLines
	}
	if lines < minRunCmdLines {
		lines = minRunCmdLines
	}
	if lines > maxRunCmdLines {
		lines = maxRunCmdLines
	}
	return lines
}

func truncateOutput(output string, maxLines int) (string, bool) {
	if output == "" {
		return "", false
	}
	truncated := false
	lines := strings.Split(output, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}
	joined := strings.Join(lines, "\n")
	if len(joined) > maxRunCmdChars {
		joined = joined[:maxRunCmdChars]
		truncated = true
	}
	return joined, truncated
}

// NewRunCmdTool creates an engine.Tool that wraps the run_cmd functionality.
func NewRunCmdTool(repoRoot string) engine.Tool {
	runner := NewSandboxRunner()
	return engine.Tool{
		Name:        "run_cmd",
		Description: "Runs a command with strict allowlist enforcement. Allowed: build tools (go, npm, yarn, python, pip, cargo, make), linters (eslint, prettier, ruff, tsc), file ops (ls, cat, grep, find, mkdir, rm, cp), git, curl/wget, shells (sh, bash), and utilities (jq, tar, zip). Supports optional timeout and output truncation.",
		SchemaJSON: `{
			"type": "object",
			"properties": {
				"cmd": {"type":"string","description":"Command name (must be in allowlist)"},
				"args": {"type":"string","description":"Command arguments as space-separated string"},
				"timeout_seconds": {"type":"integer","minimum":5,"maximum":300,"description":"Maximum seconds to allow the command to run (default: 60)"},
				"max_output_lines": {"type":"integer","minimum":5,"maximum":200,"description":"Maximum stdout/stderr lines to return (default: 40)"}
			},
			"required": ["cmd"]
		}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			cmd, ok := args["cmd"].(string)
			if !ok {
				return "", fmt.Errorf("cmd must be a string")
			}
			argsStr := ""
			if a, ok := args["args"].(string); ok {
				argsStr = a
			}
			timeout := parseTimeoutArg(args["timeout_seconds"])
			maxLines := parseMaxOutputLinesArg(args["max_output_lines"])

			return runCmdImpl(ctx, runner, repoRoot, cmd, argsStr, timeout, maxLines)
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "execution",
			Tags:     []string{"idempotent"},
		},
	}
}
