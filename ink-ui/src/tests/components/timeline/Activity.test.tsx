import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { ActivityItem } from '../../../components/timeline/ActivityItem.js';
import { ActivityLog } from '../../../components/timeline/ActivityLog.js';
import { ActivityStream } from '../../../components/timeline/ActivityStream.js';
import { AnimationProvider } from '../../../contexts/AnimationContext.js';
import type { Activity, LogLine } from '../../../types.js';

describe('Activity Components', () => {
    describe('ActivityItem', () => {
        it('renders basic activity info', () => {
            const activity: Activity = {
                id: '1',
                type: 'tool',
                tool: 'read_file',
                status: 'completed',
                target: 'file.txt',
                timestamp: new Date()
            };
            const { lastFrame } = render(<ActivityItem activity={activity} />);
            expect(lastFrame()).toContain('read_file');
            expect(lastFrame()).toContain('file.txt');
        });

        it('renders active spinner', () => {
            const activity: Activity = {
                id: '1',
                type: 'tool',
                tool: 'search',
                status: 'active',
                timestamp: new Date()
            };
            const { lastFrame } = render(
                <AnimationProvider>
                    <ActivityItem activity={activity} />
                </AnimationProvider>
            );
            expect(lastFrame()).toContain('⠋'); // Spinner char
        });
    });

    describe('ActivityLog', () => {
        it('renders error lines', () => {
            const lines: LogLine[] = [
                { id: '1', level: 'error', text: 'Something failed', timestamp: new Date(), source: 'system' }
            ];
            const { lastFrame } = render(<ActivityLog lines={lines} />);
            expect(lastFrame()).toContain('Errors (1)');
            expect(lastFrame()).toContain('Something failed');
            expect(lastFrame()).toContain('[ERROR]');
        });

        it('filters out non-error lines by default implementation logic', () => {
            // verifying the component logic which filters for error lines
            const lines: LogLine[] = [
                { id: '1', level: 'info', text: 'Just info', timestamp: new Date(), source: 'system' }
            ];
            const { lastFrame } = render(<ActivityLog lines={lines} />);
            expect(lastFrame()).toContain('Activity log empty');
        });
    });

    describe('ActivityStream', () => {
        it('renders list of activities', () => {
            const activities: Activity[] = [
                { id: '1', type: 'tool', tool: 'cmd1', status: 'completed', timestamp: new Date() },
                { id: '2', type: 'tool', tool: 'cmd2', status: 'completed', timestamp: new Date() }
            ];
            const { lastFrame } = render(<ActivityStream activities={activities} />);
            expect(lastFrame()).toContain('cmd1');
            expect(lastFrame()).toContain('cmd2');
        });
    });
});
