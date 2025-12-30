import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useEngineEvents } from '../hooks/useEngineEvents.js';
import { EngineClient } from '../engineClient.js';
import { Text } from 'ink';
import EventEmitter from 'events';

// Mock EngineClient
class MockEngineClient extends EventEmitter {
    startSession = vi.fn();
    sendUserMessage = vi.fn();
    close = vi.fn();
}

const TestComponent = ({ client, callbacks }: any) => {
    useEngineEvents(client, callbacks);
    return <Text>Listening...</Text>;
};

describe('useEngineEvents', () => {
    let client: any;
    let callbacks: any;

    beforeEach(() => {
        client = new MockEngineClient();
        callbacks = {
            onStatusChange: vi.fn(),
            onSessionReady: vi.fn(),
            onAssistantText: vi.fn(),
            onToolEvent: vi.fn(),
            onFilesChanged: vi.fn(),
            onDone: vi.fn(),
            onError: vi.fn(),
            onActivity: vi.fn(),
            onTokenUsage: vi.fn(),
            onProjectPlan: vi.fn(),
            onContext: vi.fn(),
            onToolOutput: vi.fn(),
        };
    });

    it('handles session_ready event', async () => {
        render(<TestComponent client={client} callbacks={callbacks} />);
        await new Promise(resolve => setTimeout(resolve, 0));

        client.emit('event', {
            type: 'status',
            status: 'session_ready',
            session_id: 'test-session-id'
        });

        expect(callbacks.onSessionReady).toHaveBeenCalledWith('test-session-id');
    });

    it('handles assistant_text event', async () => {
        render(<TestComponent client={client} callbacks={callbacks} />);
        await new Promise(resolve => setTimeout(resolve, 0));

        client.emit('event', {
            type: 'assistant_text',
            content: 'Hello',
            final: false
        });

        expect(callbacks.onAssistantText).toHaveBeenCalledWith('Hello', false, false);
    });

    it('handles error event', async () => {
        render(<TestComponent client={client} callbacks={callbacks} />);
        await new Promise(resolve => setTimeout(resolve, 0));

        client.emit('event', {
            type: 'error',
            message: 'Something went wrong'
        });

        expect(callbacks.onError).toHaveBeenCalledWith('Something went wrong');
        expect(callbacks.onStatusChange).toHaveBeenCalledWith('error', 'Something went wrong');
    });

    it('handles activity event', async () => {
        render(<TestComponent client={client} callbacks={callbacks} />);
        await new Promise(resolve => setTimeout(resolve, 0));

        client.emit('event', {
            type: 'activity',
            activity_id: 'act-1',
            activity_type: 'tool',
            tool: 'run_cmd',
            status: 'started',
            start_time: '2023-01-01T00:00:00Z'
        });

        expect(callbacks.onActivity).toHaveBeenCalledWith(expect.objectContaining({
            id: 'act-1',
            type: 'tool',
            tool: 'run_cmd',
            status: 'active'
        }));
    });
});
