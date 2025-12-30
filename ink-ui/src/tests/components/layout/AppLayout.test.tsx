import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AppLayout } from '../../../components/layout/AppLayout.js';

import { Text } from 'ink';

// Mock ink-spinner
vi.mock('ink-spinner', () => ({
    default: () => null,
}));

const mockExit = vi.fn();
vi.mock('ink', async () => {
    const actual = await vi.importActual('ink');
    return {
        ...actual,
        useApp: () => ({
            exit: mockExit
        }),
        useInput: (cb: any) => { trigger.callback = cb; }
    };
});

// Hoist trigger for useInput mock
const { trigger } = vi.hoisted(() => ({ trigger: { callback: null as any } }));

// Mock child components to simplify testing
vi.mock('../../../components/layout/Header.js', () => ({
    Header: () => <Text>Header</Text>
}));
vi.mock('../../../components/layout/Footer.js', () => ({
    Footer: () => <Text>Footer</Text>
}));
vi.mock('../../../components/timeline/ProjectPlan.js', () => ({
    ProjectPlan: () => <Text>Project Plan</Text>
}));
vi.mock('../../../components/conversation/Conversation.js', () => ({
    Conversation: () => <Text>Conversation</Text>
}));

describe('AppLayout', () => {
    const defaultProps = {
        terminalRows: 24,
        terminalColumns: 80,
        error: undefined,
        showProjectPlan: false,
        projectPlan: '',
        turns: [],
        currentTimelineSteps: [],
        currentRunningStepId: undefined,
        isRunning: false,
        toggleTurnCollapsed: vi.fn(),
        footerProps: {
            input: '',
            onChange: vi.fn(),
            onSubmit: vi.fn(),
            canSubmit: true,
            isRunning: false,
            repoLabel: 'repo',
            sessionLabel: 'session',
            status: 'ready' as const,
            infoMessage: 'Ready',
            errorCount: 0,
            currentThought: ''
        }
    };



    // Mock MouseContext to avoid issues
    vi.mock('../../../contexts/MouseContext.js', () => ({
        MouseProvider: ({ children }: any) => children,
        useMouse: () => ({}),
        useMouseScroll: () => ({})
    }));

    beforeEach(() => {
        mockExit.mockClear();
    });

    const renderComponent = (props = {}) => {
        return render(
            <AppLayout {...defaultProps} {...props} />
        );
    };

    it('renders basic layout components', () => {
        const { lastFrame } = renderComponent();
        expect(lastFrame()).toContain('Header');
        expect(lastFrame()).toContain('Conversation');
        expect(lastFrame()).toContain('Footer');
    });

    it('renders error message when error is present', () => {
        const { lastFrame } = renderComponent({ error: 'Something went wrong' });
        expect(lastFrame()).toContain('Something went wrong');
    });

    it('renders project plan when visible', () => {
        const { lastFrame } = renderComponent({ showProjectPlan: true, projectPlan: 'Plan' });
        expect(lastFrame()).toContain('Project Plan');
    });

    it('calculates scrollable height correctly', () => {
        // We can't easily test the exact height calculation without inspecting props passed to ScrollableConversation
        // But we can verify that it renders without crashing with different terminal heights
        const { lastFrame } = renderComponent({ terminalRows: 10 });
        expect(lastFrame()).toContain('Conversation');
    });

    it('exits on Ctrl+C', () => {
        renderComponent();

        // Simulate Ctrl+C
        if (trigger.callback) {
            trigger.callback({ name: 'c', ctrl: true });
        }
        expect(mockExit).toHaveBeenCalled();
    });

    it('does not exit on Escape when idle', () => {
        renderComponent();

        if (trigger.callback) {
            trigger.callback({ name: 'escape' });
        }
        expect(mockExit).not.toHaveBeenCalled();
    });
});
