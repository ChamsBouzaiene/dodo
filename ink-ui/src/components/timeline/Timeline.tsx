import React from "react";
import { Box, Text } from "ink";
import { TimelineItem } from "./TimelineItem.js";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the Timeline component.
 */
export type TimelineProps = {
  /** Array of timeline steps to display */
  steps: TimelineStep[];
  /** ID of the currently active invocation/step */
  currentInvocationId?: string;
};

export const Timeline: React.FC<TimelineProps> = React.memo(({ steps, currentInvocationId }) => {
  if (steps.length === 0) {
    return null;
  }

  return (
    <Box flexDirection="column">
      {steps.map((step) => (
        <TimelineItem
          key={step.id}
          step={step}
          isActive={step.id === currentInvocationId || step.status === "running"}
        />
      ))}
    </Box>
  );
}, (prevProps, nextProps) => {
  // fast path: if references are equal, do not re-render
  if (prevProps.steps === nextProps.steps && prevProps.currentInvocationId === nextProps.currentInvocationId) {
    return true;
  }

  if (prevProps.steps.length !== nextProps.steps.length) return false;
  if (prevProps.currentInvocationId !== nextProps.currentInvocationId) return false;

  // Shallow check of ids and statuses to avoid O(N) loop if possible, 
  // but if array ref changed and length is same, we assume something changed.
  // Ideally, useEngineConnection ensures stable step references.
  // For safety, we can check just the last item's status which is most likely to change?
  // Or just rely on re-rendering. React handle's 100 items easily if children are memoized.
  // The checking loop WAS the perf bottleneck if run frequently.
  // Removing it means we might re-render Timeline more often, but TimelineItem is memoized.
  return false;
});

