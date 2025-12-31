
import React from "react";
import { Box, Text } from "ink";
import { FormattedText } from "../common/FormattedText.js";
import type { PlanStep } from "../../types.js";

/**
 * Props for the PlanContent component.
 */
type PlanContentProps = {
  /** Raw string content of the plan */
  content?: string;
  /** Summary of the plan */
  summary?: string;
  /** Array of plan steps */
  steps?: PlanStep[];
  /** Target areas for the plan */
  targetAreas?: string[];
  /** Risks associated with the plan */
  risks?: string[];
};

export const PlanContent: React.FC<PlanContentProps> = ({
  content,
  summary,
  steps,
  targetAreas,
  risks,
}) => {
  // If we have structured data, render it nicely
  if (summary || steps) {
    return (
      <Box flexDirection="column" marginTop={1} borderStyle="round" borderColor="cyan" paddingX={1}>
        {summary && (
          <Box marginBottom={1}>
            <Text color="white" bold>
              Plan Summary:
            </Text>
            <Box marginLeft={1} marginTop={0}>
              <Text color="white">{summary}</Text>
            </Box>
          </Box>
        )}

        {steps && steps.length > 0 && (
          <Box marginBottom={1} flexDirection="column">
            <Text color="white" bold>
              Steps ({steps.length}):
            </Text>
            {steps.map((step, index) => {
              const stepId = step.id || `step - ${index + 1} `;
              const description = step.description || "";
              const targetFiles = step.target_files || [];
              const status = step.status || "pending";

              return (
                <Box key={stepId} marginLeft={1} marginTop={0} flexDirection="column">
                  <Box flexDirection="row">
                    <Text color="cyan" bold>
                      {index + 1}.
                    </Text>
                    <Box marginLeft={1} flexGrow={1}>
                      <Text color="white">{description}</Text>
                    </Box>
                    {status === "completed" && (
                      <Text color="green"> ✓</Text>
                    )}
                    {status === "skipped" && (
                      <Text color="gray"> ⊘</Text>
                    )}
                  </Box>
                  {targetFiles.length > 0 && (
                    <Box marginLeft={2} marginTop={0}>
                      <Text color="gray" dimColor>
                        Files: {targetFiles.join(", ")}
                      </Text>
                    </Box>
                  )}
                </Box>
              );
            })}
          </Box>
        )}

        {targetAreas && targetAreas.length > 0 && (
          <Box marginBottom={1}>
            <Text color="white" bold>
              Target Areas:
            </Text>
            <Box marginLeft={1} marginTop={0}>
              <Text color="gray">{targetAreas.join(", ")}</Text>
            </Box>
          </Box>
        )}

        {risks && risks.length > 0 && (
          <Box marginBottom={1}>
            <Text color="red" bold>
              Risks:
            </Text>
            {risks.map((risk: string, index: number) => (
              <Box key={index} marginLeft={1} marginTop={0}>
                <Text color="yellow">⚠ {risk}</Text>
              </Box>
            ))}
          </Box>
        )}
      </Box>
    );
  }

  // Fallback: render raw content if available
  if (content) {
    // Parse markdown-like formatting from the plan tool output
    const lines = content.split("\n");
    return (
      <Box flexDirection="column" marginTop={1}>
        {lines.map((line, index) => {
          const trimmed = line.trim();
          if (!trimmed) {
            // Preserve empty lines for spacing
            return <Box key={index} height={1} />;
          }

          // Detect success indicator
          if (trimmed.startsWith("✅")) {
            return (
              <Text key={index} color="green" bold>
                {trimmed}
              </Text>
            );
          }

          // Detect markdown-style headers
          if (trimmed.startsWith("Plan Summary:") || trimmed.startsWith("Plan:")) {
            return (
              <Box key={index} marginTop={index > 0 ? 1 : 0}>
                <Text color="cyan" bold>
                  {trimmed}
                </Text>
              </Box>
            );
          }

          // Detect step numbers with brackets like " 1. [ ]" or " 1. [✓]"
          if (/^\s*\d+\.\s*\[/.test(trimmed)) {
            const hasCheckmark = trimmed.includes("✓") || trimmed.includes("✅");
            return (
              <Box key={index} marginLeft={1}>
                <Text color={hasCheckmark ? "green" : "white"}>
                  {trimmed}
                </Text>
              </Box>
            );
          }

          // Detect step numbers
          if (/^\d+\./.test(trimmed)) {
            return (
              <Box key={index} marginLeft={1}>
                <Text color="white">
                  {trimmed}
                </Text>
              </Box>
            );
          }

          // Detect instructions/hints
          if (trimmed.startsWith("You can now") || trimmed.startsWith("Use 'revise_plan'")) {
            return (
              <Box key={index} marginTop={1}>
                <Text color="gray" dimColor>
                  {trimmed}
                </Text>
              </Box>
            );
          }

          // Regular text
          return (
            <Text key={index} color="white">{trimmed}</Text>
          );
        })}
      </Box>
    );
  }

  return null;
};

