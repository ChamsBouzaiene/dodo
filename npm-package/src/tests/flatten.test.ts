import { describe, it, expect } from 'vitest';
import { flattenTurns } from '../utils/flatten';
import type { Turn, TimelineStep } from '../types';

describe('flattenTurns', () => {
    it('flattens a simple turn', () => {
        const turn: Turn = {
            id: '1',
            user: 'Hello',
            assistant: 'Hi',
            done: true,
            collapsed: false,
            activities: [],
            timelineSteps: [],
            logLines: []
        };

        const items = flattenTurns([turn]);

        // Expected items:
        // 1. User Message (compact)
        // 2. Assistant Message
        // 3. Spacing Divider
        expect(items).toHaveLength(3);
        expect(items[0].type).toBe('user_message');
        expect(items[1].type).toBe('assistant_message');
        expect(items[2].type).toBe('divider');

        if (items[0].type === 'user_message') {
            expect(items[0].text).toBe('Hello');
        }
    });

    it('flattens a collapsed turn with summary', () => {
        const turn: Turn = {
            id: '1',
            user: 'Hello',
            assistant: 'Hi',
            done: true,
            collapsed: true,
            summary: 'Summary',
            activities: [],
            timelineSteps: [],
            logLines: []
        };

        const items = flattenTurns([turn]);

        // Expected items:
        // 1. User Message
        // 2. Summary (if exists)
        // 3. Status (if steps exist/running) -- here empty steps
        // 4. Spacing Divider? (Always added at end)

        // Turn has summary 'Summary', empty steps.
        // items: 
        // 1. User Message
        // 2. Text Chunk (Summary)
        // 3. No status because empty steps? No, logic says `if (steps.length > 0)`.
        // 4. Spacing Divider? (Always added at 146)

        // So 3 items. I'll assert 3.
        expect(items).toHaveLength(3); // User + Summary + Divider
        expect(items[1].type).toBe('text_chunk');
        if (items[1].type === 'text_chunk') {
            expect(items[1].text).toContain('Summary');
        }
    });

    it('flattens a turn with timeline steps', () => {
        const step: TimelineStep = {
            id: 's1',
            toolName: 'run_cmd',
            label: 'ls',
            status: 'done',
            startedAt: new Date()
        };

        const turn: Turn = {
            id: '1',
            user: 'Run',
            assistant: 'Done',
            done: true,
            collapsed: false,
            activities: [],
            timelineSteps: [step],
            logLines: []
        };

        const items = flattenTurns([turn]);

        // Expected items:
        // 1. User Message
        // 2. Assistant Message
        // 3. Timeline Step (s1)
        // 4. Divider (After steps)
        // 5. Spacing Divider
        expect(items).toHaveLength(5);
        const stepItem = items.find(i => i.type === 'timeline_step');
        expect(stepItem).toBeDefined();
        if (stepItem && stepItem.type === 'timeline_step') {
            expect(stepItem.step.id).toBe('s1');
        }
    });

    it('uses provided timeline steps for last turn', () => {
        const turn: Turn = {
            id: '1',
            user: 'Run',
            assistant: 'Done',
            done: false,
            collapsed: false,
            activities: [],
            timelineSteps: [], // Empty in turn
            logLines: []
        };

        const activeStep: TimelineStep = {
            id: 's2',
            toolName: 'run_cmd',
            label: 'ls',
            status: 'running',
            startedAt: new Date()
        };

        const items = flattenTurns([turn], [activeStep]);

        const stepItem = items.find(i => i.type === 'timeline_step');
        expect(stepItem).toBeDefined();
        if (stepItem && stepItem.type === 'timeline_step') {
            expect(stepItem.step.id).toBe('s2');
        }
    });
});
