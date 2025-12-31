import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'vitest';
import { StatusBadge } from '../../components/common/StatusBadge';
import { AnimationProvider } from '../../contexts/AnimationContext.js';

describe('StatusBadge Component', () => {
    it('renders READY status', () => {
        const { lastFrame } = render(<StatusBadge status="ready" />);
        expect(lastFrame()).toContain('READY');
    });

    it('renders RUNNING status with spinner for "thinking"', () => {
        const { lastFrame } = render(
            <AnimationProvider>
                <StatusBadge status="thinking" />
            </AnimationProvider>
        );
        expect(lastFrame()).toContain('RUNNING');
        // Typical spinner output in test env might be difficult to assert exactly frame-by-frame,
        // but we can check if it doesn't crash and renders the label.
        // Also Ink Spinner usually renders a char.
    });

    it('renders ERROR status', () => {
        const { lastFrame } = render(<StatusBadge status="error" />);
        expect(lastFrame()).toContain('ERROR');
    });

    it('renders message if provided', () => {
        const { lastFrame } = render(<StatusBadge status="ready" message="All good" />);
        expect(lastFrame()).toContain('READY');
        expect(lastFrame()).toContain('All good');
    });

    it('renders UNKNOWN for invalid status', () => {
        const { lastFrame } = render(<StatusBadge status="unknown_status" />);
        expect(lastFrame()).toContain('UNKNOWN');
    });
});
