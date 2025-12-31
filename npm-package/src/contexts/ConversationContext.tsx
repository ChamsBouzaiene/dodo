import React, { createContext, useContext, useMemo } from 'react';
import { useConversationState } from '../hooks/useConversation.js';
import type { Turn, Activity, TimelineStep, LogLine } from '../types.js';

export interface ConversationContextValue {
    turns: Turn[];
    pushTurn: (userMessage: string) => void;
    appendAssistantContent: (content: string, replace?: boolean) => void;
    markLastTurnDone: (summary?: string) => void;
    addActivity: (turnId: string, activity: Activity) => void;
    updateActivity: (turnId: string, activityId: string, updates: Partial<Activity>) => void;
    toggleTurnCollapsed: (turnId: string) => void;
    addTimelineStep: (turnId: string, step: TimelineStep) => void;
    updateTimelineStep: (turnId: string, stepId: string, updates: Partial<TimelineStep>) => void;
    appendLogLine: (turnId: string, line: LogLine) => void;
    appendToolOutput: (turnId: string, invocationId: string, output: string, stream: "stdout" | "stderr") => void;
    addContextEvent: (turnId: string, event: { kind: string; description: string; before: number; after: number }) => void;
    getCurrentTurnId: () => string | null;
    clearTurns: () => void;
}

const ConversationContext = createContext<ConversationContextValue | undefined>(undefined);

export function ConversationProvider({ children }: { children: React.ReactNode }) {
    const conversation = useConversationState();

    return (
        <ConversationContext.Provider value={conversation}>
            {children}
        </ConversationContext.Provider>
    );
}

export function useConversationContext() {
    const context = useContext(ConversationContext);
    if (!context) {
        throw new Error('useConversationContext must be used within a ConversationProvider');
    }
    return context;
}
