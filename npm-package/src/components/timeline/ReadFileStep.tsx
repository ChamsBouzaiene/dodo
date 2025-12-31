import React from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the ReadFileStep component.
 */
type ReadFileStepProps = {
    /** The read_file step data */
    step: TimelineStep;
};

export const ReadFileStep: React.FC<ReadFileStepProps> = ({ step }) => {
    const { status, metadata } = step;
    const isError = status === "failed" || metadata?.error;

    // Get file path from various possible sources
    const filePath =
        (metadata?.path as string) ||
        (metadata?.file_path as string) ||
        (metadata?.params as { file_path?: string; path?: string } | undefined)?.file_path ||
        (metadata?.params as { file_path?: string; path?: string } | undefined)?.path ||
        "Unknown file";

    // Extract just the filename for cleaner display
    const fileName = filePath.split('/').pop() || filePath;

    const statusIcon = isError ? "✗" : status === "running" ? "⋯" : "✓";
    const statusColor = isError ? "red" : status === "running" ? "gray" : "green";

    return (
        <Box>
            <Text color={statusColor}>{statusIcon} </Text>
            <Text color="gray">Reading </Text>
            <Text color="cyan">{fileName}</Text>
        </Box>
    );
};
