import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { Text } from 'ink';
import { ErrorBoundary } from '../../components/common/ErrorBoundary';
import { logger } from '../../utils/logger';

// Mock logger
vi.mock('../../utils/logger', () => ({
    logger: {
        error: vi.fn(),
    },
}));

describe('ErrorBoundary Component', () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it('renders children when no error', () => {
        const { lastFrame } = render(
            <ErrorBoundary>
                <Text>Safe Content</Text>
            </ErrorBoundary>
        );
        expect(lastFrame()).toContain('Safe Content');
    });

    it('catches error and renders fallback UI', () => {
        const ThrowError = () => {
            throw new Error('Test Crash');
        };

        // Suppress console.error for this test as React logs errors
        const consoleError = console.error;
        console.error = vi.fn();

        const { lastFrame } = render(
            <ErrorBoundary>
                <ThrowError />
            </ErrorBoundary>
        );

        console.error = consoleError;

        expect(lastFrame()).toContain('Something went wrong');
        expect(lastFrame()).toContain('Test Crash');
        expect(logger.error).toHaveBeenCalledWith(expect.stringContaining('Uncaught error'), expect.any(Object));
    });
});
