import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { spawn } from "node:child_process";
import net from "node:net";
import React from "react";
import { render } from "ink";
import App from "./ui/app.js";
import { EngineClient } from "./engineClient.js";
import { ErrorBoundary } from "./components/common/ErrorBoundary.js";
import { logger } from "./utils/logger.js";
import { ConversationProvider } from "./contexts/ConversationContext.js";

// Main async function to handle initialization
async function main() {

  const parseArgs = (argv: string[]): Record<string, string | boolean> => {
    const result: Record<string, string | boolean> = {};
    for (let i = 2; i < argv.length; i++) {
      const arg = argv[i];
      if (!arg.startsWith("--")) {
        continue;
      }
      const key = arg.slice(2);
      const next = argv[i + 1];
      if (!next || next.startsWith("--")) {
        result[key] = true;
      } else {
        result[key] = next;
        i++;
      }
    }
    return result;
  };

  const args = parseArgs(process.argv);

  const repoPath = path.resolve(
    typeof args.repo === "string" ? (args.repo as string) : process.cwd()
  );
  const engineCwd = path.resolve(
    typeof args["engine-cwd"] === "string"
      ? (args["engine-cwd"] as string)
      : path.resolve(process.cwd(), "..")
  );
  const enginePath =
    typeof args.engine === "string" ? (args.engine as string) : undefined;
  const engineAddr =
    typeof args["engine-addr"] === "string" ? (args["engine-addr"] as string) : undefined;

  const requestedSessionId =
    typeof args["session-id"] === "string" ? (args["session-id"] as string) : undefined;

  const logFile =
    typeof args["log-file"] === "string" ? (args["log-file"] as string) : undefined;
  if (logFile) {
    logger.log(`Logging started. Repository: ${repoPath}`);
  }

  // Validate repository path
  if (!fs.existsSync(repoPath)) {
    console.error("\n❌ Error: Repository path does not exist");
    console.error(`   Path: ${repoPath}\n`);
    console.error("Usage:");
    console.error("  npm run dev -- --repo <path-to-your-repo> --engine <path-to-dodo-binary> [--log-file <path>]\n");
    console.error("Example:");
    console.error("  npm run dev -- --repo ../../dodo_tasks/my-project --engine ../repl --log-file debug.log\n");
    process.exit(1);
  }

  const repoStat = fs.statSync(repoPath);
  if (!repoStat.isDirectory()) {
    console.error("\n❌ Error: Repository path is not a directory");
    console.error(`   Path: ${repoPath}\n`);
    console.error("Please provide a valid directory path.\n");
    process.exit(1);
  }

  let client: EngineClient;
  let childProcess: ReturnType<typeof spawn> | undefined;
  let engineExited: { code: number | null, signal: NodeJS.Signals | null } | undefined;

  // --- E2E MODE ---
  if (engineAddr) {
    try {
      fs.writeFileSync("e2e_debug.log", `[${new Date().toISOString()}] Index.tsx starting. EngineAddr: ${engineAddr}\n`);
    } catch (_) { }

    const [host, portStr] = engineAddr.split(":");
    const port = parseInt(portStr, 10);
    const socket = net.createConnection({ host, port });

    socket.on('connect', () => {
      try { fs.appendFileSync("e2e_debug.log", `[${new Date().toISOString()}] Socket connected to ${host}:${port}!\n`); } catch (_) { }
    });

    socket.on('error', (err) => {
      try { fs.appendFileSync("e2e_debug.log", `[${new Date().toISOString()}] Socket error: ${err.message}\n`); } catch (_) { }
    });

    socket.on('close', () => {
      try { fs.appendFileSync("e2e_debug.log", `[${new Date().toISOString()}] Socket closed.\n`); } catch (_) { }
    });

    // Use the socket for both input and output
    client = new EngineClient(socket, socket);
  }
  // --- PRODUCTION MODE ---
  else {
    const resolveEngineCommand = () => {
      const baseArgs = ["engine", "--stdio", "--repo", repoPath];

      // 1. Explicit --engine flag (highest priority)
      if (enginePath) {
        const resolvedEnginePath = path.resolve(enginePath);
        if (!fs.existsSync(resolvedEnginePath)) {
          console.error("\n❌ Error: Engine binary not found");
          console.error(`   Path: ${resolvedEnginePath}\n`);
          console.error("Please build the engine first:");
          console.error("  cd <dodo-repo> && go build -o dodo ./cmd/repl\n");
          process.exit(1);
        }
        return { command: resolvedEnginePath, args: baseArgs, cwd: path.dirname(resolvedEnginePath) };
      }

      // 2. DODO_ENGINE_PATH environment variable
      const envEnginePath = process.env.DODO_ENGINE_PATH;
      if (envEnginePath && fs.existsSync(envEnginePath)) {
        return { command: envEnginePath, args: baseArgs, cwd: path.dirname(envEnginePath) };
      }

      // 3. Look for 'dodo' in common locations
      const homeDir = process.env.HOME || process.env.USERPROFILE || "";
      const searchPaths = [
        path.join(homeDir, ".local", "bin", "dodo"),
        path.join(homeDir, "bin", "dodo"),
        "/usr/local/bin/dodo",
        path.join(engineCwd, "dodo"),
      ];

      for (const candidate of searchPaths) {
        if (fs.existsSync(candidate)) {
          return { command: candidate, args: baseArgs, cwd: path.dirname(candidate) };
        }
      }

      // 4. Fallback: go run (requires DODO_DIR or engineCwd pointing to dodo source)
      const dodoDir = process.env.DODO_DIR || engineCwd;
      if (fs.existsSync(path.join(dodoDir, "cmd", "repl"))) {
        return { command: "go", args: ["run", "./cmd/repl", ...baseArgs], cwd: dodoDir };
      }

      console.error("\n❌ Error: Could not find dodo engine");
      console.error("Options:");
      console.error("  1. Set DODO_ENGINE_PATH=/path/to/dodo binary");
      console.error("  2. Set DODO_DIR=/path/to/dodo/source (for go run)");
      console.error("  3. Install dodo to ~/.local/bin/dodo\n");
      process.exit(1);
    };

    const { command, args: engineArgs, cwd: engineWorkDir } = resolveEngineCommand();
    try {
      fs.appendFileSync('/tmp/dodo_ui_debug.log', `${new Date().toISOString()} Spawning engine: ${command} ${engineArgs.join(' ')} (cwd: ${engineWorkDir})\n`);
    } catch (_) { }

    childProcess = spawn(command, engineArgs, {
      cwd: engineWorkDir,
      stdio: ["pipe", "pipe", "pipe"],
    });

    childProcess.on('error', (err) => {
      try {
        fs.appendFileSync('/tmp/dodo_ui_debug.log', `${new Date().toISOString()} Engine spawn error: ${err.message}\n`);
      } catch (_) { }
    });

    if (!childProcess.stdin || !childProcess.stdout) {
      console.error("Failed to spawn engine: stdin/stdout missing");
      process.exit(1);
    }

    childProcess.on('exit', (code, signal) => {
      engineExited = { code, signal };
    });

    if (childProcess.stderr) {
      childProcess.stderr.on('data', (data) => {
        const line = data.toString();
        try {
          fs.appendFileSync('/tmp/dodo_backend_stderr.log', line);
        } catch (_) { }
        logger.log(`[BACKEND] ${line.trim()}`);
      });
    }

    client = new EngineClient(childProcess.stdin, childProcess.stdout);
  }

  const displayCommand = childProcess ? "Local Engine" : `Remote Engine (${engineAddr})`;

  // Run the app (DodoApp handles the connection)
  const inkApp = render(
    <ErrorBoundary>
      <ConversationProvider>
        <App
          client={client}
          repoPath={repoPath}
          requestedSessionId={requestedSessionId}
          engineCommand={displayCommand}
        />
      </ConversationProvider>
    </ErrorBoundary>,
    {
      stdin: process.stdin,
      patchConsole: true
    }
  );

  // Cleanup function to restore terminal and kill engine
  const cleanup = (code: number = 0) => {
    // Switch back to normal buffer before anything else
    process.stdout.write("\x1b[?1049l");

    try {
      client.close();
      if (childProcess && !childProcess.killed) {
        childProcess.kill("SIGINT");
      }
      inkApp.unmount();
    } catch (err) {
      // Ignore cleanup errors
    } finally {
      // Small timeout to allow stdout to flush
      setTimeout(() => {
        process.exit(code);
      }, 100);
    }
  };

  // If using local child process, cleanup on its exit
  if (childProcess) {
    childProcess.on("exit", (code, signal) => {
      cleanup(code ?? (signal ? 1 : 0));
    });
  }

  process.on("SIGINT", () => cleanup(0));
  process.on("SIGTERM", () => cleanup(0));
}

// Run main and catch any errors
main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
