import React, { useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { debugLog } from '../utils/debugLogger.js';

export interface ProjectPermissionPromptProps {
    /** The repository path requiring permission */
    repoRoot: string;
    /** Callback to send the permission response */
    onResponse: (enabled: boolean) => void;
}

/**
 * ProjectPermissionPrompt displays a prompt asking the user whether to enable
 * semantic indexing for the current project.
 */
export const ProjectPermissionPrompt: React.FC<ProjectPermissionPromptProps> = ({
    repoRoot,
    onResponse,
}) => {
    const [isReady, setIsReady] = useState(false);

    useEffect(() => {
        debugLog.lifecycle('ProjectPermissionPrompt', 'mount', `repo=${repoRoot}`);
        // Delay input handling to prevent accidental keypresses
        const timer = setTimeout(() => setIsReady(true), 500);
        return () => {
            debugLog.lifecycle('ProjectPermissionPrompt', 'unmount');
            clearTimeout(timer);
        };
    }, [repoRoot]);

    useInput((input, key) => {
        if (!isReady) return;

        if (input === 'y' || input === 'Y' || key.return) {
            debugLog.command('ProjectPermissionPrompt', 'response', { enabled: true });
            onResponse(true);
        } else if (input === 'n' || input === 'N') {
            debugLog.command('ProjectPermissionPrompt', 'response', { enabled: false });
            onResponse(false);
        }
    });

    // Extract basename for cleaner display
    const projectName = repoRoot.split('/').pop() || repoRoot;

    return (
        <Box flexDirection="column" padding={1} borderStyle="round" borderColor="cyan">
            <Text bold color="cyan">Project Indexing Setup 📊</Text>
            <Box marginTop={1}>
                <Text>Would you like to enable semantic indexing for </Text>
                <Text bold color="white">{projectName}</Text>
                <Text>?</Text>
            </Box>

            <Box marginTop={1} flexDirection="column">
                <Text color="gray">This allows Dodo to understand your codebase deeply for:</Text>
                <Text color="gray">  • Semantic code search</Text>
                <Text color="gray">  • Context-aware suggestions</Text>
                <Text color="gray">  • Better code understanding</Text>
            </Box>

            <Box marginTop={1}>
                <Text color="gray">You can add custom rules in </Text>
                <Text color="yellow">.dodo/rules</Text>
            </Box>

            <Box marginTop={1}>
                <Text color="green" bold>(Y)</Text>
                <Text>es / </Text>
                <Text color="red" bold>(N)</Text>
                <Text>o</Text>
            </Box>
        </Box>
    );
};
