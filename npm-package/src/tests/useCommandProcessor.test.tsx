import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render } from 'ink-testing-library';
import { useCommandProcessor } from '../hooks/useCommandProcessor.js';

// Mock deps
vi.mock('../utils/diagnostics.js', () => ({
    generateDiagnostics: vi.fn(),
    writeDiagnostics: vi.fn(() => '/tmp/debug.log'),
}));
vi.mock('../utils/debugLogger.js', () => ({
    debugLog: { command: vi.fn() }
}));

function renderHook<T>(hook: () => T) {
    const result = { current: null as any };
    function TestComponent() {
        result.current = hook();
        return null;
    }
    const { rerender } = render(<TestComponent />);
    return { result, rerender };
}

describe('useCommandProcessor', () => {
    const mockContext = {
        sessionId: '123',
        status: 'ready',
        isRunning: false,
        loadedConfig: {},
        isSetupRequired: false,
        isUpdateMode: false,
        clearTurns: vi.fn(),
        cancelRequest: vi.fn(),
        sendCommand: vi.fn(),
        setIsSetupRequired: vi.fn(),
        setIsUpdateMode: vi.fn(),
        setActiveModal: vi.fn(),
        setInput: vi.fn(),
    };

    const getHook = (ctx = mockContext) => renderHook(() => useCommandProcessor(ctx));

    it('returns false for non-commands', () => {
        const { result } = getHook();
        expect(result.current.processCommand('hello')).toBe(false);
    });

    it('handles /help', () => {
        const { result } = getHook();
        const handled = result.current.processCommand('/help');
        expect(handled).toBe(true);
        expect(mockContext.setActiveModal).toHaveBeenCalledWith('help');
    });

    it('handles /clear', () => {
        const { result } = getHook();
        const handled = result.current.processCommand('/clear');
        expect(handled).toBe(true);
        expect(mockContext.clearTurns).toHaveBeenCalled();
    });

    it('handles /stop when running', () => {
        const runningCtx = { ...mockContext, isRunning: true };
        const { result } = getHook(runningCtx);
        const handled = result.current.processCommand('/stop');
        expect(handled).toBe(true);
        expect(runningCtx.cancelRequest).toHaveBeenCalled();
    });
});
