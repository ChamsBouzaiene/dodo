import { useState, useRef, useCallback, useEffect } from "react";
import type { EngineClient } from "../engineClient.js";
import { useEngineEvents } from "./useEngineEvents.js";
import { debugLog } from "../utils/debugLogger.js";
import type { Activity, DisplayToolEvent, UiStatus } from "../types.js";

type ConversationActions = {
    addActivity: (turnId: string, activity: Activity) => void;
    addTimelineStep: (turnId: string, step: any) => void;
    updateTimelineStep: (turnId: string, stepId: string, updates: any) => void;
    getCurrentTurnId: () => string | null;
    addContextEvent: (turnId: string, event: any) => void;
    appendToolOutput: (turnId: string, invocationId: string, output: string, stream: "stdout" | "stderr") => void;
};

export function useToolActivities(
    client: EngineClient,
    conversation: ConversationActions
) {
    const [projectPlan, setProjectPlan] = useState<string>("");
    const [showProjectPlan, setShowProjectPlan] = useState(false);

    // Track active steps to prevent duplicate updates
    const activeStepIdsRef = useRef<Set<string>>(new Set());

    const {
        addActivity,
        addTimelineStep,
        updateTimelineStep,
        getCurrentTurnId,
        addContextEvent,
        appendToolOutput
    } = conversation;

    const THROTTLE_MS = 50;

    // Activity buffering logic
    const activityBufferRef = useRef<Activity[]>([]);
    const activityThrottleTimerRef = useRef<NodeJS.Timeout | null>(null);
    const lastActivityFlushRef = useRef<number>(0);

    const flushActivityBuffer = useCallback(() => {
        if (activityThrottleTimerRef.current) {
            clearTimeout(activityThrottleTimerRef.current);
            activityThrottleTimerRef.current = null;
        }

        const currentTurnId = getCurrentTurnId();
        if (!currentTurnId) {
            activityBufferRef.current = [];
            return;
        }

        activityBufferRef.current.forEach((activity) => {
            addActivity(currentTurnId, activity);

            if (activity.type === "tool" || activity.type === "reasoning") {
                const stepId = activity.invocationId || activity.id;

                if (activity.status === "active") {
                    if (!activeStepIdsRef.current.has(stepId)) {
                        addTimelineStep(currentTurnId, {
                            id: stepId,
                            toolName: activity.tool || "unknown",
                            label: activity.tool || "unknown",
                            status: "running",
                            startedAt: activity.timestamp,
                            type: "tool",
                            command: activity.command,
                            metadata: activity.metadata,
                        });
                        activeStepIdsRef.current.add(stepId);
                    } else {
                        updateTimelineStep(currentTurnId, stepId, {
                            metadata: activity.metadata,
                            command: activity.command || undefined,
                        });
                    }
                } else {
                    updateTimelineStep(currentTurnId, stepId, {
                        status: activity.status === "completed" ? "done" : "failed",
                        finishedAt: activity.timestamp,
                        metadata: activity.metadata,
                        command: activity.command || undefined,
                        durationMs: activity.durationMs,
                    });
                    activeStepIdsRef.current.delete(stepId);
                }
            }
        });

        activityBufferRef.current = [];
        lastActivityFlushRef.current = Date.now();
    }, [addActivity, addTimelineStep, updateTimelineStep, getCurrentTurnId]);

    // Output buffering logic
    const outputBufferRef = useRef<Map<string, { output: string, stream: "stdout" | "stderr", tool: string }>>(new Map());
    const throttleTimerRef = useRef<NodeJS.Timeout | null>(null);
    const lastFlushRef = useRef<number>(0);

    const flushOutputBuffer = useCallback(() => {
        if (throttleTimerRef.current) {
            clearTimeout(throttleTimerRef.current);
            throttleTimerRef.current = null;
        }

        const turnId = getCurrentTurnId();
        if (!turnId) return;

        outputBufferRef.current.forEach((data, key) => {
            const [invocationId] = key.split(':');
            appendToolOutput(turnId, invocationId, data.output, data.stream);
        });

        outputBufferRef.current.clear();
        lastFlushRef.current = Date.now();
    }, [getCurrentTurnId, appendToolOutput]);

    useEffect(() => {
        return () => {
            if (throttleTimerRef.current) clearTimeout(throttleTimerRef.current);
            if (activityThrottleTimerRef.current) clearTimeout(activityThrottleTimerRef.current);
        };
    }, []);

    useEngineEvents(client, {
        onActivity: useCallback((activity: Activity) => {
            activityBufferRef.current.push(activity);

            const now = Date.now();
            if (now - lastActivityFlushRef.current > THROTTLE_MS) {
                flushActivityBuffer();
            } else if (!activityThrottleTimerRef.current) {
                activityThrottleTimerRef.current = setTimeout(flushActivityBuffer, THROTTLE_MS);
            }
        }, [flushActivityBuffer]),

        onProjectPlan: useCallback((content: string, source: string) => {
            setProjectPlan(content);
            if (content && !showProjectPlan) {
                setShowProjectPlan(true);
            }
        }, [showProjectPlan]),

        onContext: useCallback((event: { kind: string; description: string; before: number; after: number }) => {
            const currentTurnId = getCurrentTurnId();
            if (currentTurnId) {
                addContextEvent(currentTurnId, event);
            }
        }, [getCurrentTurnId, addContextEvent]),

        onToolOutput: useCallback((invocationId: string, tool: string, output: string, stream: "stdout" | "stderr") => {
            const key = `${invocationId}:${stream}`;
            const existing = outputBufferRef.current.get(key);
            if (existing) {
                existing.output += output;
            } else {
                outputBufferRef.current.set(key, { output, stream, tool });
            }

            const now = Date.now();
            if (now - lastFlushRef.current > THROTTLE_MS) {
                flushOutputBuffer();
            } else if (!throttleTimerRef.current) {
                throttleTimerRef.current = setTimeout(flushOutputBuffer, THROTTLE_MS);
            }
        }, [flushOutputBuffer]),
    });

    return {
        projectPlan,
        showProjectPlan,
        setShowProjectPlan,
        activeStepIdsRef // Exported if needed by parent, e.g. to clear on new turn
    };
}
