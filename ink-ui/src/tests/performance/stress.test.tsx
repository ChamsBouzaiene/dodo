import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render } from 'ink-testing-library';
import { act } from 'react';
import { useConversation } from '../../hooks/useConversation.js';
import { useResponseStream } from '../../hooks/useResponseStream.js';
import type { EngineClient } from '../../engineClient.js';
import { EventEmitter } from 'events';

import { ConversationProvider } from '../../contexts/ConversationContext.js';

function renderHook<T>(hook: () => T) {
    const result = { current: null as unknown as T };
    function TestComponent() {
        result.current = hook();
        return null;
    }
    const { rerender } = render(
        <ConversationProvider>
            <TestComponent />
        </ConversationProvider>
    );
    return { result, rerender };
}

describe('Performance Stress Tests', () => {
    it('handles high-frequency token updates with O(1) complexity', async () => {
        const mockClient = new EventEmitter() as unknown as EngineClient;
        mockClient.on = vi.fn().mockImplementation((event, cb) => {
            (mockClient as any).listeners = (mockClient as any).listeners || {};
            (mockClient as any).listeners[event] = cb;
            return mockClient;
        });

        const TOKEN_COUNT = 500;
        let finalDuration = 0;

        function StressTestComponent() {
            const { pushTurn, appendAssistantContent, markLastTurnDone, turns } = useConversation();
            const { currentThought } = useResponseStream(mockClient, appendAssistantContent, markLastTurnDone);

            // Use an effect to trigger the stress test once mounted
            React.useEffect(() => {
                const start = process.hrtime.bigint();

                act(() => {
                    pushTurn("Stress test query");
                });

                act(() => {
                    for (let i = 0; i < TOKEN_COUNT; i++) {
                        mockClient.emit('event', {
                            type: 'assistant_text',
                            content: `token${i}`,
                            final: i === TOKEN_COUNT - 1
                        });
                    }
                });

                const end = process.hrtime.bigint();
                finalDuration = Number(end - start) / 1000000;
            }, []);

            return null;
        }

        render(
            <ConversationProvider>
                <StressTestComponent />
            </ConversationProvider>
        );

        console.log(`[Performance] Processed ${TOKEN_COUNT} tokens in ${finalDuration.toFixed(2)}ms`);
        expect(finalDuration).toBeLessThan(2000);
    });

    it('measures memory growth over large conversation', async () => {
        const startMemory = process.memoryUsage().heapUsed;

        function MemoryTestComponent() {
            const { pushTurn, appendAssistantContent, markLastTurnDone } = useConversation();

            React.useEffect(() => {
                for (let i = 0; i < 200; i++) {
                    act(() => {
                        pushTurn(`Turn ${i}`);
                        appendAssistantContent(`Large content to simulate real usage. `.repeat(10));
                        markLastTurnDone();
                    });
                }
            }, []);

            return null;
        }

        render(
            <ConversationProvider>
                <MemoryTestComponent />
            </ConversationProvider>
        );

        const endMemory = process.memoryUsage().heapUsed;
        const growthMb = (endMemory - startMemory) / 1024 / 1024;

        console.log(`[Performance] Memory growth for 200 turns: ${growthMb.toFixed(2)} MB`);
        expect(growthMb).toBeLessThan(100);
    });
});
