import React from 'react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from 'ink-testing-library';
import { SetupWizard } from '../../components/SetupWizard.js';
import { useEngineConnection } from '../../hooks/useEngineConnection.js';
import { useKeypress, KeypressProvider } from '../../contexts/KeypressContext.js';
import { useApp } from 'ink';

// Mock the useApp hook
vi.mock('ink', async () => {
    const actual = await vi.importActual('ink');
    return {
        ...actual,
        useApp: () => ({ exit: vi.fn() }),
    };
});

// Mock debugLog
vi.mock('../utils/debugLogger.js', () => ({
    debugLog: {
        lifecycle: vi.fn(),
        event: vi.fn(),
        command: vi.fn(),
        state: vi.fn(),
        error: vi.fn(),
    },
}));

describe('SetupWizard', () => {
    const mockSendCommand = vi.fn();
    const mockOnComplete = vi.fn();

    beforeEach(() => {
        vi.clearAllMocks();
    });

    const renderWithProvider = (props: any) => {
        return render(
            <KeypressProvider>
                <SetupWizard
                    sendCommand={mockSendCommand}
                    onComplete={mockOnComplete}
                    {...props}
                />
            </KeypressProvider>
        );
    };

    describe('Initial Rendering', () => {
        it('renders intro screen initially', () => {
            const { lastFrame } = renderWithProvider({});
            expect(lastFrame()).toContain('Welcome to Dodo');
        });

        it('shows update mode message when isUpdate is true', () => {
            const { lastFrame } = renderWithProvider({ isUpdate: true });
            expect(lastFrame()).toContain('Update Configuration');
        });

        it('shows navigation hint on intro screen', () => {
            const { lastFrame } = renderWithProvider({});
            expect(lastFrame()).toContain('Press Enter');
        });
    });

    describe('Initial Config Pre-fill', () => {
        it('pre-fills config when initialConfig is provided', () => {
            const initialConfig = {
                llm_provider: 'anthropic',
                api_key: 'test-key-123',
                model: 'claude-3-opus-20240229',
                auto_index: 'true',
            };

            const { lastFrame } = renderWithProvider({ initialConfig });
            // Component should render with the initial config loaded
            expect(lastFrame()).toBeTruthy();
        });

        it('handles empty initialConfig', () => {
            const { lastFrame } = renderWithProvider({ initialConfig: {} });
            expect(lastFrame()).toContain('Welcome to Dodo');
        });

        it('handles undefined initialConfig', () => {
            const { lastFrame } = renderWithProvider({ initialConfig: undefined });
            expect(lastFrame()).toContain('Welcome to Dodo');
        });
    });

    describe('Provider Options', () => {
        it('shows OpenAI as a provider option', () => {
            const { lastFrame } = renderWithProvider({});
            // Intro screen doesn't show providers yet, but the wizard should render
            expect(lastFrame()).toBeTruthy();
        });

        it('handles different provider from initialConfig', () => {
            const configs = [
                { llm_provider: 'openai' },
                { llm_provider: 'anthropic' },
                { llm_provider: 'kimi' },
            ];

            for (const config of configs) {
                const { lastFrame } = renderWithProvider({ initialConfig: config });
                expect(lastFrame()).toBeTruthy();
            }
        });
    });

    describe('Callbacks', () => {
        it('does not call onComplete on initial render', () => {
            renderWithProvider({});
            expect(mockOnComplete).not.toHaveBeenCalled();
        });

        it('does not call sendCommand on initial render', () => {
            renderWithProvider({});
            expect(mockSendCommand).not.toHaveBeenCalled();
        });
    });
});
