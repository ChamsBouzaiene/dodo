import React from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the ExecutionStep component.
 */
type ExecutionStepProps = {
    /** The running/executed step data to display */
    step: TimelineStep;
};

import { truncateOutput } from "../../utils/formatting.js";

export const ExecutionStep: React.FC<ExecutionStepProps> = ({ step }) => {
    const stdout = step.metadata?.stdout as string | undefined;
    const stderr = step.metadata?.stderr as string | undefined;
    const exitCode = step.metadata?.exit_code as number | undefined;
    const command = step.command || step.toolName;

    // Determine status color based on exit code and step status
    const isSuccess = exitCode === 0 || (exitCode === undefined && step.status === "done");
    const isError = exitCode !== undefined && exitCode !== 0;
    const isRunning = step.status === "running";

    const borderColor = isError ? "red" : isSuccess ? "green" : isRunning ? "yellow" : "gray";
    const headerColor = isError ? "red" : isSuccess ? "green" : isRunning ? "yellow" : "gray";

    return (
        <Box marginTop={1} marginLeft={1} flexDirection="column">
            <Box
                flexDirection="column"
                borderStyle="round"
                borderColor={borderColor}
                paddingX={1}
            >
                {/* Terminal Header - shows the command */}
                <Box
                    marginBottom={1}
                    borderStyle="single"
                    borderBottom={true}
                    borderLeft={false}
                    borderRight={false}
                    borderTop={false}
                    borderColor={borderColor}
                    paddingBottom={1}
                >
                    <Text color={headerColor} bold>
                        ➜ {command}
                    </Text>
                </Box>

                {/* Output content */}
                {stdout && (
                    <Box marginBottom={stderr ? 1 : 0} flexDirection="column">
                        <Text color={isError ? "red" : "white"}>
                            {truncateOutput(stdout, 20)}
                        </Text>
                    </Box>
                )}
                {stderr && (
                    <Box flexDirection="column" marginTop={stdout ? 1 : 0}>
                        <Text color="red">
                            {truncateOutput(stderr, 20)}
                        </Text>
                    </Box>
                )}
                {!stdout && !stderr && isRunning && (
                    <Box>
                        <Text color="gray" dimColor>Running...</Text>
                    </Box>
                )}
                {!stdout && !stderr && !isRunning && (
                    <Box>
                        <Text color="gray" dimColor>(No output)</Text>
                    </Box>
                )}

                {/* Footer with exit code */}
                {exitCode !== undefined && (
                    <Box
                        marginTop={1}
                        borderStyle="single"
                        borderBottom={false}
                        borderLeft={false}
                        borderRight={false}
                        borderTop={true}
                        borderColor={borderColor}
                        paddingTop={1}
                    >
                        <Text color={isError ? "red" : "green"} bold>
                            Exit code: {exitCode}
                        </Text>
                    </Box>
                )}
            </Box>
        </Box>
    );
};
