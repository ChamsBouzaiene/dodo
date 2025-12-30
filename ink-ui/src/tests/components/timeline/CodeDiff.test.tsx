import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { CodeDiff } from '../../../components/timeline/CodeDiff.js';
import type { CodeChange } from '../../../types.js';

describe('CodeDiff', () => {
    it('renders file name and line numbers', () => {
        const change: CodeChange = {
            file: 'utils.ts',
            startLine: 10,
            endLine: 15,
            before: '',
            after: ''
        };
        const { lastFrame } = render(<CodeDiff codeChange={change} />);
        expect(lastFrame()).toContain('utils.ts');
        expect(lastFrame()).toContain('L10-15');
    });

    it('renders diff content', () => {
        const change: CodeChange = {
            file: 'test.ts',
            startLine: 1,
            endLine: 2,
            before: 'old line',
            after: 'new line'
        };
        const { lastFrame } = render(<CodeDiff codeChange={change} />);
        expect(lastFrame()).toContain('- old line');
        expect(lastFrame()).toContain('+ new line');
    });
});
