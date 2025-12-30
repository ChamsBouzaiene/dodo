/**
 * E2E Test Infrastructure for Dodo CLI
 * 
 * These tests spawn the actual CLI process and interact with it
 * to verify end-to-end functionality.
 * 
 * Prerequisites:
 * 1. Engine binary: go build -o repl ./cmd/repl
 * 2. Valid .env file with API key
 * 3. Run from ink-ui directory with: npm test -- src/tests/e2e/cli.test.ts
 */

import { spawn, ChildProcess } from 'node:child_process';
import { describe, it, expect, afterEach } from 'vitest';
import path from 'node:path';

// Paths - relative to ink-ui
const INK_UI_DIR = path.resolve(__dirname, '../..');
const DODO_ROOT = path.resolve(INK_UI_DIR, '..');
const ENGINE_PATH = path.resolve(DODO_ROOT, 'repl');
const TEST_REPO = path.resolve(DODO_ROOT, 'test-repo');

interface CLIProcess {
    process: ChildProcess;
    stdout: string[];
    stderr: string[];
    send: (input: string) => void;
    waitForStderr: (matcher: string | RegExp, timeout?: number) => Promise<string>;
    close: () => void;
}

/**
 * Spawn the CLI process
 */
function spawnCLI(options: { repo?: string; debug?: boolean } = {}): CLIProcess {
    const repo = options.repo || TEST_REPO;

    // Important: Pass through all environment variables, especially API keys
    const env = {
        ...process.env,
        DODO_DEBUG: options.debug ? 'true' : 'false',
        // Force non-interactive mode
        CI: 'true',
    };

    const proc = spawn('npm', ['run', 'dev', '--', '--repo', repo, '--engine', ENGINE_PATH], {
        cwd: INK_UI_DIR,
        env,
        stdio: ['pipe', 'pipe', 'pipe'],
    });

    const stdout: string[] = [];
    const stderr: string[] = [];

    proc.stdout?.on('data', (data) => {
        stdout.push(data.toString());
    });

    proc.stderr?.on('data', (data) => {
        stderr.push(data.toString());
    });

    return {
        process: proc,
        stdout,
        stderr,
        send: (input: string) => {
            proc.stdin?.write(input);
        },
        waitForStderr: (matcher: string | RegExp, timeout = 10000) => {
            return new Promise((resolve, reject) => {
                const timer = setTimeout(() => {
                    const allStderr = stderr.join('');
                    reject(new Error(`Timeout waiting for stderr matching: ${matcher}\nGot: ${allStderr.slice(-500)}`));
                }, timeout);

                const check = () => {
                    const allStderr = stderr.join('');
                    if (typeof matcher === 'string' ? allStderr.includes(matcher) : matcher.test(allStderr)) {
                        clearTimeout(timer);
                        resolve(allStderr);
                    } else {
                        setTimeout(check, 100);
                    }
                };
                check();
            });
        },
        close: () => {
            proc.kill('SIGTERM');
        },
    };
}

describe('E2E: CLI Startup', () => {
    let cli: CLIProcess | null = null;

    afterEach(() => {
        if (cli) {
            cli.close();
            cli = null;
        }
    });

    it('spawns CLI process without immediate crash', async () => {
        cli = spawnCLI({ debug: true });

        // Wait 2 seconds - if process crashes immediately, it would exit in < 1s
        await new Promise(resolve => setTimeout(resolve, 2000));

        // Check we have some output captured (process actually started)
        const hasOutput = cli.stdout.length > 0 || cli.stderr.length > 0;
        expect(hasOutput).toBe(true);
    }, 10000);

    it('writes to debug log when DODO_DEBUG is enabled', async () => {
        // Clear debug log first
        const fs = await import('node:fs');
        const debugLogPath = '/tmp/dodo_debug.log';
        try { fs.unlinkSync(debugLogPath); } catch { /* ignore */ }

        cli = spawnCLI({ debug: true });

        // Wait for app to load and write debug log
        await new Promise(resolve => setTimeout(resolve, 5000));

        // Check debug log was created
        let logExists = false;
        try {
            fs.accessSync(debugLogPath);
            logExists = true;
        } catch { /* ignore */ }

        expect(logExists).toBe(true);
    }, 15000);
});

// Note: Ink uses raw terminal mode which makes stdout capture unreliable.
// For component testing, use ink-testing-library (see other test files).
// These E2E tests verify process spawning and debug infrastructure.
