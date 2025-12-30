import React, { useEffect } from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { useHistory } from '../hooks/useHistory.js';

function renderHook<T>(hook: () => T) {
    const result = { current: null as any };
    function TestComponent() {
        result.current = hook();
        return null;
    }
    const { rerender } = render(<TestComponent />);
    return { result, rerender };
}

// Helper to wait for state updates
const waitForUpdate = () => new Promise(resolve => setTimeout(resolve, 0));

describe('useHistory', () => {
    it('initializes empty', () => {
        const { result } = renderHook(() => useHistory());
        expect(result.current.history).toEqual([]);
        expect(result.current.historyIndex).toBe(-1);
    });

    it('adds commands', async () => {
        const { result } = renderHook(() => useHistory());

        result.current.addToHistory('cmd1');
        await waitForUpdate();
        expect(result.current.history).toEqual(['cmd1']);

        result.current.addToHistory('cmd2');
        await waitForUpdate();
        expect(result.current.history).toEqual(['cmd1', 'cmd2']);
    });

    it('prevents consecutive duplicates', async () => {
        const { result } = renderHook(() => useHistory());

        result.current.addToHistory('cmd1');
        await waitForUpdate();
        result.current.addToHistory('cmd1');
        await waitForUpdate();

        expect(result.current.history).toEqual(['cmd1']);
    });

    it('navigates up and down', async () => {
        const { result } = renderHook(() => useHistory());

        result.current.addToHistory('1');
        await waitForUpdate();
        result.current.addToHistory('2');
        await waitForUpdate();
        result.current.addToHistory('3');
        await waitForUpdate();

        // Up 1 -> "3"
        let res = result.current.navigateHistory('up');
        await waitForUpdate();
        expect(res.value).toBe('3');
        expect(result.current.historyIndex).toBe(0);

        // Up 2 -> "2"
        res = result.current.navigateHistory('up');
        await waitForUpdate();
        expect(res.value).toBe('2');
        expect(result.current.historyIndex).toBe(1);

        // Down 1 -> "3"
        res = result.current.navigateHistory('down');
        await waitForUpdate();
        expect(res.value).toBe('3');
        expect(result.current.historyIndex).toBe(0);

        // Down 1 -> empty
        res = result.current.navigateHistory('down');
        await waitForUpdate();
        expect(res.value).toBeUndefined();
        expect(result.current.historyIndex).toBe(-1);
    });
});
