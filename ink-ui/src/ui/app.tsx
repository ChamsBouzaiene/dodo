import fs from "node:fs";
import path from "node:path";
import React, { useEffect, useState, useCallback } from "react";
import { Box, useApp } from "ink";
import type { EngineClient } from "../engineClient.js";
import { useEngineConnection } from "../hooks/useEngineConnection.js";
import { AppLayout } from "../components/layout/AppLayout.js";
import { SetupWizard } from "../components/SetupWizard.js";
import { ProjectPermissionPrompt } from "../components/ProjectPermissionPrompt.js";
import { SessionProvider } from "../contexts/SessionContext.js";
import { AnimationProvider } from "../contexts/AnimationContext.js";
import { SessionPicker } from "../components/SessionPicker.js";

import { debugLog } from "../utils/debugLogger.js";
import { HelpModal } from "../components/modal/HelpModal.js";

import { useHistory } from "../hooks/useHistory.js";
import { useTerminal } from "../hooks/useTerminal.js";
import { useCommandProcessor } from "../hooks/useCommandProcessor.js";

// Debug: Log on module load
debugLog.lifecycle('App', 'mount', 'module loaded');

type AppProps = {
  client: EngineClient;
  repoPath: string;
  requestedSessionId?: string;
  engineCommand: string;
  engineExited?: { code: number | null; signal: NodeJS.Signals | null };
};

/**
 * Main Application Component.
 */
const App: React.FC<AppProps> = ({
  client,
  repoPath,
  requestedSessionId,
  engineCommand,
  engineExited,
}) => {
  const [input, setInput] = useState("");
  const [activeModal, setActiveModal] = useState<'none' | 'help'>('none');

  // Session selection state
  const [targetSessionId, setTargetSessionId] = useState(requestedSessionId);
  const [isSessionSelected, setIsSessionSelected] = useState(!!requestedSessionId);

  // Clear screen when transitioning from SessionPicker to the main app
  useEffect(() => {
    if (isSessionSelected && process.stdout?.write) {
      process.stdout.write('\x1b[2J\x1b[H');
    }
  }, [isSessionSelected]);

  // Use Custom Hooks
  const { history, historyIndex, addToHistory, navigateHistory, resetHistoryIndex } = useHistory();
  const { terminalRows, terminalColumns } = useTerminal({ enableAlternateBuffer: false });

  const {
    sessionId,
    status,
    infoMessage,
    isRunning,
    error,
    tokenUsage,
    projectPlan,
    showProjectPlan,
    errorCount,
    currentThought,
    turns,
    currentTimelineSteps,
    currentRunningStepId,
    toggleTurnCollapsed,
    submitQuery,
    sendCommand,
    isSetupRequired,
    setIsSetupRequired,
    isProjectPermissionRequired,
    setIsProjectPermissionRequired,
    pendingRepoRoot,
    reloadSession,
    loadedConfig,
    cancelRequest,
    clearTurns,
  } = useEngineConnection(client, repoPath, targetSessionId, engineExited, !isSessionSelected);

  const canSubmit = Boolean(sessionId) && !isRunning && status !== "connecting" && status !== "disconnected";
  const { exit } = useApp();

  const [isUpdateMode, setIsUpdateMode] = useState(false);

  useEffect(() => {
    if (!isSetupRequired) {
      setIsUpdateMode(false);
    }
  }, [isSetupRequired]);

  // Command Processor
  const { processCommand } = useCommandProcessor({
    sessionId,
    status,
    isRunning,
    loadedConfig,
    isSetupRequired,
    isUpdateMode,
    clearTurns,
    cancelRequest,
    sendCommand,
    setIsSetupRequired,
    setIsUpdateMode,
    setActiveModal,
    setInput,
  });

  const handleSubmit = useCallback(async () => {
    if (!canSubmit) return;
    const trimmed = input.trim();
    if (!trimmed) return;

    // Try handling as command first
    const handled = processCommand(trimmed);
    if (handled) return;

    // Add to history
    addToHistory(trimmed);

    // Submit query
    await submitQuery(trimmed);
    setInput("");
  }, [canSubmit, input, submitQuery, addToHistory, processCommand]);

  const navigate = (dir: 'up' | 'down') => {
    const res = navigateHistory(dir);
    if (res.index === -1) {
      setInput("");
    } else if (res.value !== undefined) {
      setInput(res.value);
    }
  };

  // If no session is selected, show the picker
  if (!isSessionSelected) {
    return (
      <AnimationProvider>
        <Box flexDirection="column" height={terminalRows} width="100%" justifyContent="center" alignItems="center">
          <SessionPicker
            repoPath={repoPath}
            onSelect={(id) => {
              setTargetSessionId(id);
              setIsSessionSelected(true);
            }}
          />
        </Box>
      </AnimationProvider>
    );
  }

  // If project permission is required, show the permission prompt
  if (isProjectPermissionRequired && sessionId) {
    return (
      <AnimationProvider>
        <Box flexDirection="column" height={terminalRows} width="100%" justifyContent="center" alignItems="center">
          <ProjectPermissionPrompt
            repoRoot={pendingRepoRoot || repoPath}
            onResponse={(enabled) => {
              process.stdout.write('\x1b[2J\x1b[H');

              sendCommand({
                type: 'project_permission',
                session_id: sessionId,
                indexing_enabled: enabled,
              });
              setIsProjectPermissionRequired(false);
            }}
          />
        </Box>
      </AnimationProvider>
    );
  }

  // If initial setup is required, show the wizard
  if (isSetupRequired) {
    return (
      <AnimationProvider>
        <Box flexDirection="column" height={terminalRows} width="100%" justifyContent="center" alignItems="center">
          <SetupWizard
            isUpdate={isUpdateMode}
            initialConfig={loadedConfig}
            sendCommand={(type, payload) => {
              if (type === 'save_config') {
                sendCommand({
                  type: 'save_config',
                  config: payload as Record<string, string>
                });
              }
            }}
            onComplete={() => {
              debugLog.lifecycle('App', 'update', 'SetupWizard onComplete');
              setIsSetupRequired(false);
              setIsUpdateMode(false);
              // Reload session to pick up new config
              debugLog.command('App', 'reloadSession', { sessionId });
              reloadSession();
            }}
          />
        </Box>
      </AnimationProvider>
    );
  }

  return (
    <AnimationProvider>
      <SessionProvider value={{
        sessionId,
        status,
        infoMessage,
        isRunning,
        error,
        tokenUsage,
        errorCount,
        currentThought,
        loadedConfig,
        isSetupRequired
      }}>
        <AppLayout
          terminalRows={terminalRows}
          terminalColumns={terminalColumns}
          error={error}
          showProjectPlan={showProjectPlan}
          projectPlan={projectPlan}
          currentTimelineSteps={currentTimelineSteps}
          currentRunningStepId={currentRunningStepId}
          isRunning={isRunning}
          footerProps={{
            input,
            onChange: (val) => {
              setInput(val);
              resetHistoryIndex();
            },
            onSubmit: handleSubmit,
            repoLabel: repoPath ? process.cwd() === repoPath ? path.basename(repoPath) : repoPath : "No Repo",

            // Pass history handlers to Footer
            onHistoryUp: () => navigate('up'),
            onHistoryDown: () => navigate('down'),
          }}
          // We need access to direct callbacks for navigation
          onCancelRequest={cancelRequest}
          helpModal={activeModal === 'help' ? <HelpModal onClose={() => setActiveModal('none')} /> : undefined}
          onHelp={() => setActiveModal('help')}
        />
      </SessionProvider>
    </AnimationProvider>
  );
};

export default App;
