import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { Timeline } from '../../../components/timeline/Timeline';
import { AnimationProvider } from '../../../contexts/AnimationContext.js';
import type { TimelineStep } from '../../../types';

describe('Timeline', () => {
    it('renders nothing when there are no steps', () => {
        const { lastFrame } = render(<Timeline steps={[]} />);
        expect(lastFrame()).toBe('');
    });

    it('renders multiple steps', () => {
        const steps: TimelineStep[] = [
            {
                id: '1',
                toolName: 'run_cmd',
                label: 'Step 1',
                status: 'done',
                startedAt: new Date(),
                type: 'tool',
                metadata: {}
            },
            {
                id: '2',
                toolName: 'think',
                label: 'Step 2',
                status: 'running',
                startedAt: new Date(),
                type: 'tool',
                metadata: {}
            }
        ];

        const { lastFrame } = render(
            <AnimationProvider>
                <Timeline steps={steps} />
            </AnimationProvider>
        );
        const frame = lastFrame();

        expect(frame).toContain('Step 1');
        expect(frame).toContain('Step 2');
    });
});
