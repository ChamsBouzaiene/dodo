import { useState, useRef, useEffect, useCallback } from "react";
import type { EngineClient } from "../engineClient.js";
import { useEngineEvents } from "./useEngineEvents.js";
import { logger } from "../utils/logger.js";
import { debugLog } from "../utils/debugLogger.js";
import type { UiStatus, TokenUsage } from "../types.js";
import { useApp } from "ink";

export function useSessionLifecycle(
    client: EngineClient,
    repoPath: string,
    requestedSessionId?: string,
    engineExited?: { code: number | null; signal: NodeJS.Signals | null },
    skipConnection: boolean = false
) {
    const { exit } = useApp();
    const [sessionId, setSessionId] = useState<string>();
    const [status, setStatus] = useState<UiStatus>(engineExited ? "disconnected" : "connecting");
    const [infoMessage, setInfoMessage] = useState("Connecting to engine...");
    const [error, setError] = useState<string>();
    const [tokenUsage, setTokenUsage] = useState<TokenUsage>();
    const [loadedConfig, setLoadedConfig] = useState<Record<string, string>>();
    const [isSetupRequired, setIsSetupRequired] = useState(false);
    const [isProjectPermissionRequired, setIsProjectPermissionRequired] = useState(false);
    const [pendingRepoRoot, setPendingRepoRoot] = useState<string | null>(null);

    // Track current status to prevent duplicate updates
    const statusRef = useRef<UiStatus>(status);

    // Keep ref in sync with state
    useEffect(() => {
        statusRef.current = status;
    }, [status]);

    // Initial connection logic
    useEffect(() => {
        if (skipConnection) {
            return;
        }

        if (engineExited) {
            statusRef.current = "disconnected";
            setStatus("disconnected");
            setInfoMessage("Engine process exited");
            return;
        }

        client
            .startSession({ repoRoot: repoPath, sessionId: requestedSessionId })
            .catch((err) => {
                setError(err instanceof Error ? err.message : String(err));
                statusRef.current = "error";
                setStatus("error");
            });
    }, [client, repoPath, requestedSessionId, engineExited, skipConnection]);

    // Handle engine exit
    useEffect(() => {
        if (engineExited) {
            setInfoMessage(
                `Engine exited${engineExited.code !== null ? ` (code ${engineExited.code})` : ""}`
            );
            const timer = setTimeout(() => {
                process.stdout.write("\x1b[?1049l");
                exit();
            }, 500);
            return () => clearTimeout(timer);
        }
    }, [engineExited, exit]);

    useEngineEvents(client, {
        onStatusChange: useCallback((newStatus: UiStatus, message: string) => {
            if (statusRef.current !== newStatus) {
                statusRef.current = newStatus;
                setStatus(newStatus);
                setInfoMessage(message);
            } else {
                setInfoMessage((prevMsg) => (prevMsg !== message ? message : prevMsg));
            }
        }, []),

        onSessionReady: useCallback((id: string) => {
            setSessionId(id);
            statusRef.current = "ready";
            setStatus("ready");
            setInfoMessage(`Session ready (${id.slice(0, 8)})`);
            setError(undefined);
            // Fetch config to populate model display in footer
            client.sendCommand({ type: 'get_config' });
        }, [client]),

        onError: useCallback((message: string) => {
            setError(message);
            statusRef.current = "error";
            setStatus("error");
        }, []),

        onTokenUsage: useCallback((usage: { prompt: number; limit: number; total: number }) => {
            setTokenUsage({
                used: usage.prompt,
                total: usage.limit,
                percentage: usage.limit > 0 ? (usage.prompt / usage.limit) * 100 : 0,
                sessionTotal: usage.total,
            });
        }, []),

        onSetupRequired: useCallback(() => {
            setIsSetupRequired(true);
        }, []),

        onProjectPermissionRequired: useCallback((repoRoot: string) => {
            setIsProjectPermissionRequired(true);
            setPendingRepoRoot(repoRoot);
        }, []),

        onConfigLoaded: useCallback((config: Record<string, string>) => {
            setLoadedConfig(config);
        }, []),

        onConfigReloaded: useCallback((provider: string, modelName: string) => {
            statusRef.current = "ready";
            setStatus("ready");
            setInfoMessage(`Config reloaded: ${provider} (${modelName})`);
            setLoadedConfig(prev => prev ? {
                ...prev,
                llm_provider: provider,
                model: modelName
            } : { llm_provider: provider, model: modelName });
            debugLog.event('useSessionLifecycle', 'config_reloaded', { provider, modelName });
        }, []),

        onCancelled: useCallback((reason?: string) => {
            logger.log(`[useSessionLifecycle] onCancelled received. reason=${reason}`);
            statusRef.current = "ready";
            setStatus("ready");
            setInfoMessage(reason || "Task cancelled");
        }, []),
    });

    const reloadSession = useCallback(() => {
        debugLog.command('useSessionLifecycle', 'reloadSession', { sessionId });

        if (!sessionId) {
            debugLog.error('useSessionLifecycle', 'No sessionId, cannot reload');
            return;
        }

        debugLog.command('useSessionLifecycle', 'reload_config', { sessionId });

        client.sendCommand({
            type: "reload_config",
            session_id: sessionId
        });

        setInfoMessage("Configuration reloaded!");
        statusRef.current = "ready";
        setStatus("ready");

        debugLog.state('useSessionLifecycle', 'reloadSession completed', undefined, { status: 'ready' });
    }, [sessionId, client]);

    return {
        sessionId,
        status,
        statusRef,
        infoMessage,
        setInfoMessage,
        setStatus,
        error,
        setError,
        tokenUsage,
        loadedConfig,
        isSetupRequired,
        setIsSetupRequired,
        isProjectPermissionRequired,
        setIsProjectPermissionRequired,
        pendingRepoRoot,
        reloadSession
    };
}
