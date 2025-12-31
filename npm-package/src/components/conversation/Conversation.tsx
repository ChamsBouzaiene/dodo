/**
 * Conversation - Simple conversation view
 * 
 * Renders all conversation items directly without virtualization.
 */
import React, { useMemo } from "react";
import { Box, Text } from "ink";
import type { TimelineStep } from "../../types.js";
import { flattenTurn, RenderItem } from "../../utils/flatten.js";
import { TimelineItem } from "../timeline/TimelineItem.js";
import { CodeDiff } from "../timeline/CodeDiff.js";
import { FormattedText } from "../common/FormattedText.js";
import { useConversation } from "../../hooks/useConversation.js";

/**
 * Props for the Conversation component.
 */
export type ConversationProps = {
    /** Whether the engine is currently processing/generating */
    isRunning: boolean;
    /** Timeline steps for the current turn being processed */
    timelineSteps?: TimelineStep[];
    /** ID of the currently running timeline step */
    currentRunningStepId?: string;
};

export const Conversation: React.FC<ConversationProps> = ({
    isRunning,
    timelineSteps = [],
    currentRunningStepId,
}) => {
    const { turns } = useConversation();

    // Cache of flattened items per turn
    const turnCacheRef = React.useRef<Map<string, RenderItem[]>>(new Map());

    // Flatten turns into render items
    const items = useMemo(() => {
        const result: RenderItem[] = [];
        turns.forEach((turn, index) => {
            const isLastTurn = index === turns.length - 1;
            let turnItems: RenderItem[];

            if (isLastTurn || !turnCacheRef.current.has(turn.id)) {
                const steps = isLastTurn && timelineSteps.length > 0 ? timelineSteps : turn.timelineSteps;
                turnItems = flattenTurn(turn, steps, currentRunningStepId);

                if (turn.done && !isLastTurn) {
                    turnCacheRef.current.set(turn.id, turnItems);
                }
            } else {
                turnItems = turnCacheRef.current.get(turn.id)!;
            }
            result.push(...turnItems);
        });
        return result;
    }, [turns, timelineSteps, currentRunningStepId]);

    // Empty state
    if (turns.length === 0) {
        return (
            <Box flexDirection="column">
                <Text color="gray">Describe a task below to get started.</Text>
            </Box>
        );
    }

    return (
        <Box flexDirection="column">
            {items.map((item) => (
                <RenderItemComponent
                    key={item.id}
                    item={item}
                    currentRunningStepId={currentRunningStepId}
                />
            ))}
        </Box>
    );
};

// Helper component to render individual items
const RenderItemComponent: React.FC<{
    item: RenderItem;
    currentRunningStepId?: string;
}> = React.memo(({ item, currentRunningStepId }) => {
    switch (item.type) {
        case 'user_message':
            return (
                <Box flexDirection="column">
                    <Box>
                        <Text color="gray" bold>{item.prefix} </Text>
                        <Text color="gray">{item.text.split('\n')[0]}</Text>
                    </Box>
                    {item.text.split('\n').slice(1).map((line, i) => (
                        <Box key={i} marginLeft={item.prefix.length + 1}>
                            <Text color="gray">{line}</Text>
                        </Box>
                    ))}
                </Box>
            );
        case 'label':
            return (
                <Box>
                    <Text color={item.color} bold dimColor={item.dimColor}>{item.text}</Text>
                </Box>
            );
        case 'text_chunk':
            return (
                <Box marginLeft={1}>
                    {item.dimColor ? (
                        <Text color="gray" dimColor>{item.text}</Text>
                    ) : (
                        <FormattedText content={item.text} />
                    )}
                </Box>
            );
        case 'divider':
            return null;
        case 'timeline_step':
            return (
                <TimelineItem
                    step={item.step}
                    isActive={item.step.id === currentRunningStepId || item.step.status === "running"}
                />
            );
        case 'assistant_message':
            return (
                <Box flexDirection="column">
                    <Box>
                        <Text color="green" bold>{item.prefix} </Text>
                        <FormattedText content={item.text.split('\n')[0]} />
                    </Box>
                    {item.text.split('\n').length > 1 && (
                        <Box flexDirection="column" marginLeft={item.prefix.length + 1}>
                            <FormattedText content={item.text.split('\n').slice(1).join('\n')} />
                        </Box>
                    )}
                </Box>
            );
        case 'code_change':
            return (
                <Box paddingX={1}>
                    <CodeDiff codeChange={item.codeChange} />
                </Box>
            );
        default:
            return null;
    }
});

export default Conversation;
