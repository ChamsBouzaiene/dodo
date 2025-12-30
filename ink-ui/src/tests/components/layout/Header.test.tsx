import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { Header } from '../../../components/layout/Header';

describe('Header', () => {
    it('renders the ASCII art logo and tagline', () => {
        const { lastFrame } = render(<Header />);
        const frame = lastFrame();

        // Check for parts of ascii art
        expect(frame).toContain('█████');
        // Check for version
        expect(frame).toContain('v1.0.0');
        // Check for tagline
        expect(frame).toContain('The Agentic AI Coding Assistant');
    });
});
