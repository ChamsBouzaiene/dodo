import React from "react";
import { Box, Text } from "ink";
import { Input } from "../common/Input.js";
import { StatusBadge } from "../common/StatusBadge.js";
import { Spinner } from "../common/Spinner.js";
import type { UiStatus, TokenUsage } from "../../types.js";
import { useSessionContext } from "../../contexts/SessionContext.js";

/**
 * Props for the Footer component.
 */
export type FooterProps = {
  /** The current value of the input field */
  input: string;
  /** Callback when input changes */
  onChange: (value: string) => void;
  /** Callback when input is submitted */
  onSubmit: () => void;
  /** Callback for history navigation (up arrow) */
  onHistoryUp?: () => void;
  /** Callback for history navigation (down arrow) */
  onHistoryDown?: () => void;
  /** Label for the current repository */
  repoLabel: string;
};

export const Footer: React.FC<FooterProps> = ({
  input,
  onChange,
  onSubmit,
  onHistoryUp,
  onHistoryDown,
  repoLabel,
}) => {
  const {
    sessionId: sessionLabel,
    status,
    infoMessage,
    isRunning,
    tokenUsage,
    errorCount,
    currentThought,
    loadedConfig,
  } = useSessionContext();

  const canSubmit = !isRunning && status !== "connecting" && status !== "disconnected";
  const modelLabel = loadedConfig?.model as string | undefined;
  return (
    <Box flexDirection="column" borderStyle="round" paddingX={1} flexGrow={1}>
      <Box>
        <Text color="gray">
          Tokens:{" "}
          {tokenUsage ? (
            <Text color={tokenUsage.percentage > 80 ? "red" : tokenUsage.percentage > 50 ? "yellow" : "green"}>
              Context: {tokenUsage.used} / {tokenUsage.total} ({tokenUsage.percentage.toFixed(2)}%)
              {tokenUsage.sessionTotal !== undefined && ` | Session: ${tokenUsage.sessionTotal}`}
            </Text>
          ) : (
            "—"
          )}
          {isRunning && <Text color="yellow"> │ ESC to stop</Text>}
        </Text>
      </Box>
      <Box>
        {currentThought ? (
          <Box flexDirection="column" marginBottom={0}>
            <Box>
              <Text color="gray" dimColor>
                <Spinner /> Thinking...
              </Text>
            </Box>
            <Box>
              <Text color="cyan">&gt; </Text>
              {canSubmit ? (
                <Input
                  value={input}
                  onChange={onChange}
                  onSubmit={onSubmit}
                  placeholder="Describe your task, or type /help..."
                  onHistoryUp={onHistoryUp}
                  onHistoryDown={onHistoryDown}
                />
              ) : (
                <Text color="gray">Processing... <Text color="yellow">(ESC to stop)</Text></Text>
              )}
            </Box>
          </Box>
        ) : (
          <Box>
            <Text color="cyan">&gt; </Text>
            {canSubmit ? (
              <Input
                value={input}
                onChange={onChange}
                onSubmit={onSubmit}
                placeholder="Describe your task, or type /help..."
                onHistoryUp={onHistoryUp}
                onHistoryDown={onHistoryDown}
              />
            ) : (
              <Box>
                <Text color="yellow">
                  <Spinner />
                </Text>
                <Text color="gray"> {isRunning ? "Processing... " : "Connecting to engine..."}</Text>
                {isRunning && <Text color="yellow">(ESC to stop)</Text>}
              </Box>
            )}
          </Box>
        )}
      </Box>

      {/* Metadata Section */}
      <Box borderStyle="single" borderTop borderBottom={false} borderLeft={false} borderRight={false} borderColor="gray" marginTop={0}>
        <Box flexGrow={1}>
          <Text>
            Repo: <Text color="blue">{repoLabel}</Text> │
            Session: <Text color="magenta">{sessionLabel}</Text>
            {modelLabel && <Text> │ Model: <Text color="green">{modelLabel}</Text></Text>}
          </Text>
        </Box>
        <Box>
          {errorCount !== undefined && errorCount > 0 && (
            <Text color="red">Errors: {errorCount} │ </Text>
          )}
          <StatusBadge status={status} message={infoMessage} />
        </Box>
      </Box>
    </Box>
  );
};
