# Feature: Per-Project Indexing Permissions & Configuration

**Type:** Feature
**Priority:** P1
**Effort:** M
**Owners:** TBD
**Target Release:** TBD

## Context
Dodo currently attempts to auto-index codebases if an API key is present. This can be invasive for large or sensitive repos. We want to give users explicit control over RAG (Retrieval-Augmented Generation) at the project level and educate them about custom rules.

## Problem Statement
- Indexing starts without explicit project-level consent.
- Users don't know how to customize agent behavior (Rules).
- No persistent per-project configuration file.
- the indexing apikey since we are using the open ai embedding api.

## Goals
- [ ] Implement a "First Run" prompt for new projects: "Enable Indexing?".
- [ ] Persist this decision in a local `.dodo/config.json` file.
- [ ] Support custom agent rules via `.dodo/rules` (similar to `.cursorrules`).
- [ ] Educate users about the Rules feature upon enabling indexing.

## Non-Goals
- [ ] Global whitelist of projects (decisions are stored locally in the repo).

## Requirements
### Functional
1.  **Project Configuration File**:
    - Path: `repo_root/.dodo/config.json`
    - Content: `{ "indexing_enabled": boolean }`
2.  **Startup Logic**:
    - **Step 1**: Check if `.dodo/config.json` exists.
    - **Step 2 (If Missing)**:
        - Prompt user: "Would you like to enable semantic indexing for this project? (Y/n)"
        - If **Yes**:
            - Write `{"indexing_enabled": true}` to `.dodo/config.json`.
            - Init Indexer.
            - Display Message: "Indexing started. You can add custom behavior rules in `.dodo/rules`."
        - If **No**:
            - Write `{"indexing_enabled": false}` to `.dodo/config.json`.
            - Skip Indexer.
    - **Step 3 (If Present)**:
        - Read `indexing_enabled`. If true -> Init Indexer. If false -> Skip.
3.  **Global Override**:
    - Even if Global Config says "Auto Index = Always", we **Respect the Local Config**. If local config is *missing*, we still ask (or fallback to global preference, but user requested explicit "no" to global overriding local interactions).
    - Refined Logic: Global config sets the *default* for the prompt, but the prompt should still appear for new projects to confirm intent.

### Non-Functional
- **UX**: The prompt should be non-blocking for basic features, but blocking for RAG features. Ideally a startup wizard step.

## Impacted Areas
- **Backend (`cmd/repl`)**: `env.go` startup sequence.
- **Frontend (`ink-ui`)**: New "Project Setup" state in `App.tsx`.

## Breaking Change Assessment
**Classification:** Behavior Change.
**Reasoning:** Existing projects without `.dodo/config.json` will prompt the user on the next run. This is acceptable/desirable.

## Proposed Approach
1.  **Backend**:
    - `indexer.LoadProjectConfig(repoRoot)`
    - If no config, send `event: "request_project_permission"` to UI.
    - Wait for `stdin` response (e.g. `{"type": "project_permission", "enabled": true}`).
    - Save to disk.
2.  **Frontend**:
    - Handle `request_project_permission` event.
    - Show `Box` with "Enable Indexing?" query.
    - Send response.

## Data & APIs
- **`.dodo/rules`**:
    - Plain text markdown file.
    - Loaded into the System Prompt context if it exists.

## Testing Strategy
### Integration
- Run on a clean directory. Verify prompt appears.
- Select "No". Restart. Verify prompt does NOT appear and indexing is skipped.
- Select "Yes". Restart. Verify indexing starts automatically.

## Risks & Mitigations
- **Risk**: `.dodo` folder clutter.
  - **Mitigation**: Add `.dodo` to `.gitignore` automatically if not present? No, user might want to share team config. Let user decide.

## Task Breakdown
1.  Implement `internal/project/config.go`.
2.  Update `repl/main.go` to check config before starting Agent.
3.  Implement Frontend Prompt UI.
4.  Add `.dodo/rules` loading to `prompts/system.go`.
