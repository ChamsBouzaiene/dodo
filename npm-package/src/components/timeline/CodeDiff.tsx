import React from "react";
import { Box, Text } from "ink";
import type { CodeChange } from "../../types.js";

/**
 * Props for the CodeDiff component.
 */
type CodeDiffProps = {
  /** The code change details (diff) */
  codeChange: CodeChange;
};

// Split text into lines and truncate if too long
const formatDiffLines = (text: string, prefix: string, color: string, maxLines: number = 10): JSX.Element[] => {
  const lines = text.split("\n").slice(0, maxLines);
  const hasMore = text.split("\n").length > maxLines;

  const elements = lines.map((line, idx) => (
    <Box key={idx}>
      <Text color={color}>{prefix} {line || "(empty line)"}</Text>
    </Box>
  ));

  if (hasMore) {
    elements.push(
      <Box key="more">
        <Text color="gray">  ...</Text>
      </Box>
    );
  }

  return elements;
};

export const CodeDiff: React.FC<CodeDiffProps> = ({ codeChange }) => {
  const { file, before, after, startLine, endLine } = codeChange;

  // Format location
  let location = file;
  if (startLine && endLine) {
    location += ` (L${startLine}-${endLine})`;
  } else if (startLine) {
    location += ` (L${startLine})`;
  }

  return (
    <Box flexDirection="column" marginLeft={2} marginTop={1} marginBottom={1}>
      <Box>
        <Text color="magenta">📝 {location}</Text>
      </Box>

      {before && (
        <Box flexDirection="column" marginLeft={1}>
          {formatDiffLines(before, "-", "red", 5)}
        </Box>
      )}

      {after && (
        <Box flexDirection="column" marginLeft={1}>
          {formatDiffLines(after, "+", "green", 5)}
        </Box>
      )}
    </Box>
  );
};


