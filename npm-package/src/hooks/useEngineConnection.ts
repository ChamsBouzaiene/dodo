import { useState, useRef, useEffect, useCallback, useMemo } from "react";
import type { EngineClient } from "../engineClient.js";
import { useConversation } from "./useConversation.js";
import { logger } from "../utils/logger.js";
import { debugLog } from "../utils/debugLogger.js";
import { useApp } from "ink";
import { useSessionLifecycle } from "./useSessionLifecycle.js";
import { useResponseStream } from "./useResponseStream.js";
import { useToolActivities } from "./useToolActivities.js";
import { useEngineEvents } from "./useEngineEvents.js";

export function useEngineConnection(
    client: EngineClient,
    repoPath: string,
    requestedSessionId?: string,
    engineExited?: { code: number | null; signal: NodeJS.Signals | null },
    skipConnection: boolean = false
) {
    const { exit } = useApp();
    const [errorCount, setErrorCount] = useState(0);

    const {
        turns,
        pushTurn,
        appendAssistantContent,
        markLastTurnDone,
        addActivity,
        updateActivity,
        toggleTurnCollapsed,
        addTimelineStep,
        updateTimelineStep,
        getCurrentTurnId,
        appendLogLine,
        appendToolOutput,
        addContextEvent,
        clearTurns,
    } = useConversation();

    // 1. Session Lifecycle Management
    const {
        sessionId,
        status,
        statusRef,
        infoMessage,
        setInfoMessage,
        setStatus,
        error: lifecycleError,
        setError: setLifecycleError,
        tokenUsage,
        loadedConfig,
        isSetupRequired,
        setIsSetupRequired,
        isProjectPermissionRequired,
        setIsProjectPermissionRequired,
        pendingRepoRoot,
        reloadSession
    } = useSessionLifecycle(client, repoPath, requestedSessionId, engineExited, skipConnection);

    // 2. Response Streaming (Throttled)
    const {
        currentThought,
        setCurrentThought
    } = useResponseStream(client, appendAssistantContent, markLastTurnDone);

    // 3. Tool Activities Project Plan
    const {
        projectPlan,
        showProjectPlan,
        activeStepIdsRef
    } = useToolActivities(client, {
        addActivity,
        addTimelineStep,
        updateTimelineStep,
        getCurrentTurnId,
        addContextEvent,
        appendToolOutput
    });

    // 4. Session History / Resume
    useEngineEvents(client, {
        onSessionHistory: useCallback((title: string, summary: string, _messages: any[]) => {
            if (summary) {
                // For resume, we push a special turn to show the summary
                // We'll use a unique ID that we can reference
                const resumeTurnId = `resume-${Date.now()}`;

                // We can't easily push a turn with a specific ID using pushTurn,
                // but we can just use the fact that addContextEvent is called after a render.
                // Instead, let's just push a turn and then the summary will be added to it
                // because it's the last turn.

                pushTurn(`Resumed: ${title}`);

                // We need to wait for the turn to be pushed, but addContextEvent 
                // in useConversation already handles finding the last turn if we pass null? 
                // No, it requires turnId.

                // Actually, let's just mark the last turn done with the summary.
                const timer = setTimeout(() => {
                    markLastTurnDone(summary);
                }, 100);
                return () => clearTimeout(timer);
            }
        }, [pushTurn, markLastTurnDone])
    });

    // Helper to submit user input
    // This needs to interact with sessionId and status, which are now in useSessionLifecycle
    // We can just act on them.
    const submitQuery = useCallback((query: string) => {
        if (!sessionId) return;
        logger.state("User Query Submitted", { query, sessionId });
        pushTurn(query);
        activeStepIdsRef.current.clear();
        // setIsRunning(true); // Removed isRunning state - derived from status?
        // statusRef.current = "thinking"; 
        setStatus("thinking");
        // setError(undefined);
        setLifecycleError(undefined);
        client.sendUserMessage(sessionId, query);
    }, [sessionId, pushTurn, client, setStatus, setLifecycleError]);

    // Derived isRunning state for backward compatibility
    // In original code, isRunning was explicit. Here we can derive it or keep it simple.
    // The original code set isRunning=true on submit, and false on Done/Error/Cancel.
    // Let's use status to derive it if possible, or re-introduce a local isRunning if needed.
    // Original: const [isRunning, setIsRunning] = useState(false);
    // Actually, let's keep isRunning based on status for now to avoid breaking App.tsx contracts too much.
    // Or better:
    const isRunning = status === "thinking" || status === "running_tools";

    // Cancel Request Logic (moved from original hook)
    const cancelRequest = useCallback(() => {
        logger.log(`[useEngineConnection] cancelRequest called. sessionId=${sessionId}, status=${status}, isRunning=${isRunning}`);

        if (!sessionId) {
            logger.error('[useEngineConnection] No sessionId, cannot cancel');
            return false;
        }

        if (!isRunning) {
            logger.log('[useEngineConnection] Cancel skipped - not running');
            return false;
        }

        logger.log('[useEngineConnection] Sending cancel_request');

        // Send cancel_request command
        client.sendCommand({
            type: "cancel_request",
            session_id: sessionId
        });

        setInfoMessage("Stopping...");
        return true;
    }, [sessionId, isRunning, status, client, setInfoMessage]);

    // Memoize current turn's timeline steps and log lines
    const { currentTimelineSteps, currentLogLines, currentRunningStep } = useMemo(() => {
        const currentTurn = turns.length > 0 ? turns[turns.length - 1] : null;
        const timelineSteps = currentTurn?.timelineSteps || [];
        const logLines = currentTurn?.logLines || [];
        const runningStep = timelineSteps.find((s) => s.status === "running");
        return {
            currentTimelineSteps: timelineSteps,
            currentLogLines: logLines,
            currentRunningStep: runningStep,
        };
    }, [turns]);


    return {
        sessionId,
        status,
        infoMessage,
        isRunning,
        error: lifecycleError,
        tokenUsage,
        projectPlan,
        showProjectPlan,
        errorCount,
        currentThought,
        turns,
        currentTimelineSteps,
        currentRunningStepId: currentRunningStep?.id,
        toggleTurnCollapsed,
        submitQuery,
        setInput: () => { },
        sendCommand: client.sendCommand.bind(client),
        isSetupRequired,
        setIsSetupRequired,
        isProjectPermissionRequired,
        setIsProjectPermissionRequired,
        pendingRepoRoot,
        loadedConfig,
        reloadSession,
        cancelRequest,
        clearTurns,
    };
}
