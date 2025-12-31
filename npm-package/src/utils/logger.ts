import fs from 'node:fs';
import path from 'node:path';

const LOG_FILE = path.resolve(process.cwd(), 'ui_debug.log');

// Helper to safely stringify objects with circular references
function safeStringify(obj: any, space: number = 2): string {
    const seen = new WeakSet();
    return JSON.stringify(obj, (key, value) => {
        if (typeof value === "object" && value !== null) {
            if (seen.has(value)) {
                return "[Circular]";
            }
            seen.add(value);
        }
        return value;
    }, space);
}

const STATE_LOG_FILE = path.resolve(process.cwd(), 'ui_state.log');

// Throttling state
let lastStateLogTime = 0;
const STATE_LOG_THROTTLE_MS = 1000;

const logQueue: { file: string; message: string }[] = [];
let isProcessing = false;

async function processQueue() {
    if (isProcessing || logQueue.length === 0) return;
    isProcessing = true;
    while (logQueue.length > 0) {
        const item = logQueue.shift();
        if (item) {
            try {
                await fs.promises.appendFile(item.file, item.message);
            } catch (e) {
                // Last resort fallback to stderr if even logging fails
            }
        }
    }
    isProcessing = false;
}

function queueLog(file: string, message: string) {
    logQueue.push({ file, message });
    void processQueue();
}

export const logger = {
    log: (message: string, data?: any) => {
        const timestamp = new Date().toISOString();
        const logMessage = `[${timestamp}] [INFO] ${message} ${data ? safeStringify(data, 0) : ''}\n`;
        queueLog(LOG_FILE, logMessage);
    },
    error: (message: string, error?: any) => {
        const timestamp = new Date().toISOString();
        let errorDetails = '';

        if (error instanceof Error) {
            errorDetails = safeStringify({
                message: error.message,
                stack: error.stack,
                name: error.name
            });
        } else if (typeof error === 'object' && error !== null) {
            try {
                errorDetails = safeStringify(error);
                if (errorDetails === '{}') {
                    const props = Object.getOwnPropertyNames(error).reduce((acc, key) => {
                        acc[key] = (error as any)[key];
                        return acc;
                    }, {} as any);
                    errorDetails = safeStringify(props);
                }
            } catch (e) {
                errorDetails = String(error);
            }
        } else {
            errorDetails = String(error);
        }

        const errorMessage = `[${timestamp}] [ERROR] ${message} ${errorDetails}\n`;
        queueLog(LOG_FILE, errorMessage);
    },
    clear: () => {
        try {
            fs.writeFileSync(LOG_FILE, '');
            fs.writeFileSync(STATE_LOG_FILE, '');
        } catch (_) { }
    },
    snapshot: (name: string, data: any) => {
        const timestamp = new Date().toISOString();
        const snapshotData = safeStringify(data, 2);
        const logMessage = `\n[${timestamp}] [SNAPSHOT] === ${name} ===\n${snapshotData}\n=====================================\n`;
        queueLog(LOG_FILE, logMessage);
    },
    state: (message: string, data?: any, immediate: boolean = false) => {
        const now = Date.now();
        if (!immediate && now - lastStateLogTime < STATE_LOG_THROTTLE_MS) {
            return;
        }

        const timestamp = new Date().toISOString();
        const logMessage = `[${timestamp}] [STATE] ${message} ${data ? safeStringify(data, 0) : ''}\n`;
        queueLog(STATE_LOG_FILE, logMessage);

        if (!immediate) {
            lastStateLogTime = now;
        }
    }
};
