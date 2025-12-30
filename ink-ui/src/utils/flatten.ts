import type { Turn, TimelineStep, CodeChange } from "../types.js";
import { UI_CONFIG } from "../config.js";

export type RenderItem =
    | { type: 'user_message'; id: string; prefix: string; text: string }
    | { type: 'assistant_message'; id: string; prefix: string; text: string }
    | { type: 'label'; id: string; text: string; color: string; dimColor?: boolean }
    | { type: 'text_chunk'; id: string; text: string; color?: string; dimColor?: boolean }
    | { type: 'timeline_step'; id: string; step: TimelineStep; turnId: string }
    | { type: 'divider'; id: string }
    | { type: 'code_change'; id: string; codeChange: CodeChange };

/**
 * Flattens a list of turns into a granular list of render items.
 * This allows for smoother scrolling by breaking down large turns.
 */
export type VirtualItem = RenderItem;

export function flattenTurns(
    turns: Turn[],
    timelineSteps: TimelineStep[] = [],
    currentRunningStepId?: string
): RenderItem[] {
    const items: RenderItem[] = [];

    turns.forEach((turn, index) => {
        const isLastTurn = index === turns.length - 1;
        const steps = isLastTurn && timelineSteps.length > 0 ? timelineSteps : turn.timelineSteps;
        items.push(...flattenTurn(turn, steps, currentRunningStepId));
    });

    return items;
}

export function flattenTurn(
    turn: Turn,
    steps: TimelineStep[],
    currentRunningStepId?: string
): RenderItem[] {
    const items: RenderItem[] = [];
    const { PREFIXES, HIDDEN_TOOLS, SHOW_REASONING } = UI_CONFIG.CONVERSATION;

    // 1. User Message (compact format: "> message" on same line)
    items.push({
        type: 'user_message',
        id: `turn-${turn.id}-user`,
        prefix: PREFIXES.USER,
        text: turn.user
    });

    if (turn.collapsed) {
        // Collapsed State - show summary only
        if (turn.summary) {
            items.push({
                type: 'text_chunk',
                id: `turn-${turn.id}-summary`,
                text: `📝 ${turn.summary}`,
                color: 'gray'
            });
        }

        const activeCount = steps.filter(s => s.status === 'running').length;
        const completedCount = steps.filter(s => s.status === 'done').length;

        if (steps.length > 0) {
            items.push({
                type: 'text_chunk',
                id: `turn-${turn.id}-status`,
                text: `${activeCount > 0 ? `⏳ ${activeCount} running • ` : ''}✅ ${completedCount} completed`,
                color: 'gray'
            });
        }
    } else {
        // Expanded State

        // 2. Assistant Message (Show early as it often contains reasoning/intent)
        if (turn.assistant) {
            items.push({
                type: 'assistant_message',
                id: `turn-${turn.id}-assistant`,
                prefix: PREFIXES.ASSISTANT,
                text: turn.assistant
            });
        } else if (!turn.done) {
            items.push({
                type: 'assistant_message',
                id: `turn-${turn.id}-assistant-placeholder`,
                prefix: PREFIXES.ASSISTANT,
                text: '...'
            });
        }

        // 3. Timeline Steps with inline Code Changes
        const visibleSteps = steps.filter(step => {
            if (step.toolName && HIDDEN_TOOLS.includes(step.toolName)) return false;
            const isReasoning = (!step.toolName || step.toolName === "") && step.metadata?.content;
            const isThinking = step.toolName === "think" && step.metadata?.reasoning;
            if (!SHOW_REASONING && (isReasoning || isThinking)) return false;
            return true;
        });

        visibleSteps.forEach(step => {
            items.push({
                type: 'timeline_step',
                id: `step-${step.id}`,
                step,
                turnId: turn.id
            });

            // Inline code changes for this step if it's an edit tool
            if (step.toolName === 'edit') {
                // Try to find matching edit activity by invocationId or just matching the file
                // For now, let's check all activities for this turn that have a codeChange
                const matchingEdit = turn.activities?.find(a =>
                    a.type === 'edit' &&
                    a.codeChange &&
                    (a.invocationId === step.id || (step.metadata?.path && a.codeChange.file === step.metadata.path))
                );

                if (matchingEdit?.codeChange) {
                    items.push({
                        type: 'code_change',
                        id: `edit-inline-${matchingEdit.id}`,
                        codeChange: matchingEdit.codeChange
                    });
                }
            }
        });

        if (visibleSteps.length > 0) {
            items.push({ type: 'divider', id: `turn-${turn.id}-divider-steps` });
        }

        // 4. Summary (if exists)
        if (turn.summary) {
            items.push({
                type: 'text_chunk',
                id: `turn-${turn.id}-summary-footer`,
                text: turn.summary,
                color: 'gray'
            });
        }
    }

    // Spacing between turns
    items.push({ type: 'divider', id: `turn-${turn.id}-end-spacing` });

    return items;
}
