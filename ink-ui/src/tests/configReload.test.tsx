/**
 * Tests for config hot-reload functionality
 * - config_loaded event handling
 * - config_reloaded event handling  
 * - Model sync in footer after config change
 */

import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from 'ink-testing-library';
import { Footer, type FooterProps } from '../components/layout/Footer.js';
import { AnimationProvider } from '../contexts/AnimationContext.js';

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

describe('Config Hot-Reload: Footer Model Sync', () => {
    const baseProps: FooterProps = {
        input: '',
        onChange: vi.fn(),
        onSubmit: vi.fn(),
        repoLabel: 'test-repo',
    };

    const renderWithProvider = (sessionOverrides: any = {}) => {
        Object.assign(mockSessionContext, {
            ...sessionOverrides,
            // If modelLabel is passed (legacy prop logic in test), put it in loadedConfig
            loadedConfig: sessionOverrides.modelLabel ? { model: sessionOverrides.modelLabel } : (sessionOverrides.loadedConfig || {})
        });

        return render(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );
    };

    it('initially shows no model when loadedConfig is empty', () => {
        const { lastFrame } = renderWithProvider({ modelLabel: undefined });
        expect(lastFrame()).not.toContain('Model:');
    });

    it('shows model after config is loaded', () => {
        const { lastFrame } = renderWithProvider({ modelLabel: 'gpt-4o' });
        expect(lastFrame()).toContain('Model:');
        expect(lastFrame()).toContain('gpt-4o');
    });

    it('updates model when config changes', () => {
        const { lastFrame, rerender } = renderWithProvider({ modelLabel: 'gpt-4o' });

        expect(lastFrame()).toContain('gpt-4o');

        // Simulate config reload with new model
        // Update mock context
        Object.assign(mockSessionContext, {
            loadedConfig: { model: 'claude-3-opus' }
        });

        rerender(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );

        expect(lastFrame()).toContain('claude-3-opus');
        expect(lastFrame()).not.toContain('gpt-4o');
    });

    it('handles switching between providers', () => {
        const { lastFrame, rerender } = renderWithProvider({ modelLabel: 'gpt-4o' });

        expect(lastFrame()).toContain('gpt-4o');

        // Switch to Kimi
        Object.assign(mockSessionContext, {
            loadedConfig: { model: 'kimi-k2-250711' }
        });

        rerender(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );

        expect(lastFrame()).toContain('kimi-k2-250711');
    });
});

describe('Config Hot-Reload: Status Updates', () => {
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
            ...sessionOverrides
        });
        return render(
            <AnimationProvider>
                <Footer {...baseProps} />
            </AnimationProvider>
        );
    };

    it('shows ready status after config reload', () => {
        const { lastFrame } = renderWithProvider({
            status: 'ready',
            infoMessage: 'Config reloaded: openai (gpt-4o)'
        });

        expect(lastFrame()).toContain('READY');
    });

    it('shows connecting status during reload', () => {
        const { lastFrame } = renderWithProvider({
            status: 'connecting',
            infoMessage: 'Reloading configuration...'
        });

        expect(lastFrame()).toContain('CONNECTING');
    });
});
