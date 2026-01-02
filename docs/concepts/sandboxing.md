# Sandboxed Execution

Security is a primary concern when letting an AI agent execute commands on your machine. Dodo uses **Docker containers** to sandbox risky operations, ensuring the agent cannot accidentally (or maliciously) harm your system.

## How it Works

When Dodo needs to run a shell command (e.g., via `run_cmd` or `run_tests`), it checks your `sandbox` configuration.

### Auto Mode (`sandbox_mode: auto`)

This is the default and recommended mode.

1.  **Detection**: Dodo checks if the Docker daemon is reachable.
2.  **Containerization**: If Docker is found, Dodo spins up an ephemeral container tailored to your project language (e.g., `golang:alpine` for Go, `node:alpine` for JS).
3.  **Isolation**:
    -   The command runs **inside the container**, not on your host shell.
    -   Your project directory is mounted as a volume so the agent can see your files.
    -   The container has **no network access** (by default) and restricted capabilities.
4.  **Cleanup**: The container is destroyed immediately after the command finishes.

### Host Mode (`sandbox_mode: host`)

If Docker is not available (or explicitly disabled), Dodo falls back to running commands directly on your host operating system shell (zsh/bash/powershell).

!!! warning "Security Risk"
    In Host Mode, the agent has the same permissions as your user user. It could theoretically run `rm -rf ~` or upload your SSH keys if instructed by a malicious prompt injection. **Use Host Mode only with trusted models and verified prompts.**

## Configuration

You can configure sandboxing per-session or globally in `~/.dodo/config.yaml`.

```yaml
sandbox:
  mode: auto      # Options: auto, docker, host
  timeout: 300    # Kill commands after 5 minutes
  
  # Advanced: Custom container image
  image: my-custom-build-image:latest
```
