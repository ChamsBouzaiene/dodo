# Configuration

Dodo is designed to be flexible. You can configure it to use almost any LLM provider and customize safety settings.

## The Setup Wizard (Recommended)

The easiest way to configure Dodo is using the built-in interactive wizard.

1.  Start Dodo:
    ```bash
    dodo
    ```
2.  Type `/conf` in the input bar (or select **Configure** from the main menu starting screen).

The wizard will guide you through:
-   **Selecting a Provider**: OpenAI, Anthropic, Gemini, or Local.
-   **Entering API Keys**: Securely stored on your device.
-   **Sandbox Settings**: Enabling Docker for safe execution.

Configuration is saved to `~/.dodo/config.yaml`.

---

## Manual Configuration

You can also create or edit the config file manually at `~/.dodo/config.yaml`.

### LLM Providers

=== "OpenAI"

    ```yaml
    llm:
      provider: openai
      api_key: sk-your-api-key
      model: gpt-4o
    ```

=== "Anthropic"

    ```yaml
    llm:
      provider: anthropic
      api_key: sk-ant-your-key
      model: claude-3-5-sonnet-20240620
    ```

=== "Local / Ollama"

    You can use any OpenAI-compatible endpoint (like Ollama or LM Studio).

    ```yaml
    llm:
      provider: openai
      base_url: http://localhost:11434/v1
      api_key: ollama  # Value doesn't matter for Ollama, but must be present
      model: llama3
    ```

## Safety & Sandboxing

Dodo strongly recommends using **Docker** to sandbox the agent's execution. This prevents the agent from accidentally (or intentionally) modifying files outside your project or installing unwanted software.

To enable sandboxing:

1.  Ensure [Docker Desktop](https://www.docker.com/products/docker-desktop/) is installed and running.
2.  Set `sandbox_mode: auto` in your config.

```yaml
sandbox:
  mode: auto  # Use "docker" to force fail if docker is missing
```

!!! warning "Without Sandbox"
    If you disable sandboxing (`mode: host`), the agent runs commands properly on your machine shell. **Only use this with trusted models.**
