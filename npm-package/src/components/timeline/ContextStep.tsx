import React from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the ContextStep component.
 */
type ContextStepProps = {
    /** The context event step data */
    step: TimelineStep;
};

export const ContextStep: React.FC<ContextStepProps> = ({ step }) => {
    const contextBefore = step.metadata?.before as number | undefined;
    const contextAfter = step.metadata?.after as number | undefined;
    const contextSaved = contextBefore && contextAfter ? contextBefore - contextAfter : 0;
    const contextPercent = contextBefore && contextSaved ? Math.round((contextSaved / contextBefore) * 100) : 0;

    return (
        <Box flexDirection="row" alignItems="center">
            <Text color="gray">
                🧠 {step.label}
            </Text>
            {contextSaved > 0 && (
                <Box marginLeft={1}>
                    <Text color="dim" dimColor>
                        (saved {contextSaved} tokens, {contextPercent}%)
                    </Text>
                </Box>
            )}
        </Box>
    );
};
