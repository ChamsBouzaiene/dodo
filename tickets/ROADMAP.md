# Dodo Product Roadmap

This roadmap organizes the pending tickets from `tickets/todo` into a logical execution order, prioritizing core infrastructure, user control, and capabilities.

## Phase 1: Capabilities & Power (The "Dev" Release)
*Goal: Unlock real development workflows.*

7.  **[P1] Dynamic Permissions (Expanded Allowlist)** (`feature_dynamic_permissions.md`)
    *   **Why:** Unblocks tools like `curl`, `tar`, `grep` safely.
    *   *Dependencies:* User Config (to persist "Always Allow" decisions).

8.  **[P1] Background Execution** (`feature_background_execution.md`)
    *   **Why:** True multitasking. Allows running a server and writing code simultaneously.
    *   *Dependencies:* Dynamic Permissions (users might need to approve background tasks).

    *+ [x] Start dodo from any folder he is invoked in
    

## Phase 2: Polish & Distribution (v0.2.0)
*Goal: Professionalize the experience.*


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
