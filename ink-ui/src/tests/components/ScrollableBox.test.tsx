import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'vitest';
import { Text } from 'ink';
import ScrollableBox from '../../components/common/ScrollableBox';

describe('ScrollableBox Component', () => {
    it('renders content correctly when fits', () => {
        const { lastFrame } = render(
            <ScrollableBox height={5}>
                <Text>Line 1</Text>
                <Text>Line 2</Text>
            </ScrollableBox>
        );
        expect(lastFrame()).toContain('Line 1');
        expect(lastFrame()).toContain('Line 2');
    });

    it('renders visible portion only when overflowing', () => {
        const items = Array.from({ length: 10 }, (_, i) => <Text key={i}>Line {i}</Text>);

        // Height 3: 1 line reserved for scrollbar/content? 
        // Logic: visibleHeight = Math.max(1, height - 1)
        // With height=4, visible=3.

        const { lastFrame } = render(
            <ScrollableBox height={4} autoScroll={false}>
                {items}
            </ScrollableBox>
        );

        // Initially at top (offset 0)
        expect(lastFrame()).toContain('Line 0');
        expect(lastFrame()).toContain('Line 1');
        expect(lastFrame()).toContain('Line 2');
        expect(lastFrame()).not.toContain('Line 3');
    });

    it('auto-scrolls to bottom by default', async () => {
        const items = Array.from({ length: 10 }, (_, i) => <Text key={i}>Line {i}</Text>);
        const { lastFrame } = render(
            <ScrollableBox height={4}>
                {items}
            </ScrollableBox>
        );

        // Should see last items
        // Height 4 -> 3 visible lines.
        // Last items: Line 7, 8, 9.
        await new Promise(r => setTimeout(r, 10));
        expect(lastFrame()).toContain('Line 9');
    });

    it('scrolls with arrow keys', async () => {
        const items = Array.from({ length: 10 }, (_, i) => <Text key={i}>Line {i}</Text>);
        const { lastFrame, stdin } = render(
            <ScrollableBox height={4} autoScroll={false}>
                {items}
            </ScrollableBox>
        );

        // Start at top: Line 0-2
        expect(lastFrame()).toContain('Line 0');

        // Scroll Down
        await new Promise(r => setTimeout(r, 10));
        stdin.write('\u001B[B'); // Down Arrow
        await new Promise(r => setTimeout(r, 10));

        // Offset 1: Line 1-3
        expect(lastFrame()).not.toContain('Line 0');
        expect(lastFrame()).toContain('Line 1');
    });

    it('scrolls to bottom with G', async () => {
        const items = Array.from({ length: 10 }, (_, i) => <Text key={i}>Line {i}</Text>);
        const { lastFrame, stdin } = render(
            <ScrollableBox height={4} autoScroll={false}>
                {items}
            </ScrollableBox>
        );

        expect(lastFrame()).toContain('Line 0');

        await new Promise(r => setTimeout(r, 10));
        stdin.write('G');
        await new Promise(r => setTimeout(r, 10));

        expect(lastFrame()).toContain('Line 9');
    });
});
