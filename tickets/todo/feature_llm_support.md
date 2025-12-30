# Feature: Extended LLM Support & Local Models

**Type:** Feature
**Priority:** P1
**Effort:** M
**Owners:** TBD
**Target Release:** v0.1.0

## Context
Users increasingly want to use "Local LLMs" (via LM Studio, Ollama) for privacy/cost reasons, or alternative massive models like Google Gemini. The current system is hardcoded for OpenAI/Anthropic/Kimi.

## Problem Statement
- **No Generic Config**: Users cannot point Dodo to `http://localhost:1234/v1` (LM Studio).
- **Missing Providers**: Google Gemini is natively supported by our Engine but not exposed in the new Config logic.
- **Hard to Switch**: Changing providers requires editing env vars manually (pre-Config feature).

## Goals
- [ ] Add `generic-openai` provider to factory (supports BaseURL override).
- [ ] Add `gemini` provider to factory.
- [ ] Ensure Config Wizard (Ticket #1) includes these options.
- [ ] Validate "Context Window" limits for local models.

## Requirements
### Functional
1.  **Generic OpenAI Provider**:
    - Should accept `api_base` (BaseURL), `api_key` (dummy if local), `model_name`.
    - Presets for popular local runners: "LM Studio", "Ollama".
2.  **Google Gemini Settings**:
    - `api_key`, `model_name` (default: `gemini-1.5-pro`).
3.  **Config Wizard Updates**:
    - When asking "Select Provider":
        - [ ] OpenAI
        - [ ] Anthropic
        - [ ] Google Gemini
        - [ ] Local / Custom (OpenAI Compatible) → Prompt for Base URL.

### Non-Functional
- **Error Messages**: If a local model is unreachable, show a clear "Connection Refused - is LM Studio running?" error.

## Impacted Areas
- `internal/providers/factory.go`
- `internal/providers/generic.go` (New)
- `internal/providers/gemini.go` (New)

## Task Breakdown
1.  Refactor `factory.go` to support `generic-openai`.
2.  Implement `gemini` client (using Google AI Go SDK or REST).
3.  Update Config structs to hold BaseURL.
