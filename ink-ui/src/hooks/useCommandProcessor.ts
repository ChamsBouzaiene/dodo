import { useCallback } from "react";
import { debugLog } from "../utils/debugLogger.js";
import { generateDiagnostics, writeDiagnostics } from "../utils/diagnostics.js";

type CommandContext = {
    sessionId?: string;
    status: string;
    isRunning: boolean;
    loadedConfig?: Record<string, string>;
    isSetupRequired: boolean;
    isUpdateMode: boolean;
    clearTurns: () => void;
    cancelRequest: () => void;
    sendCommand: (cmd: any) => void;
    setIsSetupRequired: (val: boolean) => void;
    setIsUpdateMode: (val: boolean) => void;
    setActiveModal: (modal: 'none' | 'help') => void;
    setInput: (val: string) => void;
};

export type UseCommandProcessorReturn = {
    processCommand: (input: string) => boolean; // returns true if handled as command
};

export function useCommandProcessor(ctx: CommandContext): UseCommandProcessorReturn {
    const processCommand = useCallback((input: string): boolean => {
        const trimmed = input.trim();
        if (!trimmed.startsWith("/")) return false;

        if (trimmed === "/help") {
            ctx.setActiveModal('help');
            ctx.setInput("");
            return true;
        }
        if (trimmed === "/exit" || trimmed === "/quit") {
            process.stdout.write("\x1b[?1049l");
            process.exit(0);
            return true; // Technically redundant
        }
        if (trimmed === "/clear") {
            ctx.clearTurns();
            ctx.setInput("");
            return true;
        }
        if (trimmed === '/debug') {
            const diagnostics = generateDiagnostics({
                sessionId: ctx.sessionId,
                status: ctx.status,
                isRunning: ctx.isRunning,
                loadedConfig: ctx.loadedConfig,
                isSetupRequired: ctx.isSetupRequired,
                isUpdateMode: ctx.isUpdateMode,
            });
            const filePath = writeDiagnostics(diagnostics);
            debugLog.command('App', '/debug', { outputPath: filePath });
            ctx.setInput("");
            return true;
        }
        if (trimmed === '/stop') {
            if (ctx.isRunning) {
                ctx.cancelRequest();
                debugLog.command('App', '/stop', { sessionId: ctx.sessionId });
            }
            ctx.setInput("");
            return true;
        }
        if (trimmed === '/configure') {
            ctx.setIsUpdateMode(true);
            ctx.setIsSetupRequired(true);
            ctx.sendCommand({ type: 'get_config' });
            ctx.setInput("");
            return true;
        }

        return false; // Unknown command? Or maybe true if we want to swallow all slash commands?
        // Current App.tsx logic only handles specific ones. 
        // If unknown, it falls through to submitQuery? 
        // Wait, Dodo engine might have slash commands? 
        // No, usually slash commands are client-side.
        // If I return false, it goes to `submitQuery`.
        return false;

    }, [ctx]);

    return { processCommand };
}
