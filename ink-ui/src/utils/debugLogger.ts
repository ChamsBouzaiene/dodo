import fs from 'node:fs';

// Always log startup to verify logging works (regardless of DODO_DEBUG)
try {
    fs.appendFileSync('/tmp/dodo_debug.log', `[${new Date().toISOString()}] debugLogger.ts loaded, DODO_DEBUG=${process.env.DODO_DEBUG}\n`);
} catch {
    // Ignore errors
}
type LogCategory = 'event' | 'command' | 'state' | 'lifecycle' | 'error' | 'traffic';
interface DebugEntry {
    timestamp: string;
    category: LogCategory;
    component: string;
    message: string;
    data?: Record<string, unknown>;
}

/**
 * Structured debug logger for AI-friendly debugging.
 * 
 * Enable with: DODO_DEBUG=true
 * Output: /tmp/dodo_debug.log (JSON lines format)
 */
class DebugLogger {
    private logPath = '/tmp/dodo_debug.log';

    /**
     * Check if debug mode is enabled (checked dynamically each time)
     */
    isEnabled(): boolean {
        return process.env.DODO_DEBUG === 'true';
    }

    /**
     * Write a structured log entry
     */
    async log(entry: Omit<DebugEntry, 'timestamp'>): Promise<void> {
        if (!this.isEnabled()) return;

        const fullEntry: DebugEntry = {
            timestamp: new Date().toISOString(),
            ...entry,
        };

        try {
            await fs.promises.appendFile(this.logPath, JSON.stringify(fullEntry) + '\n');
        } catch {
            // Silently fail - debug logging shouldn't crash the app
        }
    }

    /**
     * Log an event received from the backend
     */
    event(component: string, eventType: string, data?: object): void {
        this.log({
            category: 'event',
            component,
            message: `Received event: ${eventType}`,
            data: data as Record<string, unknown>,
        });
    }

    /**
     * Log a command sent to the backend
     */
    command(component: string, cmdType: string, data?: object): void {
        this.log({
            category: 'command',
            component,
            message: `Sent command: ${cmdType}`,
            data: data as Record<string, unknown>,
        });
    }

    /**
     * Log a state change
     */
    state(component: string, change: string, before?: unknown, after?: unknown): void {
        this.log({
            category: 'state',
            component,
            message: change,
            data: { before, after },
        });
    }

    /**
     * Log component lifecycle events
     */
    lifecycle(component: string, phase: 'mount' | 'unmount' | 'update', details?: string): void {
        this.log({
            category: 'lifecycle',
            component,
            message: `${phase}${details ? `: ${details}` : ''}`,
        });
    }

    /**
     * Log an error with context
     */
    error(component: string, error: Error | string, context?: object): void {
        this.log({
            category: 'error',
            component,
            message: error instanceof Error ? error.message : error,
            data: {
                ...(error instanceof Error ? { stack: error.stack } : {}),
                ...context,
            } as Record<string, unknown>,
        });
    }

    /**
     * Clear the debug log file
     */
    clear(): void {
        try {
            fs.writeFileSync(this.logPath, '');
        } catch {
            // Silently fail
        }
    }
}

// Singleton instance
export const debugLog = new DebugLogger();
