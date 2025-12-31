import React from "react";
import { Box, Text } from "ink";
import { FormattedText } from "../common/FormattedText.js";

/**
 * Props for the ProjectPlan component.
 */
type ProjectPlanProps = {
    /** The project plan content to display */
    content: string;
    /** Whether the plan is visible */
    visible: boolean;
};

export const ProjectPlan: React.FC<ProjectPlanProps> = ({ content, visible }) => {
    if (!visible || !content) {
        return null;
    }

    return (
        <Box
            borderStyle="round"
            borderColor="cyan"
            flexDirection="column"
            paddingX={1}
            marginBottom={1}
        >
            <Box marginBottom={1}>
                <Text color="cyan" bold>
                    📋 Project Plan
                </Text>
            </Box>
            <Box>
                <FormattedText content={content} />
            </Box>
        </Box>
    );
};
