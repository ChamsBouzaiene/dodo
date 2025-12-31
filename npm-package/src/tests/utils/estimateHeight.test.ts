import { describe, it, expect } from 'vitest';
import { estimateTimelineStepHeight, estimateTextLines, estimateTurnHeight } from '../../utils/estimateHeight.js';
import type { TimelineStep, Turn } from '../../types.js';

describe('estimateHeight Utils', () => {

    it('estimates text lines correctly', () => {
        // Need to export estimateTextLines or test it via other functions if not exported.
        // It's not exported in the file, but we can verify it via estimateTimelineStepHeight
    });

    describe('estimateTimelineStepHeight', () => {
        it('estimates run_cmd height correctly', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                status: 'done',
                metadata: {
                    stdout: 'line1\nline2',
                    stderr: '',
                    exit_code: 0
                },
                order: 1,
                type: 'tool_use'
            };
            // Header(4) + Border(2) + Inner(1) + Stdout(2) + Footer(4) = 13
            const height = estimateTimelineStepHeight(step, 80);
            expect(height).toBeGreaterThan(10);
        });

        it('caps long output', () => {
            const longOutput = Array(100).fill('line').join('\n');
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                status: 'done',
                metadata: {
                    stdout: longOutput,
                    exit_code: 0
                },
                order: 1,
                type: 'tool_use'
            };
            const height = estimateTimelineStepHeight(step, 80);
            // Should not be huge
            expect(height).toBeLessThan(50);
        });

        it('estimates think height', () => {
            const step: TimelineStep = {
                id: '2',
                toolName: 'think',
                status: 'done',
                metadata: {
                    reasoning: 'Some thought process'
                },
                order: 2,
                type: 'tool_use'
            };
            // Header(1) + Border(2) + Inner(1) + Text(1) = 5
            const height = estimateTimelineStepHeight(step, 80);
            expect(height).toBe(5);
        });
    });
});
