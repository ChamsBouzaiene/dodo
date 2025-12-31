import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render } from 'ink-testing-library';
import { useTerminal, UseTerminalReturn, UseTerminalOptions } from '../hooks/useTerminal.js';
import { ANSI } from '../utils/ansi.js';

// Custom renderHook for Ink environment
function renderHook(hook: (props?: any) => UseTerminalReturn, initialProps?: UseTerminalOptions) {
    const result = { current: null as unknown as UseTerminalReturn };

    function TestComponent({ props }: { props?: UseTerminalOptions }) {
        result.current = hook(props);
        return null;
    }

    const { rerender, unmount } = render(<TestComponent props={initialProps} />);

    return {
        result,
        rerender: (newProps?: UseTerminalOptions) => rerender(<TestComponent props={newProps} />),
        unmount
    };
}

describe('useTerminal', () => {
    // Save original descriptors/values
    const originalWrite = process.stdout.write;
    const originalOn = process.stdout.on;
    const originalOff = process.stdout.off;
    // rows might be a getter or value
    const originalRowsDescriptor = Object.getOwnPropertyDescriptor(process.stdout, 'rows');

    beforeEach(() => {
        // Mock methods
        process.stdout.write = vi.fn();
        process.stdout.on = vi.fn();
        process.stdout.off = vi.fn();
        (process.stdout as any).isTTY = true;

        // Mock rows
        Object.defineProperty(process.stdout, 'rows', {
            value: 24,
            configurable: true
        });
    });

    afterEach(() => {
        // Restore
        process.stdout.write = originalWrite;
        process.stdout.on = originalOn;
        process.stdout.off = originalOff;

        if (originalRowsDescriptor) {
            Object.defineProperty(process.stdout, 'rows', originalRowsDescriptor);
        } else {
            // If it didn't exist (unlikely in node), delete it
            // delete process.stdout.rows; 
            // In typical node env it exists.
        }
    });

    it('initializes with default rows', () => {
        const { result } = renderHook(() => useTerminal());
        expect(result.current.terminalRows).toBe(24);
    });

    it('initializes with custom rows if stdout is available', () => {
        Object.defineProperty(process.stdout, 'rows', { value: 50, configurable: true });
        const { result } = renderHook(() => useTerminal());
        expect(result.current.terminalRows).toBe(50);
    });

    it('updates rows on resize event', async () => {
        let resizeHandler: (() => void) | undefined;
        (process.stdout.on as any) = vi.fn((event, handler) => {
            if (event === 'resize') resizeHandler = handler;
        });

        const { result } = renderHook(() => useTerminal());
        expect(result.current.terminalRows).toBe(24);

        // Simulate resize
        Object.defineProperty(process.stdout, 'rows', { value: 40, configurable: true });
        resizeHandler?.();
        await new Promise(r => setTimeout(r, 10));

        expect(result.current.terminalRows).toBe(40);
    });

    it('enters alternate buffer on mount by default', async () => {
        renderHook(() => useTerminal());
        await new Promise(r => setTimeout(r, 10));

        const calls = (process.stdout.write as any).mock.calls.map((c: any[]) => c[0]);
        // console.error('Enters calls:', calls);

        const entered = calls.some((c: string) => c.includes(ANSI.ALTERNATE_BUFFER_ENTER));
        const cleared = calls.some((c: string) => c.includes(ANSI.CLEAR_SCREEN));

        expect(entered).toBe(true);
        expect(cleared).toBe(true);
    });

    it('exits alternate buffer on unmount', async () => {
        const { unmount } = renderHook(() => useTerminal());
        unmount();
        await new Promise(r => setTimeout(r, 10));

        const calls = (process.stdout.write as any).mock.calls.map((c: any[]) => c[0]);
        // console.error('Exits calls:', calls);

        const exited = calls.some((c: string) => c.includes(ANSI.ALTERNATE_BUFFER_EXIT));
        expect(exited).toBe(true);
    });

    it('does not enter alternate buffer if disabled', () => {
        renderHook((props) => useTerminal(props), { enableAlternateBuffer: false });
        expect(process.stdout.write).not.toHaveBeenCalledWith(expect.stringContaining(ANSI.ALTERNATE_BUFFER_ENTER));
    });
});
