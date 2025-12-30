import React from 'react';
import { describe, it, expect } from 'vitest';
import { render } from 'ink-testing-library';
import { FormattedText } from '../../components/common/FormattedText.js';

describe('FormattedText', () => {
    it('renders plain text', () => {
        const { lastFrame } = render(<FormattedText content="Hello World" />);
        expect(lastFrame()).toContain('Hello World');
    });

    it('renders bold text', () => {
        // Ink testing library output might not show ansi codes easily in assertions
        // but we can check content
        const { lastFrame } = render(<FormattedText content="**Bold**" />);
        expect(lastFrame()).toContain('Bold');
    });

    it('renders code blocks', () => {
        const markdown = "```\nconst x = 1;\n```";
        const { lastFrame } = render(<FormattedText content={markdown} />);
        expect(lastFrame()).toContain('const x = 1;');
    });

    it('renders lists', () => {
        const markdown = "- Item 1\n- Item 2";
        const { lastFrame } = render(<FormattedText content={markdown} />);
        expect(lastFrame()).toContain('Item 1');
        expect(lastFrame()).toContain('Item 2');
    });
});
