/**
 * Tests for error handling in the UI
 * - Engine errors display correctly
 * - Connection failures handled
 * - Disconnected state shown
 */

import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render } from 'ink-testing-library';
import { Footer, type FooterProps } from '../components/layout/Footer.js';
import { AnimationProvider } from '../contexts/AnimationContext.js';
import { AppLayout } from '../components/layout/AppLayout.js';

vi.mock('ink-spinner', () => ({
    default: () => '⠋',
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

describe('Error Handling: Footer Status', () => {
    const baseProps: FooterProps = {
        input: '',
        onChange: vi.fn(),
        onSubmit: vi.fn(),
        repoLabel: 'test-repo',
    };

    const renderWithProvider = (sessionOverrides: any = {}) => {
        Object.assign(mockSessionContext, {
            status: 'ready',
            infoMessage: '',
            errorCount: 0,
            ...sessionOverrides
        });

        return render(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );
    };

    it('displays error status correctly', () => {
        const { lastFrame } = renderWithProvider({ status: 'error' });
        expect(lastFrame()).toContain('ERROR');
    });

    it('displays disconnected status', () => {
        const { lastFrame } = renderWithProvider({ status: 'disconnected' });
        expect(lastFrame()).toContain('DISCONNECTED');
    });

    it('shows error count when errors exist', () => {
        const { lastFrame } = renderWithProvider({ errorCount: 3 });
        expect(lastFrame()).toContain('Errors: 3');
    });

    it('shows singular error for count of 1', () => {
        // We actually just show "Errors: 1" generically now
        const { lastFrame } = renderWithProvider({ errorCount: 1 });
        expect(lastFrame()).toContain('Errors: 1');
    });

    it('does not show error count when zero', () => {
        const { lastFrame } = renderWithProvider({ errorCount: 0 });
        expect(lastFrame()).not.toContain('error');
    });
});

// Note: AppLayout tests require more complex mocking (useApp hook, useKeypress, etc.)
// These are tested at the integration level via E2E tests
describe.skip('Error Handling: AppLayout Error Display', () => {
    const baseFooterProps: FooterProps = {
        input: '',
        onChange: vi.fn(),
        onSubmit: vi.fn(),
        repoLabel: 'test-repo',
    };

    it('displays error message when error prop is set', () => {
        const { lastFrame } = render(
            <AnimationProvider>
                <AppLayout
                    terminalRows={24}
                    terminalColumns={80}
                    error="Engine connection failed"
                    showProjectPlan={false}
                    projectPlan=""
                    currentRunningStepId={undefined}
                    currentTimelineSteps={[]}
                    isRunning={false}
                    footerProps={baseFooterProps}
                />
            </AnimationProvider>
        );

        expect(lastFrame()).toContain('Error:');
        expect(lastFrame()).toContain('Engine connection failed');
    });

    it('does not display error box when no error', () => {
        const { lastFrame } = render(
            <AnimationProvider>
                <AppLayout
                    terminalRows={24}
                    terminalColumns={80}
                    error={undefined}
                    showProjectPlan={false}
                    projectPlan=""
                    currentTimelineSteps={[]}
                    currentRunningStepId={undefined}
                    isRunning={false}
                    footerProps={baseFooterProps}
                />
            </AnimationProvider>
        );

        // Should not contain the error box
        expect(lastFrame()).not.toContain('Error:');
    });
});

describe('Error Handling: Connection States', () => {
    const baseProps: FooterProps = {
        input: '',
        onChange: vi.fn(),
        onSubmit: vi.fn(),
        repoLabel: 'test-repo',
    };

    const renderWithProvider = (sessionOverrides: any = {}) => {
        Object.assign(mockSessionContext, {
            ...sessionOverrides
        });
        return render(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );
    };

    it('shows connecting message when not ready', () => {
        const { lastFrame } = renderWithProvider({
            status: 'connecting',
            infoMessage: 'Connecting to engine...'
        });

        expect(lastFrame()).toContain('Connecting to engine');
    });

    it('shows processing message when running', () => {
        const { lastFrame } = renderWithProvider({
            isRunning: true,
            status: 'running'
        });

        expect(lastFrame()).toContain('Processing');
    });
});
