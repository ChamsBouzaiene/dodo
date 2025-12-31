import React from "react";
import { Box, Text } from "ink";
import { StatusBadge } from "../common/StatusBadge.js";
import { getToolConfig } from "../../utils/toolConfig.js";
import { PlanContent } from "./PlanContent.js";
import { ExecutionStep } from "./ExecutionStep.js";
import { ReasoningStep } from "./ReasoningStep.js";
import { ContextStep } from "./ContextStep.js";
import { RespondStep } from "./RespondStep.js";
import { ReadFileStep } from "./ReadFileStep.js";
import type { TimelineStep } from "../../types.js";



/**
 * Props for the TimelineItem component.
 */
export type TimelineItemProps = {
  /** The timeline step data to render */
  step: TimelineStep;
  /** Whether this step is currently active or running */
  isActive?: boolean;
};

/**
 * Renders a single step in the timeline (tool execution, context event, etc.)
 * Handles specific formatting for different tool types (plan, think, execute).
 */
export const TimelineItem: React.FC<TimelineItemProps> = React.memo(({ step, isActive = false }) => {
  const config = getToolConfig(step.toolName);

  // Format duration
  let durationText = "";
  if (step.durationMs !== undefined && step.durationMs > 0) {
    const seconds = (step.durationMs / 1000).toFixed(2);
    durationText = ` (${seconds}s)`;
  }

  // Check if this is a plan tool with plan content
  const isPlanTool = step.toolName === "plan";
  const planContent = step.metadata?.plan_content as string | undefined;
  const planSummary = step.metadata?.plan_summary as string | undefined;
  const planSteps = step.metadata?.plan_steps as any[] | undefined;
  const planTargetAreas = step.metadata?.plan_target_areas as string[] | undefined;
  const planRisks = step.metadata?.plan_risks as string[] | undefined;

  // Check if this is an execution tool with output
  const isExecutionTool = step.toolName === "run_cmd" || step.toolName === "run_tests" || step.toolName === "run_build";

  // Check if this is a think tool with reasoning
  const isThinkTool = step.toolName === "think";

  // Check if this is a reasoning step (assistant thinking) - toolName is empty and has content
  const isReasoning = (!step.toolName || step.toolName === "") && step.metadata?.content;

  // Extract error information for failed tools
  const errorMessage = step.metadata?.error as string | undefined;
  const errorResult = step.metadata?.error_result as string | undefined;
  const isFailed = step.status === "failed";
  const displayError = errorMessage || errorResult;

  // Check if this is a respond tool
  const isRespondTool = step.toolName === "respond";

  // Check if this is a read_file tool
  const isReadFileTool = step.toolName === "read_file";

  // Check if this is a context event
  const isContextEvent = step.type === "context_event";

  return (
    <Box
      flexDirection="column"
      marginLeft={1}
    >
      {/* Render context event */}
      {isContextEvent && <ContextStep step={step} />}

      {/* Render read_file tool with simplified display (no header) */}
      {isReadFileTool && <ReadFileStep step={step} />}

      {/* Standard header for non-context, non-read_file tools */}
      {!isContextEvent && !isReadFileTool && (
        <Box flexDirection="row" alignItems="center" justifyContent="space-between">
          <Box flexDirection="row" alignItems="center" flexGrow={1}>
            <Text color={config.color} bold>
              {config.icon}
            </Text>
            {step.toolName && (
              <Box marginLeft={1}>
                <Text color={config.color} bold>
                  [{step.toolName.toUpperCase()}]
                </Text>
              </Box>
            )}
            <Box marginLeft={1} flexGrow={1}>
              <Text>{step.label}</Text>
            </Box>
          </Box>
          <Box marginLeft={2}>
            <StatusBadge status={step.status} />
          </Box>
        </Box>
      )}

      {step.command && (
        <Box marginTop={0} marginLeft={1}>
          <Text color="dim" dimColor>
            $ {step.command}
          </Text>
        </Box>
      )}

      {/* Render plan content for plan tool */}
      {isPlanTool && step.status === "done" && (planContent || planSummary || planSteps) && (
        <Box marginTop={1} marginLeft={1} flexDirection="column">
          <PlanContent
            content={planContent}
            summary={planSummary}
            steps={planSteps}
            targetAreas={planTargetAreas}
            risks={planRisks}
          />
        </Box>
      )}

      {/* Render error message for failed tools */}
      {isFailed && displayError && (
        <Box marginTop={1} marginLeft={1} flexDirection="column">
          <Text color="red" bold>Error:</Text>
          <Box marginLeft={1}>
            <Text color="red">{displayError}</Text>
          </Box>
        </Box>
      )}

      {/* Render command output for execution tools (show even when failed) */}
      {isExecutionTool && <ExecutionStep step={step} />}

      {/* Render reasoning for think tool or assistant reasoning steps */}
      {(isThinkTool || isReasoning) && <ReasoningStep step={step} />}

      {/* Render respond tool content */}
      {isRespondTool && <RespondStep step={step} />}
    </Box>
  );
}, (prevProps, nextProps) => {
  // Only re-render if step data actually changed
  return (
    prevProps.step.id === nextProps.step.id &&
    prevProps.step.label === nextProps.step.label &&
    prevProps.step.status === nextProps.step.status &&
    prevProps.step.metadata === nextProps.step.metadata &&
    prevProps.step.finishedAt?.getTime() === nextProps.step.finishedAt?.getTime() &&
    prevProps.isActive === nextProps.isActive
  );
});

