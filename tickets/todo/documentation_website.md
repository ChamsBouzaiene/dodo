# Feature: Documentation Website (MkDocs Material)

**Type:** Feature (Docs)
**Priority:** P2
**Effort:** M
**Owners:** TBD
**Target Release:** v0.1.0

## Context
Dodo needs a professional, public-facing documentation site to explain its architecture, features, and usage. A static website is superior to a README for SEO, navigation, and presenting rich media (GIFs, diagrams). We aim for the aesthetic of [swe-agent.com](https://swe-agent.com).

## Problem Statement
- The single README.md is becoming overcrowded.
- No dedicated space for architectural deep dives or tutorials.
- Lack of a visual "Hero" page to showcase the CLI.

## Goals
- [ ] Create a static documentation site using **MkDocs Material**.
- [ ] Host it on **GitHub Pages**.
- [ ] Include a "Hero" section with the Dodo Value Prop and CLI Demo GIF.
- [ ] Document Architecture, Installation, and Usage in separate sections.

## Requirements
### Functional
1.  **Framework**: MkDocs with the Material theme (`mkdocs-material`).
2.  **Navigation Structure**:
    - **Home**: Hero text ("Dodo – an open coding agent built in Go"), Demo GIF, Key Features.
    - **Getting Started**: Installation (npm/go), Configuration (API Keys).
    - **Architecture**:
        - Diagram (Mermaid).
        - Explanation of Engine (Go) vs UI (Ink).
        - Sandbox details.
    - **Usage**:
        - Interactive Mode vs CLI Mode.
        - Slash Commands.
        - Tips & Tricks.
    - **Reference**: Protocol specs, Config options.
3.  **Visuals**:
    - Dark mode support (default).
    - Mermaid diagram support.
    - CLI GIF (placeholder for now, provided later).

### Non-Functional
- **Deployment**: Automated via GitHub Actions on push to `main`.
- **Search**: Fast, client-side search.

## Impacted Areas
- `docs/` directory (content).
- `.github/workflows/` (deployment pipeline).
- `mkdocs.yml` (config).

## Task Breakdown
1.  **Setup**: Initialize MkDocs project and configure `mkdocs.yml` with Material theme.
2.  **Content**: Migrate content from README and expand into separate pages.
3.  **Pipeline**: Create `.github/workflows/deploy-docs.yml`.
4.  **Polish**: Add custom CSS if needed to match the "Dodo" brand (e.g., logo).

## Risks & Mitigations
- **Risk**: Documentation drifting from code.
  - **Mitigation**: Add a "Check Docs" step to PR reviews for features.

## Open Questions
- Custom domain (e.g. `dodoai.com`) will be added later.
