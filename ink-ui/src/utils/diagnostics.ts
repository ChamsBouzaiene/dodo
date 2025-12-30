import fs from 'node:fs';

/**
 * Diagnostic snapshot for debugging
 */
interface DiagnosticSnapshot {
    timestamp: string;
    session: {
        id?: string;
        status: string;
        isRunning: boolean;
    };
    config?: Record<string, string>;
    environment: {
        LLM_PROVIDER?: string;
        DODO_DEBUG?: string;
        NODE_ENV?: string;
    };
    componentStates: {
        isSetupRequired: boolean;
        isUpdateMode: boolean;
    };
    recentEvents?: Array<{ timestamp: string; type: string }>;
}

/**
 * Generate a diagnostic snapshot of the current application state
 */
export function generateDiagnostics(params: {
    sessionId?: string;
    status: string;
    isRunning: boolean;
    loadedConfig?: Record<string, string>;
    isSetupRequired: boolean;
    isUpdateMode: boolean;
}): DiagnosticSnapshot {
    return {
        timestamp: new Date().toISOString(),
        session: {
            id: params.sessionId,
            status: params.status,
            isRunning: params.isRunning,
        },
        config: params.loadedConfig,
        environment: {
            LLM_PROVIDER: process.env.LLM_PROVIDER,
            DODO_DEBUG: process.env.DODO_DEBUG,
            NODE_ENV: process.env.NODE_ENV,
        },
        componentStates: {
            isSetupRequired: params.isSetupRequired,
            isUpdateMode: params.isUpdateMode,
        },
    };
}

/**
 * Write diagnostics to a file for AI analysis
 */
export function writeDiagnostics(diagnostics: DiagnosticSnapshot): string {
    const filePath = '/tmp/dodo_diagnostics.json';
    try {
        fs.writeFileSync(filePath, JSON.stringify(diagnostics, null, 2));
        return filePath;
    } catch (error) {
        return `Error writing diagnostics: ${error}`;
    }
}

/**
 * Read recent debug log entries (last N lines)
 */
export function readRecentDebugLogs(count = 50): string[] {
    const logPath = '/tmp/dodo_debug.log';
    try {
        const content = fs.readFileSync(logPath, 'utf-8');
        const lines = content.trim().split('\n');
        return lines.slice(-count);
    } catch {
        return [];
    }
}
