# Feature: User Configuration & First-Run Setup Wizard

**Type:** Feature
**Priority:** P0
**Effort:** M
**Owners:** TBD
**Target Release:** TBD

## Context
Currently, Dodo relies on environment variables (like `OPENAI_API_KEY`) loaded from the shell or local `.env` files. This creates friction for new users and lacks a unified way to persist preferences like the default model or indexing settings. We want to move towards a standard, persistent configuration model similar to CLI tools like `gh` or `aws`.

## Problem Statement
- No persistent storage for user preferences (LLM provider, API keys, indexing choices).
- New users face a "blank screen" or crash if keys are missing; no onboarding flow.
- Changing the model requires restarting with new env vars.
- No discoverable way to re-configure the agent while running.

## Goals
- [ ] Implement a cross-platform configuration file (XDG standard).
- [ ] Create an interactive "First Run" wizard to capture keys and preferences.
- [ ] Support OpenAI, Anthropic, and Kimi K2 providers.
- [ ] Add `/configure` and `/help` slash commands within the agent.
- [ ] Ensure local project `.env` files still take precedence (backward compatibility).

## Non-Goals
- [ ] OS Keychain integration (plain text JSON is acceptable for V1).
- [ ] GUI settings panel (text-based wizard only).

## Requirements
### Functional
1.  **Config Storage**:
    - **Path**: Follow OS standards (`os.UserConfigDir` in Go / `appData` in Node).
        - Linux/Mac: `~/.config/dodo/config.json`
        - Windows: `%APPDATA%\dodo\config.json`
    - **Schema**:
        ```json
        {
          "llm_provider": "openai", // or "anthropic", "kimi"
          "api_key": "sk-...",
          "model": "gpt-4o",
          "embedding_key": "...",
          "auto_index": true
        }
        ```
2.  **Setup Wizard**:
    - **Trigger**: Runs automatically if the config file is missing.
    - **Flow**:
        1.  "Select LLM Provider": [OpenAI, Anthropic, Kimi K2]
        2.  "Enter API Key" (Hidden input)
        3.  "Select Default Model" (Context-dependent options)
        4.  "Enable Auto-Indexing for Projects?" (Y/n)
    - **Action**: Saves `config.json` to the correct OS path.
3.  **Runtime Integration**:
    - The backend (`repl`) must load this config on startup.
    - **Precedence Order** (Highest to Lowest):
        1.  Local `.env` file / Shell Environment Variables
        2.  Global `config.json`
4.  **Slash Commands**:
    - `/configure`: Re-opens the setup wizard (or a mini-version of it) inside the chat loop.
    - `/help`: Displays help text, including current configuration status (e.g., "Using gpt-4o via OpenAI").

### Non-Functional
- **Cross-Platform**: Must verify path logic works on Windows.
- **Security**: File permissions should be set to 0600 (read/write by user only) on Create.

## Impacted Areas
- **Backend (`cmd/repl`)**:
    - `internal/config`: New package for loading/saving config.
    - `env.go`: Updated to merge config with environment.
- **Frontend (`ink-ui`)**:
    - `index.tsx`: Startup check logic.
    - `components/Wizard.tsx`: New component for the setup flow.
- **Agent Logic**:
    - Integration of `/configure` command to trigger UI state change.

## Breaking Change Assessment
**Classification:** Non-breaking / Additive.
**Reasoning:**
- If a user has an existing workflow using `.env`, it takes precedence, so their setup won't break.
- The wizard only triggers if no config exists (and effectively if no env vars are set, or depending on implementation choice, purely on config file absence).

## Proposed Approach
1.  **Backend Config Package**:
    - Create `internal/config/manager.go` using `os.UserConfigDir()`.
    - Implement `Load()` and `Save()`.
2.  **Frontend Wizard**:
    - On `ink-ui` start, check if config exists (via specific IPC message or direct file check if running locally).
    - If missing, render `<SetupWizard />` instead of `<App />`.
    - On complete, write file and switch to `<App />`.
3.  **Command Handling**:
    - Add `/configure` to the slash command parser. `client` receives this and notifies UI to switch to "Config Mode".

## Testing Strategy
### Unit
- Test `config.Load()` with various JSON inputs.
- Test Precedence logic (Env var vs Config).

### Integration
- Simulate a "fresh install" by renaming existing config, verifying Wizard appears.
- Verify generated `config.json` is valid JSON and contains correct keys.

## Risks & Mitigations
- **Risk**: Windows path issues.
  - **Mitigation**: Use `path/filepath` and `os.UserConfigDir` extensively. Avoid hardcoded `/`.
- **Risk**: User types wrong API key.
  - **Mitigation**: Add a "Test Connection" step in the wizard before saving.

## Task Breakdown
1.  Create `internal/config` package in Backend.
2.  Implement `SetupWizard` component in Frontend.
3.  Wire up Frontend-to-Backend config saving.
4.  Update Backend startup implementation to read config.
5.  Implement `/configure` and `/help` commands.

## Update (2025-12-14)
**Status**: Complete ✅
**Assignee**: @antigravity
**Effort**: Medium (2-3 days)

### Implementation Details
- **Config Storage**: Uses XDG-compliant path (`~/.config/dodo/config.json` on macOS/Linux).
- **Format**: Simple JSON key-value pairs.
- **Precedence**: Environment variables > Config File.
- **Wizard**: Interactive CLI wizard in `ink-ui` using `SetupWizard` component.
- **Integration**: Backend emits `setup_required` event if config is missing; Frontend renders wizard.
- **Protocol**: Added `save_config` command and `setup_complete` status.
- **Runtime**: Config is applied immediately to the running process upon save.
