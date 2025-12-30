# Documentation: Update README & Marketing Content

**Type:** Refactor (Docs)
**Priority:** P2
**Effort:** S
**Owners:** TBD
**Target Release:** v0.1.0

## Context
As Dodo prepares for a wider release (v0.1.0), the documentation needs to clearly articulate its unique value proposition, especially compared to well-funded proprietary tools. We need to highlight its "Indie/Hacker" spirit, architectural simplicity (Go + Node), and strong security features.

## Problem Statement
- The current README misses the key differentiator: **Open Source & Simplicity**.
- No visual architecture diagram to explain the "Hybrid" design (Go Engine  + Ink UI).
- Comparison with giants (Gemini/Claude) is missing.

## Goals
- [ ] Add a **Mermaid.js Architecture Diagram**.
- [ ] Update the **Features List** to reflect v0.1.0 status (Sessions, Config, Sandbox).
- [ ] Add a **"Why Dodo?" / Comparison** section highlighting:
    - Local-First & Independent.
    - Built in pure Go (no heavy agent frameworks).
    - Single-developer passion project (Early Stage / "Contributions Welcome").
    - Explicit Comparison with Claude Code / Gemini CLI.

## Requirements
### Content Updates
1.  **Architecture Section**:
    - Add a diagram showing: `CLI <-> Stdio <-> Engine <-> Docker Sandbox`.
2.  **Comparison Table**:
    | Feature | Dodo | Claude Code | Gemini CLI |
    | :--- | :--- | :--- | :--- |
    | **Type** | Open Source (MIT) | Closed / Proprietary | Proprietary |
    | **Execution** | Docker Sandboxed (Safe) | Local Host (Risky) | IDE / Cloud |
    | **Models** | Agnostic (GPT, Claude, etc) | Anthropic Only | Google Only |
    | **Vibe** | Indie / Hacker / Extensible | Corporate Product | Corporate Product |
3.  **V0.1.0 Features**:
    - Persistent Sessions.
    - Global User Config (`~/.dodo/config.json`).
    - Smart "First Run" Indexing permissions.

### Narrative
- Emphasize that Dodo is built by **one person** in highly performant Go, proving that powerful agents don't need complex frameworks like LangChain.
- Call to Action: "This is early stage technology. Join the revolution and contribute."

## Impacted Areas
- `README.md`

## Task Breakdown
1.  Create `tickets/assets/architecture.mermaid` (or embed in README).
2.  Rewrite "Introduction" to be punchier.
3.  Add "Why Dodo?" section.
4.  Update "Features" checklist.
