import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { ProjectPlan } from '../../../components/timeline/ProjectPlan.js';
import { PlanContent } from '../../../components/timeline/PlanContent.js';

describe('Project Plan Components', () => {
    describe('ProjectPlan', () => {
        it('renders nothing when not visible', () => {
            const { lastFrame } = render(<ProjectPlan content="My Plan" visible={false} />);
            expect(lastFrame()).toBe('');
        });

        it('renders content when visible', () => {
            const { lastFrame } = render(<ProjectPlan content="My Plan Content" visible={true} />);
            expect(lastFrame()).toContain('Project Plan');
            expect(lastFrame()).toContain('My Plan Content');
        });
    });

    describe('PlanContent', () => {
        it('renders structured summary', () => {
            const { lastFrame } = render(<PlanContent summary="This is a summary" />);
            expect(lastFrame()).toContain('Plan Summary:');
            expect(lastFrame()).toContain('This is a summary');
        });

        it('renders steps', () => {
            const steps = [
                { id: '1', description: 'Step One', status: 'pending' as const },
                { id: '2', description: 'Step Two', status: 'completed' as const }
            ];
            const { lastFrame } = render(<PlanContent steps={steps} />);
            expect(lastFrame()).toContain('Step One');
            expect(lastFrame()).toContain('Step Two');
            expect(lastFrame()).toContain('✓'); // completed step
        });

        it('renders risks', () => {
            const risks = ['Risk 1', 'Risk 2'];
            const { lastFrame } = render(<PlanContent summary="Plan" risks={risks} />);
            expect(lastFrame()).toContain('Risks:');
            expect(lastFrame()).toContain('Risk 1');
            expect(lastFrame()).toContain('Risk 2');
        });

        it('renders raw content', () => {
            const { lastFrame } = render(<PlanContent content="Raw markdown plan" />);
            expect(lastFrame()).toContain('Raw markdown plan');
        });
    });
});
