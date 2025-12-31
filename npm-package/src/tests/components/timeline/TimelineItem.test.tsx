import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'vitest';
import { TimelineItem } from '../../../components/timeline/TimelineItem.js';
import { AnimationProvider } from '../../../contexts/AnimationContext.js';
import { ExecutionStep } from '../../../components/timeline/ExecutionStep.js';
import { ReasoningStep } from '../../../components/timeline/ReasoningStep.js';
import { ContextStep } from '../../../components/timeline/ContextStep.js';
import { RespondStep } from '../../../components/timeline/RespondStep.js';
import { ReadFileStep } from '../../../components/timeline/ReadFileStep.js';
import type { TimelineStep } from '../../../types.js';

describe('TimelineItem Components', () => {
    describe('ExecutionStep', () => {
        it('renders running state correctly', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                label: 'run_cmd',
                status: 'running',
                startedAt: new Date(),
                type: 'tool',
                metadata: {}
            };
            const { lastFrame } = render(<ExecutionStep step={step} />);
            expect(lastFrame()).toContain('Running...');
        });

        it('renders stdout correctly', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                label: 'run_cmd',
                status: 'done',
                startedAt: new Date(),
                type: 'tool',
                metadata: { stdout: 'Hello World' }
            };
            const { lastFrame } = render(<ExecutionStep step={step} />);
            expect(lastFrame()).toContain('Hello World');
        });

        it('renders stderr correctly', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                label: 'run_cmd',
                status: 'failed',
                startedAt: new Date(),
                type: 'tool',
                metadata: { stderr: 'Error occurred' }
            };
            const { lastFrame } = render(<ExecutionStep step={step} />);
            expect(lastFrame()).toContain('Error occurred');
        });
    });

    describe('ReasoningStep', () => {
        it('renders thinking process', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'think',
                label: 'think',
                status: 'done',
                startedAt: new Date(),
                type: 'tool',
                metadata: { reasoning: 'I am thinking...' }
            };
            const { lastFrame } = render(<ReasoningStep step={step} />);
            expect(lastFrame()).toContain('Thinking Process:');
            expect(lastFrame()).toContain('I am thinking...');
        });
    });

    describe('ContextStep', () => {
        it('renders context savings', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'context',
                label: 'Context Update',
                status: 'done',
                startedAt: new Date(),
                type: 'context_event',
                contextEventType: 'summarize',
                metadata: { before: 1000, after: 500 }
            };
            const { lastFrame } = render(<ContextStep step={step} />);
            expect(lastFrame()).toContain('🧠 Context Update');
            expect(lastFrame()).toContain('saved 500 tokens');
            expect(lastFrame()).toContain('50%');
        });
    });

    describe('RespondStep', () => {
        it('renders final response summary', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'respond',
                label: 'respond',
                status: 'done',
                startedAt: new Date(),
                type: 'tool',
                metadata: { summary: 'Task completed' }
            };
            const { lastFrame } = render(<RespondStep step={step} />);
            expect(lastFrame()).toContain('✔');
            expect(lastFrame()).toContain('Task completed');
        });
    });

    describe('ReadFileStep', () => {
        it('renders file reading status', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'read_file',
                label: 'read_file',
                status: 'done',
                startedAt: new Date(),
                type: 'tool',
                metadata: { file_path: '/path/to/file.txt' }
            };
            const { lastFrame } = render(
                <AnimationProvider>
                    <ReadFileStep step={step} />
                </AnimationProvider>
            );
            expect(lastFrame()).toContain('Reading');
            expect(lastFrame()).toContain('file.txt');
        });
    });

    describe('TimelineItem Integration', () => {
        it('renders ExecutionStep for run_cmd', () => {
            const step: TimelineStep = {
                id: '1',
                toolName: 'run_cmd',
                label: 'run_cmd',
                status: 'running',
                startedAt: new Date(),
                type: 'tool',
                metadata: {}
            };
            const { lastFrame } = render(
                <AnimationProvider>
                    <TimelineItem step={step} />
                </AnimationProvider>
            );
            expect(lastFrame()).toContain('Running...');
        });
    });
});
