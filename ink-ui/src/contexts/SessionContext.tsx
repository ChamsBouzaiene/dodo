import React, { createContext, useContext } from 'react';
import type { UiStatus, TokenUsage } from '../types.js';

interface SessionContextValue {
    sessionId?: string;
    status: UiStatus;
    infoMessage: string;
    isRunning: boolean;
    error?: string;
    tokenUsage?: TokenUsage;
    errorCount: number;
    currentThought: string;
    loadedConfig?: Record<string, string | number | boolean>;
    isSetupRequired: boolean;
}

const SessionContext = createContext<SessionContextValue | undefined>(undefined);

export function SessionProvider({
    children,
    value
}: {
    children: React.ReactNode;
    value: SessionContextValue;
}) {
    return (
        <SessionContext.Provider value={value}>
            {children}
        </SessionContext.Provider>
    );
}

export function useSessionContext() {
    const context = useContext(SessionContext);
    if (!context) {
        throw new Error('useSessionContext must be used within a SessionProvider');
    }
    return context;
}
