import React, { useState, useEffect } from 'react';
import { Box, Text, useApp, useInput } from 'ink';
import fs from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';

type SessionMeta = {
    id: string;
    title: string;
    updatedAt: Date;
    summary?: string;
};

type SessionPickerProps = {
    repoPath: string;
    onSelect: (sessionId?: string) => void;
};

const formatDate = (date: Date) => {
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (days === 0) {
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else if (days === 1) {
        return 'Yesterday';
    } else {
        return date.toLocaleDateString();
    }
};

export const SessionPicker: React.FC<SessionPickerProps> = ({ repoPath, onSelect }) => {
    const [sessions, setSessions] = useState<SessionMeta[]>([]);
    const [selectedIndex, setSelectedIndex] = useState(0);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const loadSessions = async () => {
            try {
                const hash = crypto.createHash('sha256').update(path.resolve(repoPath)).digest('hex').substring(0, 12);
                const sessionDir = path.join(os.homedir(), '.dodo', 'sessions', hash);

                if (!fs.existsSync(sessionDir)) {
                    setSessions([]);
                    return;
                }

                const files = fs.readdirSync(sessionDir);
                const loadedSessions: SessionMeta[] = [];

                for (const file of files) {
                    if (!file.endsWith('.json')) continue;

                    try {
                        const content = fs.readFileSync(path.join(sessionDir, file), 'utf-8');
                        const data = JSON.parse(content);
                        loadedSessions.push({
                            id: data.id,
                            title: data.title || 'Untitled Session',
                            updatedAt: new Date(data.updated_at),
                            summary: data.summary
                        });
                    } catch (e) {
                        // Ignore individual file read errors
                    }
                }

                // Sort by UpdatedAt descending (newest first)
                loadedSessions.sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime());
                setSessions(loadedSessions);

                // Auto-start new session if no existing sessions
                if (loadedSessions.length === 0) {
                    onSelect(undefined);
                    return;
                }
            } catch (err: any) {
                // Ignore general errors
            } finally {
                setLoading(false);
            }
        };

        loadSessions();
    }, [repoPath, onSelect]);

    // Add "New Session" as the first option
    const options = [
        { id: 'new', title: '+ Start New Session', updatedAt: new Date(), summary: '' },
        ...sessions
    ];

    const { exit } = useApp();

    useInput((input, key) => {
        if (loading) return;

        if (key.upArrow) {
            setSelectedIndex(prev => Math.max(0, prev - 1));
        }
        if (key.downArrow) {
            setSelectedIndex(prev => Math.min(options.length - 1, prev + 1));
        }
        if (key.return) {
            if (options[selectedIndex]) {
                const selected = options[selectedIndex];
                onSelect(selected.id === 'new' ? undefined : selected.id);
            }
        }
        // Emergency exit
        if (input === 'q' || (key.ctrl && input === 'c')) {
            exit();
        }
    }, { isActive: !loading });

    if (loading) {
        return <Box padding={1}><Text>Loading sessions...</Text></Box>;
    }

    return (
        <Box flexDirection="column" padding={1} borderStyle="round" borderColor="blue">
            <Box marginBottom={1}>
                <Text bold color="cyan">Select a Session</Text>
            </Box>

            {options.map((option, index) => {
                const isSelected = index === selectedIndex;
                return (
                    <Box key={option.id} flexDirection="column" marginBottom={0}>
                        <Box>
                            <Text color={isSelected ? "green" : undefined}>
                                {isSelected ? "> " : "  "}
                            </Text>
                            <Text bold={isSelected} color={isSelected ? "white" : "gray"}>
                                {option.title}
                            </Text>
                            <Box marginLeft={2}>
                                {option.id !== 'new' && (
                                    <Text color="gray">
                                        ({formatDate(option.updatedAt)})
                                    </Text>
                                )}
                            </Box>
                        </Box>
                        {isSelected && option.summary && (
                            <Box marginLeft={2} marginBottom={1}>
                                <Text color="gray" italic>
                                    └─ {option.summary.length > 80 ? option.summary.substring(0, 80) + '...' : option.summary}
                                </Text>
                            </Box>
                        )}
                    </Box>
                );
            })}

            <Box marginTop={1}>
                <Text color="gray">Use UP/DOWN to navigate, ENTER to select</Text>
                <Text color="gray">Press 'q' or 'Ctrl+C' to quit</Text>
            </Box>
        </Box>
    );
};
