import React from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the RespondStep component.
 */
type RespondStepProps = {
    /** The response step data */
    step: TimelineStep;
};

export const RespondStep: React.FC<RespondStepProps> = ({ step }) => {
    const respondSummary = step.metadata?.summary as string | undefined;
    const respondFiles = step.metadata?.files_changed as string[] | undefined;

    if (step.status !== "done") return null;

    return (
        <Box marginTop={0} marginLeft={2} flexDirection="column">
            <Box>
                <Text color="green">✔ </Text>
                <Text color="white">{respondSummary || "Task completed"}</Text>
            </Box>
            {respondFiles && respondFiles.length > 0 && (
                <Box marginLeft={2}>
                    <Text color="gray" dimColor>
                        Files: {respondFiles.join(", ")}
                    </Text>
                </Box>
            )}
        </Box>
    );
};
