# Research: Release Pipeline & Distribution Strategy

**Type:** Research
**Priority:** P1
**Effort:** M
**Owners:** TBD
**Target Release:** TBD

## Context
Dodo consists of a Go backend (engine) and a Node.js frontend (TUI). Currently, it runs only in a development environment (`go run` + `npm start`). To enable public adoption, we need a streamlined "one-line install" process for users on Mac, Linux, and Windows.

## Problem Statement
- **Installation Friction**: Cloning git repos and compiling dependencies is too hard for end-users.
- **Cross-Platform**: No automated builds for different OSs.
- **Updates**: Users have no way of knowing when bugs are fixed or features added.

## Goals
- [ ] Enable `npm install -g dodo-ai` as the primary installation method.
- [ ] Automate cross-platform Go binary builds using GitHub Actions.
- [ ] Implement an "Update Available" notification on startup.

## Requirements
### Functional
1.  **Distribution Channel**:
    - **NPM**: The main entry point. The package `@dodo/cli` (or similar) will act as a wrapper.
2.  **CI/CD Pipeline**:
    - **Trigger**: Tag push (e.g., `v1.0.0`).
    - **Action**:
        1. Compile Go binaries (Darwin/ARM64, Darwin/AMD64, Linux, Windows).
        2. Upload binaries to **GitHub Releases**.
        3. Publish NPM package to registry.
3.  **Installation Logic**:
    - The NPM package's `postinstall` script (or runtime launcher) detects the OS/Arch.
    - Downloads the corresponding compressed binary from the matched GitHub Release version.
    - Caches it locally (`~/.dodo/bin`).
4.  **Update Notification**:
    - On startup, the CLI fetches the `latest` version tag from GitHub/NPM.
    - If `local < latest`, print: "Update available: v1.0.1 -> v1.0.2. Run 'npm i -g dodo-ai' to update."

### Non-Functional
- **Security**: Verify checksums of downloaded binaries.

## Impacted Areas
- **Infra**: `.github/workflows/`.
- **Repo**: New `cli-wrapper` package (Node.js).

## Proposed Approach (The "Binary Downloader" Pattern)
This is a common pattern used by tools like `esbuild` or `sentry-cli`.

1.  **Backend (`go-releaser`)**:
    - Use [GoReleaser](https://goreleaser.com/) in GitHub Actions.
    - Configured to build `dodo-engine` for all targets.
    - Outputs `tar.gz` files to GitHub Releases.
2.  **Frontend (NPM Package)**:
    - Contains the compiled TS code (Ink UI) + a `bin/dodo` launcher.
    - Launcher:
        - Checks if `dodo-engine` binary is present in `node_modules` or `~/.dodo/bin`.
        - If not, downloads it from GitHub Releases based on `package.json` version.
        - Spawns the binary and connects via Stdio.

## Risks & Mitigations
- **Risk**: GitHub API rate limits for checking updates.
  - **Mitigation**: Cache the "last check" timestamp for 24h.
- **Risk**: `postinstall` scripts failing (permissions/firewalls).
  - **Mitigation**: Perform the download *lazily* on first run, showing a progress bar to the user. This is more robust than `postinstall`.

## Task Breakdown
1.  Configure `goreleaser` for the backend.
2.  Create the Node.js wrapper package structure.
3.  Write the "Lazy Downloader" logic (detect OS -> URL -> Stream Download).
4.  Set up GitHub Actions workflow.
