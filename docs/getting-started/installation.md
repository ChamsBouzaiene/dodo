# Installation

Dodo consists of two parts:
1.  **The CLI (NPM)**: A React-based terminal UI.
2.  **The Engine (Go)**: A high-performance agent runtime (automatically managed by the CLI).

## Recommended: NPM Install

The easiest way to install Dodo is via `npm`. This installs the CLI wrapper which will automatically download the correct engine binary for your OS (macOS, Linux, or Windows).

```bash
npm install -g dodo-ai
```

Verify the installation:

```bash
dodo --version
```

!!! success "Ready to go!"
    You can now run `dodo` in any directory. The first time you run it, it will download the ~15MB engine binary.

---

## Alternative: Build from Source

If you prefer to build the engine yourself (e.g., for contributing), you'll need **Go 1.21+** and **Node.js 18+**.

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/ChamsBouzaiene/dodo.git
    cd dodo
    ```

2.  **Build the Engine:**
    ```bash
    go build -o repl ./cmd/repl
    ```

3.  **Run the UI:**
    ```bash
    cd ink-ui
    npm install
    npm run dev -- --engine ../repl
    ```
