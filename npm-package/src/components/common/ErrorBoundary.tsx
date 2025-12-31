import React, { Component, ErrorInfo, ReactNode } from "react";
import { Box, Text } from "ink";
import { logger } from "../../utils/logger.js";

interface Props {
    children: ReactNode;
}

interface State {
    hasError: boolean;
    error: Error | null;
}

/**
 * ErrorBoundary
 *
 * Catches JavaScript errors in its child component tree, logs those errors,
 * and displays a fallback UI instead of the component tree that crashed.
 */
export class ErrorBoundary extends Component<Props, State> {
    public state: State = {
        hasError: false,
        error: null,
    };

    public static getDerivedStateFromError(error: Error): State {
        return { hasError: true, error };
    }

    public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        logger.error("Uncaught error in UI:", {
            message: error.message,
            stack: error.stack,
            componentStack: errorInfo.componentStack
        });
    }

    public render() {
        if (this.state.hasError) {
            return (
                <Box flexDirection="column" padding={1} borderColor="red" borderStyle="round">
                    <Text color="red" bold>Something went wrong.</Text>
                    <Text color="red">{this.state.error?.message}</Text>
                    <Text color="gray">Check ui_debug.log for details.</Text>
                </Box>
            );
        }

        return this.props.children;
    }
}
