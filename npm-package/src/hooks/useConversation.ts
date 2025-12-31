import { useCallback, useState, useContext } from "react";
import type { Turn, Activity, TimelineStep, LogLine } from "../types.js";
import { useConversationContext } from "../contexts/ConversationContext.js";

/**
 * Helper to perform shallow equality check on metadata objects
 */
function isMetadataEqual(a: any, b: any): boolean {
  if (a === b) return true;
  if (!a || !b) return a === b;
  const keysA = Object.keys(a);
  const keysB = Object.keys(b);
  if (keysA.length !== keysB.length) return false;
  return keysA.every(key => a[key] === b[key]);
}

/**
 * Hook to manage conversation state. 
 * Used only by ConversationProvider.
 */
export function useConversationState() {
  const [turns, setTurns] = useState<Turn[]>([]);

  const pushTurn = useCallback((userMessage: string) => {
    const id = `${Date.now().toString(36)}-${Math.random().toString(16).slice(2, 6)}`;
    setTurns((prev) => [...prev, {
      id,
      user: userMessage,
      assistant: "",
      done: false,
      collapsed: false,
      activities: [],
      timelineSteps: [],
      logLines: [],
    }]);
  }, []);

  const appendAssistantContent = useCallback((content: string, replace = false) => {
    setTurns((prev) => {
      if (prev.length === 0) return prev;
      const lastIndex = prev.length - 1;
      const last = prev[lastIndex];
      const newAssistant = replace ? content : last.assistant + content;

      if (last.assistant === newAssistant) return prev;

      const updated = [...prev];
      updated[lastIndex] = { ...last, assistant: newAssistant };
      return updated;
    });
  }, []);

  const markLastTurnDone = useCallback((summary?: string) => {
    setTurns((prev) => {
      if (prev.length === 0) return prev;
      const lastIndex = prev.length - 1;
      const last = prev[lastIndex];

      const updated = [...prev];
      updated[lastIndex] = { ...last, done: true, summary: summary || last.summary };
      return updated;
    });
  }, []);

  const addActivity = useCallback((turnId: string, activity: Activity) => {
    setTurns((prev) => {
      if (prev.length === 0) return prev;
      const lastIndex = prev.length - 1;

      // Optimize for active turn
      if (prev[lastIndex].id === turnId) {
        const updated = [...prev];
        const last = prev[lastIndex];
        updated[lastIndex] = {
          ...last,
          activities: [...last.activities, activity],
        };
        return updated;
      }

      // Fallback to map for older turns
      return prev.map((turn) => {
        if (turn.id === turnId) {
          return { ...turn, activities: [...turn.activities, activity] };
        }
        return turn;
      });
    });
  }, []);

  const updateActivity = useCallback(
    (turnId: string, activityId: string, updates: Partial<Activity>) => {
      setTurns((prev) => {
        if (prev.length === 0) return prev;
        const lastIndex = prev.length - 1;

        const updateTurnActivities = (turn: Turn) => {
          const updatedActivities = turn.activities.map((activity) => {
            if (activity.id === activityId) {
              // Only update if there are actual changes
              const hasChanges = Object.entries(updates).some(([key, value]) => {
                if (key === 'metadata') return !isMetadataEqual(activity.metadata, value);
                return (activity as any)[key] !== value;
              });
              if (!hasChanges) return activity;
              return { ...activity, ...updates };
            }
            return activity;
          });
          const activitiesChanged = updatedActivities.some((a, i) => a !== turn.activities[i]);
          if (!activitiesChanged) return turn;
          return { ...turn, activities: updatedActivities };
        };

        // Optimize for active turn
        if (prev[lastIndex].id === turnId) {
          const updated = [...prev];
          updated[lastIndex] = updateTurnActivities(prev[lastIndex]);
          return updated;
        }

        return prev.map((turn) => {
          if (turn.id === turnId) {
            return updateTurnActivities(turn);
          }
          return turn;
        });
      });
    },
    []
  );

  const toggleTurnCollapsed = useCallback((turnId: string) => {
    setTurns((prev) => {
      return prev.map((turn) => {
        if (turn.id === turnId) {
          return { ...turn, collapsed: !turn.collapsed };
        }
        return turn;
      });
    });
  }, []);

  const addTimelineStep = useCallback((turnId: string, step: TimelineStep) => {
    setTurns((prev) => {
      if (prev.length === 0) return prev;
      const lastIndex = prev.length - 1;

      // Optimize for active turn
      if (prev[lastIndex].id === turnId) {
        const updated = [...prev];
        const last = prev[lastIndex];
        updated[lastIndex] = {
          ...last,
          timelineSteps: [...(last.timelineSteps || []), step],
        };
        return updated;
      }

      return prev.map((turn) => {
        if (turn.id === turnId) {
          return { ...turn, timelineSteps: [...(turn.timelineSteps || []), step] };
        }
        return turn;
      });
    });
  }, []);

  const updateTimelineStep = useCallback(
    (turnId: string, stepId: string, updates: Partial<TimelineStep>) => {
      setTurns((prev) => {
        if (prev.length === 0) return prev;
        const lastIndex = prev.length - 1;

        const updateTurnSteps = (turn: Turn) => {
          const updatedSteps = (turn.timelineSteps || []).map((step) => {
            if (step.id === stepId) {
              // Only update if there are actual changes
              const hasChanges = Object.entries(updates).some(([key, value]) => {
                if (key === 'metadata') return !isMetadataEqual(step.metadata, value);
                return (step as any)[key] !== value;
              });
              if (!hasChanges) return step;
              return { ...step, ...updates };
            }
            return step;
          });
          const stepsChanged = updatedSteps.some((s, i) => s !== (turn.timelineSteps || [])[i]);
          if (!stepsChanged) return turn;
          return { ...turn, timelineSteps: updatedSteps };
        };

        // Optimize for active turn
        if (prev[lastIndex].id === turnId) {
          const updated = [...prev];
          updated[lastIndex] = updateTurnSteps(prev[lastIndex]);
          return updated;
        }

        return prev.map((turn) => {
          if (turn.id === turnId) {
            return updateTurnSteps(turn);
          }
          return turn;
        });
      });
    },
    []
  );

  const appendLogLine = useCallback((turnId: string, line: LogLine) => {
    setTurns((prev) => {
      if (prev.length === 0) return prev;
      const lastIndex = prev.length - 1;

      if (prev[lastIndex].id === turnId) {
        const updated = [...prev];
        const last = prev[lastIndex];
        updated[lastIndex] = {
          ...last,
          logLines: [...(last.logLines || []), line],
        };
        return updated;
      }

      return prev.map((turn) => {
        if (turn.id === turnId) {
          return { ...turn, logLines: [...(turn.logLines || []), line] };
        }
        return turn;
      });
    });
  }, []);

  const appendToolOutput = useCallback(
    (
      turnId: string,
      invocationId: string,
      output: string,
      stream: "stdout" | "stderr"
    ) => {
      if (!output) return; // Skip empty updates

      setTurns((prev) => {
        if (prev.length === 0) return prev;
        const lastIndex = prev.length - 1;

        const updateTurnOutput = (turn: Turn) => {
          const updatedSteps = (turn.timelineSteps || []).map((step) => {
            if (step.id === invocationId) {
              const currentMetadata = step.metadata || {};
              const currentStream = (currentMetadata[stream] as string) || "";
              return {
                ...step,
                metadata: {
                  ...currentMetadata,
                  [stream]: currentStream + output,
                },
              };
            }
            return step;
          });
          return { ...turn, timelineSteps: updatedSteps };
        };

        if (prev[lastIndex].id === turnId) {
          const updated = [...prev];
          updated[lastIndex] = updateTurnOutput(prev[lastIndex]);
          return updated;
        }

        return prev.map((turn) => {
          if (turn.id === turnId) {
            return updateTurnOutput(turn);
          }
          return turn;
        });
      });
    },
    []
  );

  const addContextEvent = useCallback((turnId: string, event: { kind: string; description: string; before: number; after: number }) => {
    const step: TimelineStep = {
      id: `ctx-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
      toolName: "context",
      label: event.description,
      status: "done",
      startedAt: new Date(),
      finishedAt: new Date(),
      type: "context_event",
      contextEventType: event.kind as "compress" | "summarize",
      metadata: {
        before: event.before,
        after: event.after,
      },
    };
    addTimelineStep(turnId, step);
  }, [addTimelineStep]);

  const getCurrentTurnId = useCallback(() => {
    if (turns.length === 0) return null;
    return turns[turns.length - 1].id;
  }, [turns]);

  const clearTurns = useCallback(() => {
    setTurns([]);
  }, []);

  return {
    turns,
    pushTurn,
    appendAssistantContent,
    markLastTurnDone,
    addActivity,
    updateActivity,
    toggleTurnCollapsed,
    addTimelineStep,
    updateTimelineStep,
    appendLogLine,
    appendToolOutput,
    addContextEvent,
    getCurrentTurnId,
    clearTurns,
  };
}

/**
 * Hook to consume conversation state from Context.
 */
export function useConversation() {
  return useConversationContext();
}
