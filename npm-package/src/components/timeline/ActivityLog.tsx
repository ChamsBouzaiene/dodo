import React, { useMemo } from "react";
import { Box, Text } from "ink";
import type { LogLine } from "../../types.js";

/**
 * Props for the ActivityLog component.
 */
type ActivityLogProps = {
  /** Array of log lines to display */
  lines: LogLine[];
  /** Maximum number of lines to show (default: 200) */
  maxLines?: number;
};

export const ActivityLog: React.FC<ActivityLogProps> = React.memo(({ lines, maxLines = 200 }) => {
  // Filter to only show error lines
  const errorLines = useMemo(() => {
    return lines.filter(line => line.level === "error").slice(-maxLines);
  }, [lines, maxLines]);

  const displayLines = errorLines;

  if (displayLines.length === 0) {
    return (
      <Box padding={1}>
        <Text color="gray" dimColor>
          Activity log empty...
        </Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" padding={1}>
      <Box marginBottom={1}>
        <Text color="red" bold>
          Errors ({displayLines.length})
        </Text>
      </Box>
      <Box flexDirection="column" height={10} overflow="hidden">
        {displayLines.map((line) => {
          let color: string = "white";
          let prefix = "";

          // Determine color and prefix based on source and level
          if (line.source === "command") {
            prefix = "$ ";
            color = "cyan";
          } else if (line.level === "error") {
            color = "red";
            prefix = "[ERROR] ";
          } else if (line.level === "success") {
            color = "green";
            prefix = "[OK] ";
          } else if (line.toolName) {
            prefix = `[${line.toolName.toUpperCase()}] `;
            color = "gray";
          }

          // Format timestamp
          const timeStr = line.timestamp.toLocaleTimeString();

          return (
            <Box key={line.id} flexDirection="row">
              <Text color="dim" dimColor>
                {timeStr}
              </Text>
              <Text color="dim" dimColor>
                {" "}
              </Text>
              <Box flexGrow={1}>
                <Text color={color}>
                  {prefix}
                  {line.text}
                </Text>
              </Box>
            </Box>
          );
        })}
      </Box>
    </Box>
  );
}, (prevProps, nextProps) => {
  // Only re-render if lines array reference changed
  return prevProps.lines === nextProps.lines && prevProps.maxLines === nextProps.maxLines;
});

