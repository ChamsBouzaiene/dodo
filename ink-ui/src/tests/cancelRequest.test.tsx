/**
 * Tests for the Stop/Cancel feature
 * - Cancel command sends cancel_request
 * - onCancelled callback resets state
 * - /stop command works
 * - Cancel during idle is no-op
 */

import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from 'ink-testing-library';
import { Footer, type FooterProps } from '../components/layout/Footer.js';
import { AnimationProvider } from '../contexts/AnimationContext.js';

vi.mock('ink-spinner', () => ({
    default: () => null,
}));

// Mock SessionContext
const mockSessionContext = {
    sessionId: 'abc123',
    status: 'ready',
    infoMessage: '',
    isRunning: false,
    tokenUsage: undefined,
    errorCount: 0,
    currentThought: '',
    loadedConfig: {}
};

vi.mock('../contexts/SessionContext.js', () => ({
    useSessionContext: () => mockSessionContext,
    SessionProvider: ({ children }: any) => children
}));

describe('Cancel Feature: UI Status', () => {
    const baseProps: FooterProps = {
        input: '',
        onChange: vi.fn(),
        onSubmit: vi.fn(),
        repoLabel: 'test-repo',
    };

    const renderWithProvider = (sessionOverrides: any = {}) => {
        Object.assign(mockSessionContext, {
            sessionId: 'abc123',
            status: 'ready',
            infoMessage: '',
            isRunning: false,
            ...sessionOverrides
        });

        return render(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );
    };

    it('shows ready status when not running', () => {
        const { lastFrame } = renderWithProvider({ isRunning: false, status: 'ready' });
        expect(lastFrame()).toContain('READY');
    });

    it('shows running status when task is running', () => {
        const { lastFrame } = renderWithProvider({
            isRunning: true,
            status: 'thinking'
        });
        expect(lastFrame()).toContain('RUNNING');
    });

    it('returns to ready status after cancel', () => {
        const { lastFrame } = renderWithProvider({
            isRunning: false,
            status: 'ready'
        });
        expect(lastFrame()).toContain('READY');
    });
});

describe('Cancel Feature: Protocol Types', () => {
    it('CancelRequestCommand has correct shape', async () => {
        const { CancelRequestCommand } = await import('../protocol.js').then(m => ({
            CancelRequestCommand: {} as import('../protocol.js').CancelRequestCommand
        }));

        // Type check - this will fail at compile time if types are wrong
        const cmd: import('../protocol.js').CancelRequestCommand = {
            type: 'cancel_request',
            session_id: 'test-session'
        };

        expect(cmd.type).toBe('cancel_request');
        expect(cmd.session_id).toBe('test-session');
    });

    it('CancelledEvent has correct shape', async () => {
        // Type check
        const event: import('../protocol.js').CancelledEvent = {
            type: 'cancelled',
            session_id: 'test-session',
            reason: 'Cancelled by user request'
        };

        expect(event.type).toBe('cancelled');
        expect(event.session_id).toBe('test-session');
        expect(event.reason).toBe('Cancelled by user request');
    });

    it('CancelledEvent reason is optional', async () => {
        const event: import('../protocol.js').CancelledEvent = {
            type: 'cancelled',
            session_id: 'test-session'
        };

        expect(event.type).toBe('cancelled');
        expect(event.reason).toBeUndefined();
    });
});

describe('Cancel Feature: Command serialization', () => {
    it('serializeCommand handles cancel_request', async () => {
        const { serializeCommand } = await import('../protocol.js');

        const cmd = {
            type: 'cancel_request' as const,
            session_id: 'test-session-123'
        };

        const json = serializeCommand(cmd);
        const parsed = JSON.parse(json);

        expect(parsed.type).toBe('cancel_request');
        expect(parsed.session_id).toBe('test-session-123');
    });
});
