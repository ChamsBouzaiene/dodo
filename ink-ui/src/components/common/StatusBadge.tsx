import { Box, Text } from "ink";
import { Spinner } from "./Spinner.js";
import type { TimelineStep } from "../../types.js";

/**
 * Props for the StatusBadge component.
 */
export type StatusBadgeProps = {
  /** The current status string (e.g., 'ready', 'thinking') */
  status: string;
  /** Optional message to display alongside the status */
  message?: string;
};

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, message }) => {
  let color: string;
  let label: string;
  let showSpinner = false;

  switch (status) {
    case "pending":
    case "booting":
    case "connecting":
      color = "gray";
      label = status.toUpperCase();
      break;
    case "ready":
      color = "green";
      label = "READY";
      break;
    case "running":
    case "thinking":
    case "running_tools":
      color = "yellow";
      label = "RUNNING";
      showSpinner = true;
      break;
    case "done":
      color = "green";
      label = "DONE";
      break;
    case "failed":
    case "error":
    case "disconnected":
      color = "red";
      label = status === "error" ? "ERROR" : status.toUpperCase();
      break;
    default:
      color = "gray";
      label = "UNKNOWN";
  }

  return (
    <Box>
      {showSpinner && (
        <Box marginRight={1}>
          <Spinner />
        </Box>
      )}
      <Text color={color} bold>
        {label}
      </Text>
      {message && (
        <Text color="gray"> {message}</Text>
      )}
    </Box>
  );
};






