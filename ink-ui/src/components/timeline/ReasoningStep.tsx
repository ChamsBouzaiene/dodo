import React from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the ReasoningStep component.
 */
type ReasoningStepProps = {
    /** The reasoning/thinking step data */
    step: TimelineStep;
};

export const ReasoningStep: React.FC<ReasoningStepProps> = ({ step }) => {
    // Check if this is a think tool with reasoning
    const isThinkTool = step.toolName === "think";
    const reasoning = step.metadata?.reasoning as string | undefined;

    // Check if this is a reasoning step (assistant thinking) - toolName is empty and has content
    const isReasoning = (!step.toolName || step.toolName === "") && step.metadata?.content;
    const reasoningContent = step.metadata?.content as string | undefined;

    if (isThinkTool && reasoning && step.status === "done") {
        return (
            <Box marginTop={1} marginLeft={1} marginBottom={1} flexDirection="column" borderStyle="round" borderColor="blue" paddingX={1}>
                <Box marginBottom={1}>
                    <Text color="blue" bold>Thinking Process:</Text>
                </Box>
                <Text color="white">
                    {reasoning}
                </Text>
            </Box>
        );
    }

    if (isReasoning && reasoningContent) {
        return (
            <Box marginTop={1} marginLeft={1} flexDirection="column">
                <Text color="blue" dimColor>
                    {reasoningContent}
                </Text>
            </Box>
        );
    }

    return null;
};
