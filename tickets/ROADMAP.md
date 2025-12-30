# Dodo Product Roadmap

This roadmap organizes the pending tickets from `tickets/todo` into a logical execution order, prioritizing core infrastructure, user control, and capabilities.

## Phase 1: Foundations & Control (v0.1.0 - The "Usable" Release)
*Goal: Remove friction for new users and provide basic control mechanisms.*

1.  **[x] User Config & First-Run Wizard** (`feature_user_config_and_setup.md`)
    *   **Why:** Core dependency. Removes the reliance on environment variables and enables persistence for all future features (provider selection, indexing prefs, etc.).
    *   *Dependencies:* None.

2.  **[x] Stop/Cancel Running Task** (`feature_stop_command.md`)
    *   **Why:** Fundamental safety valve. Users must be able to stop an agent without killing the process.
    *   *Dependencies:* Basic protocol work.

3.  **[x] Slash Command System & Help** (`feature_slash_commands_and_help.md`)
    *   **Why:** Discoverability. Users must know what they can do (`/help`) and how to trigger the new config wizard (`/configure`).
    *   *Dependencies:* None (but integrates with Config).

4.  **[x] Basic Commands & History** (`feature_basic_commands_and_history.md`)
    *   **Why:** Standard CLI expectations. `/exit`, `/clear`, and Up/Down history navigation.
    *   *Dependencies:* Input handling.

## Phase 2: Memory & Context (The "Smart" Release)
*Goal: Make the agent "remember" and respect user boundaries.*

4.  **[x] Persistent Sessions** (`feature_persistent_sessions.md`)
    *   **Why:** Critical UX. The agent shouldn't have amnesia between restarts.
    *   *Dependencies:* User Config (to store session prefs/paths).

5.  **[x] Project Indexing Permissions** (`feature_project_indexing_permission.md`)
    *   **Why:** Privacy/Consent. Prevent the agent from aggressively scanning large/private repos without asking.
    *   *Dependencies:* User Config (to store project-level decisions) and also the indexing apikey since we are using the open ai embedding api.

6.  **[x] Expanded LLM Support (Local/Gemini)** (`feature_llm_support.md`)
    *   **Why:** Cost & Privacy. Enables users to use Ollama/LM Studio.
    *   *Dependencies:* User Config (to save custom base URLs and providers).

## Phase 3: Capabilities & Power (The "Dev" Release)
*Goal: Unlock real development workflows.*

7.  **[P1] Dynamic Permissions (Expanded Allowlist)** (`feature_dynamic_permissions.md`)
    *   **Why:** Unblocks tools like `curl`, `tar`, `grep` safely.
    *   *Dependencies:* User Config (to persist "Always Allow" decisions).

8.  **[P1] Background Execution** (`feature_background_execution.md`)
    *   **Why:** True multitasking. Allows running a server and writing code simultaneously.
    *   *Dependencies:* Dynamic Permissions (users might need to approve background tasks).

    *+ [x] Start dodo from any folder he is invoked in
    

## Phase 4: Polish & Distribution (v0.2.0)
*Goal: Professionalize the experience.*

11. **[P1] Release Pipeline** (`research_release_pipeline.md`)
    *   **Why:** Scale. `npm install -g dodo-ai`.
    *   *Dependencies:* Codebase stability.


9.  **[P2] Command Autocomplete** (`feature_autocomplete.md`)
    *   **Why:** High-end UX polish. Makes slash commands feel "native".
    *   *Dependencies:* Slash Command System.

10. **[P2] Documentation Upgrade** (`documentation_update.md` & `documentation_website.md`)
    *   **Why:** Adoption. Marketing the "Indie/Hacker" spirit.
    *   *Dependencies:* Feature set being relatively stable.


---

## Tracking
- [ ] Phase 1: Foundations
- [ ] Phase 2: Memory & Context
- [ ] Phase 3: Capabilities
- [ ] Phase 4: Polish
