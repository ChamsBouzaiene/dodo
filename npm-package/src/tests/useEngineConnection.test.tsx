import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useEngineConnection } from '../hooks/useEngineConnection.js';
import { EngineClient } from '../engineClient.js';
import { Text } from 'ink';

// Mock dependencies
vi.mock('ink', async () => {
    const actual = await vi.importActual('ink');
    return {
        ...actual,
        useApp: vi.fn(() => ({ exit: vi.fn() })),
    };
});

vi.mock('../engineClient.js', () => {
    return {
        EngineClient: class {
            startSession = vi.fn().mockResolvedValue('session-id');
            sendUserMessage = vi.fn();
            sendCommand = vi.fn();
            on = vi.fn();
            off = vi.fn();
            close = vi.fn();
            removeAllListeners = vi.fn();
        },
    };
});

vi.mock('../hooks/useEngineEvents.js', () => ({
    useEngineEvents: vi.fn(),
}));

vi.mock('../hooks/useConversation.js', () => ({
    useConversation: vi.fn(() => ({
        turns: [],
        pushTurn: vi.fn(),
        appendAssistantContent: vi.fn(),
        markLastTurnDone: vi.fn(),
        addActivity: vi.fn(),
        updateActivity: vi.fn(),
        toggleTurnCollapsed: vi.fn(),
        addTimelineStep: vi.fn(),
        updateTimelineStep: vi.fn(),
        getCurrentTurnId: vi.fn(),
        appendLogLine: vi.fn(),
        appendToolOutput: vi.fn(),
        addContextEvent: vi.fn(),
    })),
}));

vi.mock('../utils/logger.js', () => ({
    logger: {
        state: vi.fn(),
    },
}));

const TestComponent = ({ client, repoPath, engineExited }: any) => {
    const { status, infoMessage, isRunning, error } = useEngineConnection(client, repoPath, undefined, engineExited);
    return (
        <Text>
            Status: {status}
            Info: {infoMessage}
            Running: {String(isRunning)}
            Error: {String(error)}
        </Text>
    );
};

describe('useEngineConnection', () => {
    let client: EngineClient;

    beforeEach(() => {
        vi.clearAllMocks();
        client = new EngineClient({} as any, {} as any);
    });

    it('initializes with default state', () => {
        const { lastFrame } = render(<TestComponent client={client} repoPath="/repo/path" />);

        expect(lastFrame()).toContain('Status: connecting');
        expect(lastFrame()).toContain('Info: Connecting to engine...');
        expect(lastFrame()).toContain('Running: false');
    });

    it('handles engine exit', async () => {
        const { lastFrame } = render(
            <TestComponent
                client={client}
                repoPath="/repo/path"
                engineExited={{ code: 1, signal: null }}
            />
        );

        // Wait for effects to run
        await new Promise(resolve => setTimeout(resolve, 10));

        expect(lastFrame()).toContain('Status: disconnected');
        expect(lastFrame()).toContain('Info: Engine exited (code 1)');
    });

    it('handles engine exit with signal (no code)', async () => {
        const { lastFrame } = render(
            <TestComponent
                client={client}
                repoPath="/repo/path"
                engineExited={{ code: null, signal: 'SIGTERM' }}
            />
        );

        await new Promise(resolve => setTimeout(resolve, 10));

        expect(lastFrame()).toContain('Status: disconnected');
        // When code is null, message is just "Engine exited" without code
        expect(lastFrame()).toContain('Engine exited');
    });

    it('handles clean engine exit (code 0)', async () => {
        const { lastFrame } = render(
            <TestComponent
                client={client}
                repoPath="/repo/path"
                engineExited={{ code: 0, signal: null }}
            />
        );

        await new Promise(resolve => setTimeout(resolve, 10));

        expect(lastFrame()).toContain('Status: disconnected');
        expect(lastFrame()).toContain('Engine exited (code 0)');
    });

    // Note: Testing async state updates and event callbacks is complex with mocked hooks.
    // We are primarily testing the initial state and effect logic here.
    // Integration tests would be better for full flow verification.
});
