
import * as pty from 'node-pty';
import path from 'path';

export class CliWrapper {
    private ptyProcess: pty.IPty | null = null;
    private buffer = '';

    constructor(
        private options: {
            cols?: number,
            rows?: number,
            env?: NodeJS.ProcessEnv
        } = {}
    ) { }

    start(args: string[] = []): void {
        const indexArgs = ['src/index.tsx', ...args];
        // We use tsx to run the TS file directly
        const cmd = 'npx';
        const cmdArgs = ['tsx', ...indexArgs];

        this.ptyProcess = pty.spawn(cmd, cmdArgs, {
            name: 'xterm-color',
            cols: this.options.cols || 80,
            rows: this.options.rows || 30,
            cwd: process.cwd(),
            env: { ...process.env, ...this.options.env }
        });

        this.ptyProcess.onData((data) => {
            this.buffer += data;
        });
    }

    write(data: string): void {
        if (!this.ptyProcess) throw new Error('Process not started');
        this.ptyProcess.write(data);
    }

    async waitForOutput(pattern: string | RegExp, timeoutMs = 5000): Promise<void> {
        const startTime = Date.now();

        while (Date.now() - startTime < timeoutMs) {
            if (typeof pattern === 'string') {
                if (this.buffer.includes(pattern)) return;
            } else {
                if (pattern.test(this.buffer)) return;
            }
            await new Promise(r => setTimeout(r, 100));
        }

        throw new Error(`Timeout waiting for pattern: ${pattern}\nBuffer content:\n${this.buffer}`);
    }

    getBuffer(): string {
        return this.buffer;
    }

    clearBuffer(): void {
        this.buffer = '';
    }

    stop(): void {
        if (this.ptyProcess) {
            this.ptyProcess.kill();
            this.ptyProcess = null;
        }
    }

    isRunning(): boolean {
        return !!this.ptyProcess;
    }
}
