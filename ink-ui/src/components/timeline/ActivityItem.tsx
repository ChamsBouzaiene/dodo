import { Box, Text } from "ink";
import { Spinner } from "../common/Spinner.js";
import type { Activity } from "../../types.js";

/**
 * Props for the ActivityItem component.
 */
type ActivityItemProps = {
  /** The activity/tool execution data */
  activity: Activity;
};

// Icon mapping for different tools
const getActivityIcon = (activity: Activity): string => {
  if (activity.type === "thinking" || activity.type === "reasoning") {
    return "🧠";
  }

  if (activity.type === "edit") {
    return "✏️";
  }

  // Tool-specific icons
  switch (activity.tool) {
    case "read_file":
    case "read_span":
      return "📖";
    case "grep":
    case "codebase_search":
      return "🔍";
    case "search_replace":
    case "propose_diff":
    case "write":
      return "✏️";
    case "run_terminal_cmd":
      return "⚙️";
    case "think":
      return "🧠";
    default:
      return "🔧";
  }
};

// Format metadata for display
const formatMetadata = (activity: Activity): string => {
  const metadata = activity.metadata || {};
  const parts: string[] = [];

  // Add line numbers if available
  if (metadata.start_line && metadata.end_line) {
    parts.push(`L${metadata.start_line}-${metadata.end_line}`);
  } else if (metadata.start_line) {
    parts.push(`L${metadata.start_line}`);
  }

  // Add pattern for search operations
  if (metadata.pattern) {
    parts.push(`"${metadata.pattern}"`);
  }

  // Add query for codebase_search
  if (metadata.query) {
    parts.push(`"${metadata.query}"`);
  }

  // Add command for terminal
  if (metadata.command) {
    parts.push(`"${metadata.command}"`);
  }

  // Add result size if available
  if (metadata.result_size) {
    const kb = Math.round(metadata.result_size / 1024);
    if (kb > 0) {
      parts.push(`${kb}KB`);
    }
  }

  return parts.length > 0 ? `(${parts.join(", ")})` : "";
};

export const ActivityItem: React.FC<ActivityItemProps> = ({ activity }) => {
  const icon = getActivityIcon(activity);
  const metadata = formatMetadata(activity);

  // Status-based coloring
  let color = "gray";
  if (activity.status === "completed") {
    color = "green";
  } else if (activity.status === "failed") {
    color = "red";
  }

  return (
    <Box>
      {activity.status === "active" && (
        <Box marginRight={1}>
          <Spinner />
        </Box>
      )}
      <Text color={color}>
        {icon} {activity.tool || activity.type}
      </Text>
      {activity.target && (
        <Text color={color}> {activity.target}</Text>
      )}
      {metadata && (
        <Text color="gray"> {metadata}</Text>
      )}
    </Box>
  );
};


