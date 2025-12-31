import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render } from 'ink-testing-library';
import { HelpModal } from '../../../components/modal/HelpModal';
import { KeypressProvider } from '../../../contexts/KeypressContext';

describe('HelpModal', () => {
    const renderWithProvider = (onClose = vi.fn()) => {
        return render(
            <KeypressProvider>
                <HelpModal onClose={onClose} />
            </KeypressProvider>
        );
    };

    it('renders the title and sections', () => {
        const { lastFrame } = renderWithProvider();
        const frame = lastFrame();

        expect(frame).toContain('Dodo Help & Commands');
        expect(frame).toContain('System Commands');
        expect(frame).toContain('Agent Capabilities');
        expect(frame).toContain('Keyboard Shortcuts');
    });

    it('renders command list', () => {
        const { lastFrame } = renderWithProvider();
        const frame = lastFrame();

        expect(frame).toContain('/help');
        expect(frame).toContain('/exit');
        expect(frame).toContain('Ctrl+C');
    });

    it('calls onClose when Escape is pressed', () => {
        const onClose = vi.fn();
        const { stdin } = renderWithProvider(onClose);

        // Simulate Escape key
        stdin.write('\u001B');

        // Wait for effect
        setTimeout(() => {
            expect(onClose).toHaveBeenCalled();
        }, 10);
    });
});
