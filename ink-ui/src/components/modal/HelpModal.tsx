import React from 'react';
import { Box, Text, useInput } from 'ink';

/**
 * Props for the HelpModal component.
 */
export type HelpModalProps = {
    /** Callback to close the modal */
    onClose: () => void;
};

type HelpSection = {
    title: string;
    items: {
        command: string;
        description: string;
    }[];
};

const SECTIONS: HelpSection[] = [
    {
        title: 'System Commands',
        items: [
            { command: '/help', description: 'Show this help menu' },
            { command: '/exit', description: 'Exit the application' },
            { command: '/clear', description: 'Clear conversation history' },
            { command: '/stop', description: 'Stop current running task' },
        ]
    },
    {
        title: 'Agent Capabilities',
        items: [
            { command: 'Run Tests', description: 'Agent can run project tests' },
            { command: 'Edit Files', description: 'Agent can search & edit code' },
            { command: 'Search', description: 'Agent can search codebase' },
        ]
    },
    {
        title: 'Keyboard Shortcuts',
        items: [
            { command: 'Esc', description: 'Close modal / cancel task' },
            { command: 'Ctrl+C', description: 'Force exit application' },
            { command: '↑ / ↓', description: 'Navigate history' },
            { command: 'F1', description: 'Toggle Help' },
        ]
    }
];

export const HelpModal: React.FC<HelpModalProps> = ({ onClose }) => {
    useInput((_input, key) => {
        if (key.escape) {
            onClose();
        }
    }, { isActive: true });

    return (
        <Box
            borderStyle="double"
            borderColor="cyan"
            paddingX={2}
            paddingY={1}
            flexDirection="column"
            alignSelf="center"
        >
            <Box marginBottom={1} justifyContent="center">
                <Text bold color="cyan">Dodo Help & Commands</Text>
            </Box>

            {SECTIONS.map((section, idx) => (
                <Box key={section.title} flexDirection="column" marginBottom={idx === SECTIONS.length - 1 ? 0 : 1}>
                    <Text bold underline color="white">{section.title}</Text>
                    {section.items.map(item => (
                        <Box key={item.command} marginLeft={2}>
                            <Box width={15}>
                                <Text color="green">{item.command}</Text>
                            </Box>
                            <Text>{item.description}</Text>
                        </Box>
                    ))}
                </Box>
            ))}

            <Box marginTop={1} justifyContent="center" borderStyle="single" borderTop={false} borderLeft={false} borderRight={false} borderBottom={true} borderColor="gray">
                {/* Separator */}
            </Box>
            <Box marginTop={1} justifyContent="center">
                <Text color="gray">Press <Text bold color="white">ESC</Text> to close</Text>
            </Box>
        </Box>
    );
};
