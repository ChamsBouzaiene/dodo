import { lexer } from "marked";
import type { Turn, TimelineStep } from "../types.js";
import type { RenderItem } from "./flatten.js";
import { UI_CONFIG } from "../config.js";
import type { VirtualItem } from "./flatten.js";

/**
 * Type definition for a tool height estimator function.
 * Takes a step and returns the estimated height in lines.
 */
type ToolEstimator = (step: TimelineStep, width: number) => number;

/**
 * Helper to estimate height of wrapped text.
 */
function estimateTextLines(text: string | undefined, width: number = 80): number {
  if (!text) return 0;
  const safeText = String(text);

  const explicitLines = safeText.split('\n');
  let totalLines = 0;

  for (const line of explicitLines) {
    // Each line wraps based on width
    totalLines += Math.max(1, Math.ceil(line.length / width));
  }

  return totalLines;
}

/**
 * Estimates height `run_cmd` tool (ExecutionStep).
 */
const estimateRunCmd: ToolEstimator = (step, width) => {
  const config = UI_CONFIG.TOOLS.RUN_CMD;
  // Base height: Header + Borders + Inner Margin
  let height = config.HEADER_HEIGHT + config.BORDER_HEIGHT + config.MARGIN_INNER;

  const stdout = step.metadata?.stdout as string | undefined;
  const stderr = step.metadata?.stderr as string | undefined;
  const exitCode = step.metadata?.exit_code as number | undefined;

  // Add height for stdout (capped)
  if (stdout) {
    const textHeight = estimateTextLines(stdout, width); // Use actual width
    // If truncated, add 1 line for the "... (N lines hidden)" message
    const truncatedHeight = Math.min(config.MAX_OUTPUT_LINES, textHeight);
    height += truncatedHeight;
    if (textHeight > config.MAX_OUTPUT_LINES) {
      height += 1;
    }
  }

  // Add height for stderr (capped)
  if (stderr) {
    if (stdout) {
      height += 2; // Gap
    }

    const textHeight = estimateTextLines(stderr, width);
    const truncatedHeight = Math.min(config.MAX_OUTPUT_LINES, textHeight);
    height += truncatedHeight;
    if (textHeight > config.MAX_OUTPUT_LINES) {
      height += 1;
    }
  }

  // If running or no output, we show a placeholder line
  if (!stdout && !stderr) {
    height += 1;
  }

  // Footer with exit code
  if (exitCode !== undefined) {
    height += config.FOOTER_HEIGHT;
  }

  return height;
};

/**
 * Estimates height for 'think' tool (ReasoningStep).
 */
const estimateThink: ToolEstimator = (step, width) => {
  const config = UI_CONFIG.TOOLS.THINK;
  // Base height: Header + Borders + Inner Margin
  let height = config.HEADER_HEIGHT + config.BORDER_HEIGHT + config.MARGIN_INNER;

  const reasoning = step.metadata?.reasoning as string | undefined;
  if (reasoning) {
    height += estimateTextLines(reasoning, width);
  }

  return height;
};

/**
 * Estimates height for 'read_file' tool (ReadFileStep).
 */
const estimateReadFile: ToolEstimator = (step, width) => {
  const config = UI_CONFIG.TOOLS.READ_FILE;
  let height = config.HEADER_HEIGHT + config.BORDER_HEIGHT + config.MARGIN_INNER;

  const content = step.metadata?.content as string | undefined;
  if (content) {
    const textHeight = estimateTextLines(content, width);
    const truncatedHeight = Math.min(config.MAX_CONTENT_LINES, textHeight);
    height += truncatedHeight;
    if (textHeight > config.MAX_CONTENT_LINES) {
      height += 1;
    }
  } else {
    height += 1;
  }

  return height;
};

/**
 * Estimates height for 'respond' tool (RespondStep).
 */
const estimateRespond: ToolEstimator = (step, width) => {
  let height = 1; // Summary line

  const summary = step.metadata?.summary as string | undefined;
  if (summary) {
    height += estimateTextLines(summary, width) - 1;
  }

  const files = step.metadata?.files_changed as string[] | undefined;
  if (files && files.length > 0) {
    height += 1;
  }

  return height;
};

/**
 * Estimates height for structured 'plan' tool.
 */
const estimatePlan: ToolEstimator = (step, width) => {
  // Base height: Header + Borders + padding
  let height = 1 + 2 + 1;

  const summary = step.metadata?.plan_summary as string | undefined;
  const steps = step.metadata?.plan_steps as any[] | undefined;
  const targetAreas = step.metadata?.plan_target_areas as string[] | undefined;
  const risks = step.metadata?.plan_risks as string[] | undefined;

  const innerWidth = width - 4; // Accounting for border and paddingX(1)

  if (summary) {
    height += 1; // "Plan Summary:"
    height += estimateTextLines(summary, innerWidth);
    height += 1; // marginBottom
  }

  if (steps && steps.length > 0) {
    height += 1; // "Steps (N):"
    for (const s of steps) {
      const desc = s.description || "";
      const files = s.target_files || [];
      height += estimateTextLines(desc, innerWidth - 3); // -3 for "1. " prefix
      if (files.length > 0) {
        height += estimateTextLines(`Files: ${files.join(", ")}`, innerWidth - 5);
      }
    }
    height += 1; // marginBottom
  }

  if (targetAreas && targetAreas.length > 0) {
    height += 1; // "Target Areas:"
    height += estimateTextLines(targetAreas.join(", "), innerWidth - 2);
    height += 1; // marginBottom
  }

  if (risks && risks.length > 0) {
    height += 1; // "Risks:"
    for (const risk of risks) {
      height += estimateTextLines(`⚠ ${risk}`, innerWidth - 2);
    }
    height += 1; // marginBottom
  }

  return height;
};

const estimateContext: ToolEstimator = (step, width) => {
  return UI_CONFIG.TOOLS.CONTEXT.HEIGHT;
};

const estimateDefault: ToolEstimator = (step) => {
  return 4; // Generic fallback
};

const TOOL_ESTIMATORS: Record<string, ToolEstimator> = {
  "run_cmd": estimateRunCmd,
  "run_tests": estimateRunCmd,
  "run_build": estimateRunCmd,
  "think": estimateThink,
  "read_file": estimateReadFile,
  "respond": estimateRespond,
  "plan": estimatePlan,
  "context_event": estimateContext,
};

/**
 * Estimates height of markdown content including formatting overhead
 */
function estimateMarkdownHeight(markdown: string | undefined, width: number): number {
  if (!markdown) return 0;

  try {
    const tokens = lexer(markdown);
    let lines = 0;

    for (const token of tokens) {
      if (token.type === 'code') {
        const codeLines = token.text.split('\n').length;
        lines += codeLines + 3;
      } else if (token.type === 'paragraph') {
        lines += estimateTextLines(token.text, width) + 1;
      } else if (token.type === 'list') {
        for (const item of token.items) {
          lines += estimateTextLines(item.text, width - 3);
        }
        lines += 1;
      } else if (token.type === 'heading') {
        lines += estimateTextLines(token.text, width) + 2;
      } else if (token.type === 'space') {
        continue;
      } else {
        lines += estimateTextLines(token.raw, width);
      }
    }
    return lines;
  } catch (e) {
    return estimateTextLines(markdown, width);
  }
}

/**
 * Estimates height of a timeline step based on its content
 */
export function estimateTimelineStepHeight(step: TimelineStep, width: number): number {
  let estimator = estimateDefault;

  if (step.type === "context_event") {
    estimator = TOOL_ESTIMATORS["context_event"];
  } else if (step.toolName && TOOL_ESTIMATORS[step.toolName]) {
    estimator = TOOL_ESTIMATORS[step.toolName];
  }

  // We pass width to estimateTextLines implicitly via closures or we need to update estimators to take width
  // layout.ts estimators didn't take width, they assumed ~80.
  // For better accuracy we should probably propagate width, but layout.ts didn't.
  // Let's rely on the estimator logic which is now "config-aware" but maybe width-naive for now to match layout.ts behavior.

  // Actually, step.metadata often has text. estimateTextLines is called inside estimators.
  // I hardcoded 80 above. To be precise let's update tool estimators to use the width passed here?
  // For now, to keep it simple and consistent with `layout.ts` logic I ported, I'll use the hardcoded 80 behavior OR 
  // better, just return what the estimator says + margin.

  return estimator(step, width);
}

/**
 * Estimates height of a single virtual item
 */
let estimateCallCount = 0;
// setInterval(() => {
//   if (estimateCallCount > 0) {

//     estimateCallCount = 0;
//   }
// }, 1000);

export function estimateItemHeight(
  item: VirtualItem,
  terminalWidth: number = 80
): number {
  estimateCallCount++;
  // Reduce width slightly more to account for scrollbar to prevent jitter
  const contentWidth = Math.max(20, terminalWidth - 8);

  switch (item.type) {
    case 'user_message':
      return estimateTextLines(item.text, contentWidth);
    case 'label':
      return 1;
    case 'text_chunk':
      return estimateTextLines(item.text, contentWidth);
    case 'divider':
      return 1;
    case 'timeline_step':
      // Reuse estimateTimelineStepHeight but we need to account for margins defined in layout/config
      // layout.ts added MARGIN_Y. estimateTimelineStepHeight returns the step height itself.
      return estimateTimelineStepHeight(item.step, contentWidth) + UI_CONFIG.TOOLS.COMMON.MARGIN_Y;

    case 'assistant_message':
      return estimateTextLines(item.text, contentWidth);

    case 'code_change':
      // Estimate height for code change (placeholder logic or detailed)
      // Assuming simplistic height for now: header + few lines
      return 4;

    // Legacy or fallback
    default:
      return 1;
  }
}

/**
 * Estimates total height of a turn in terminal lines
 */
export function estimateTurnHeight(
  turn: Turn,
  timelineSteps: TimelineStep[],
  terminalWidth: number = 80
): number {
  // Account for padding/margins in content width
  const contentWidth = Math.max(20, terminalWidth - 6);
  let lines = 0;

  // "you" header
  lines += 1;

  // User message + marginBottom
  lines += estimateTextLines(turn.user, contentWidth);
  lines += 1;

  if (turn.collapsed) {
    // Collapsed state: summary + activity count (compact)
    if (turn.summary) {
      lines += Math.min(2, estimateTextLines(turn.summary, contentWidth));
    }
    if (timelineSteps.length > 0) {
      lines += 1; // Activity count line
    }
  } else {
    // Expanded state: timeline + assistant message + summary

    // Timeline steps
    for (const step of timelineSteps) {
      lines += estimateTimelineStepHeight(step, contentWidth) + UI_CONFIG.TOOLS.COMMON.MARGIN_Y;
    }
    if (timelineSteps.length > 0) {
      lines += 1; // marginBottom after timeline
    }

    // "assistant" header
    lines += 1;

    // Assistant message - use markdown estimation
    lines += estimateMarkdownHeight(turn.assistant || "...", contentWidth);
    lines += 1; // marginBottom

    // Summary if available
    if (turn.summary) {
      lines += estimateTextLines(turn.summary, contentWidth);
      lines += 1; // marginBottom
    }
  }

  // marginBottom on the turn container
  lines += 1;

  return lines;
}

/**
 * Calculate total height of all turns
 */
export function estimateTotalHeight(
  turns: Turn[],
  currentTimelineSteps: TimelineStep[],
  terminalWidth: number = 80
): number {
  let total = 0;
  const lastIdx = turns.length - 1;

  for (let i = 0; i < turns.length; i++) {
    const steps = i === lastIdx ? currentTimelineSteps : turns[i].timelineSteps;
    total += estimateTurnHeight(turns[i], steps, terminalWidth);
  }

  return total;
}
