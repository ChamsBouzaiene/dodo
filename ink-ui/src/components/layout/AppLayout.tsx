import React, { useRef, useCallback } from "react";
import { Box, Text, useApp, useInput } from "ink";
import { Header } from "./Header.js";
import { Footer, type FooterProps } from "./Footer.js";
import { Conversation } from "../conversation/Conversation.js";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the AppLayout component.
 */
type AppLayoutProps = {
    /** Total available height in terminal rows */
    terminalRows: number;
    /** Total available width in terminal columns */
    terminalColumns: number;
    /** Optional error message to display at the top */
    error?: string;
    /** Whether to split the view and show the project plan */
    showProjectPlan: boolean;
    /** Content of the project plan */
    projectPlan: string;
    /** Whether the engine is active/processing */
    isRunning: boolean;
    /** ID of the currently running timeline step */
    currentRunningStepId?: string;
    /** Timeline steps for the current active turn */
    currentTimelineSteps: TimelineStep[];
    /** Props passed down to the Footer component */
    footerProps: FooterProps;
    /** Handler to cancel the current request */
    onCancelRequest?: () => void;
    /** content of the help modal if active */
    helpModal?: React.ReactNode;
    /** Handler to toggle help modal */
    onHelp?: () => void;
};

export const AppLayout: React.FC<AppLayoutProps> = React.memo(({
    terminalRows,
    terminalColumns,
    error,
    showProjectPlan,
    projectPlan,
    currentTimelineSteps,
    currentRunningStepId,
    isRunning,
    footerProps,
    onCancelRequest,
    helpModal,
    onHelp,
}) => {
    const { exit } = useApp();

    const handlerRef = useRef({
        exit,
        isRunning,
        onCancelRequest,
        onHelp,
        helpModal: !!helpModal
    });
    handlerRef.current = {
        exit,
        isRunning,
        onCancelRequest,
        onHelp,
        helpModal: !!helpModal
    };

    useInput((input, key) => {
        const { exit, isRunning, onCancelRequest, onHelp, helpModal } = handlerRef.current;

        if (key.ctrl && input === 'c') {
            exit();
            return;
        }

        if (key.escape) {
            if (helpModal) {
                return;
            }

            if (onCancelRequest) {
                onCancelRequest();
            }
            return;
        }

        if (input === '\u001bOP') { // F1
            onHelp?.();
            return;
        }
    }, { isActive: true });

    return (
        <Box flexDirection="column">
            {/* Header */}
            <Box paddingX={1} flexShrink={0}>
                <Header />
            </Box>

            {/* Error display */}
            {error && (
                <Box paddingX={1}>
                    <Box borderStyle="round" borderColor="red" paddingX={1}>
                        <Text color="red">Error: {error}</Text>
                    </Box>
                </Box>
            )}

            {/* Conversation - uses Static for completed turns, natural scrolling */}
            <Box flexGrow={1} flexDirection="column" paddingX={1}>
                <Conversation
                    isRunning={isRunning}
                    timelineSteps={currentTimelineSteps}
                    currentRunningStepId={currentRunningStepId}
                />
            </Box>

            {/* Help modal overlay */}
            {helpModal && (
                <Box position="absolute" width="100%" height="100%" alignItems="center" justifyContent="center">
                    {helpModal}
                </Box>
            )}

            {/* Footer */}
            {!helpModal && (
                <Box flexShrink={0} paddingX={1}>
                    <Footer {...footerProps} />
                </Box>
            )}
        </Box>
    );
});
