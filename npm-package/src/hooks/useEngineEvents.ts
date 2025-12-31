import { useCallback, useEffect, useRef } from "react";
import type { Event, ActivityEvent as ProtocolActivityEvent } from "../protocol.js";
import type { EngineClient } from "../engineClient.js";
import type { Turn, DisplayToolEvent, UiStatus, Activity } from "../types.js";

const MAX_TOOL_EVENTS = 10;

type UseEngineEventsCallbacks = {
  onStatusChange?: (status: UiStatus, message: string) => void;
  onSessionReady?: (sessionId: string) => void;
  onAssistantText?: (content: string, replace: boolean, final: boolean) => void;
  onToolEvent?: (event: DisplayToolEvent) => void;
  onFilesChanged?: (files: string[]) => void;
  onDone?: (summary?: string, files?: string[]) => void;
  onError?: (message: string) => void;
  onActivity?: (activity: Activity) => void;
  onTokenUsage?: (usage: { prompt: number; limit: number; total: number }) => void;
  onProjectPlan?: (content: string, source: string) => void;
  onContext?: (event: { kind: string; description: string; before: number; after: number }) => void;
  onToolOutput?: (invocationId: string, tool: string, output: string, stream: "stdout" | "stderr") => void;
  onSetupRequired?: () => void;
  onSetupComplete?: () => void;
  onConfigLoaded?: (config: Record<string, string>) => void;
  onConfigReloaded?: (provider: string, modelName: string) => void;
  onCancelled?: (reason?: string) => void;
  onSessionHistory?: (title: string, summary: string, messages: Array<{ role: string, content: string }>) => void;
  onProjectPermissionRequired?: (repoRoot: string) => void;
};

export function useEngineEvents(
  client: EngineClient,
  callbacks: UseEngineEventsCallbacks
) {
  // ... (refs setup) ...
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
    };
  }, []);

  const callbacksRef = useRef(callbacks);

  useEffect(() => {
    callbacksRef.current = callbacks;
  });

  const handleEvent = useCallback(
    (event: Event) => {
      if (!mountedRef.current) return;
      const currentCallbacks = callbacksRef.current;

      switch (event.type) {
        case "status":
          if (event.status === "session_ready" && event.session_id) {
            if (currentCallbacks.onSessionReady) currentCallbacks.onSessionReady(event.session_id);
            // onSessionReady already updates status, no need to call onStatusChange again
          } else if (event.status === "engine_ready") {
            // Skip engine_ready status update
          } else if (event.status === "setup_complete") {
            if (currentCallbacks.onSetupComplete) currentCallbacks.onSetupComplete();
            // Also update status to ready?
            const status = mapStatusToUiStatus(event.status);
            if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange(status, event.detail ?? event.status);
          } else {
            const status = mapStatusToUiStatus(event.status);
            if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange(status, event.detail ?? event.status);
          }
          break;

        case "assistant_text":
          if (currentCallbacks.onAssistantText) {
            currentCallbacks.onAssistantText(
              event.content,
              false, // Never replace, always append for assistant text
              event.final ?? false
            );
          }
          break;

        case "tool_event":
          if (currentCallbacks.onToolEvent) {
            currentCallbacks.onToolEvent({
              tool: event.tool,
              phase: event.phase,
              success: event.success,
              details: event.details,
              timestamp: new Date(),
            });
          }
          break;

        case "files_changed":
          if (currentCallbacks.onFilesChanged) currentCallbacks.onFilesChanged(event.files);
          break;

        case "done":
          if (currentCallbacks.onDone) currentCallbacks.onDone(event.summary, event.files_changed);
          if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange("ready", event.summary ?? "Done");
          break;

        case "error":
          if (currentCallbacks.onError) currentCallbacks.onError(event.message);
          if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange("error", event.message);
          break;

        case "token_usage":
          if (currentCallbacks.onTokenUsage) {
            currentCallbacks.onTokenUsage({
              prompt: event.prompt_tokens,
              limit: event.limit,
              total: event.total,
            });
          }
          break;

        case "project_plan":
          if (currentCallbacks.onProjectPlan) currentCallbacks.onProjectPlan(event.content, event.source);
          break;

        case "context":
          if (currentCallbacks.onContext) {
            currentCallbacks.onContext({
              kind: event.kind,
              description: event.description,
              before: event.before,
              after: event.after,
            });
          }
          break;

        case "tool_output":
          if (event.stream === "stdout" || event.stream === "stderr") {
            if (currentCallbacks.onToolOutput) {
              currentCallbacks.onToolOutput(
                event.invocation_id,
                event.tool,
                event.output,
                event.stream
              );
            }
          }
          break;

        case "activity":
          // Convert protocol ActivityEvent to UI Activity type
          const activity: Activity = {
            id: event.activity_id,
            type: mapActivityType(event.activity_type),
            tool: event.tool,
            target: event.target,
            metadata: event.metadata,
            status: mapActivityStatus(event.status),
            timestamp: new Date(), // Fallback if start_time not parsed
            startTime: event.start_time ? new Date(event.start_time) : undefined,
            endTime: event.end_time ? new Date(event.end_time) : undefined,
            durationMs: event.duration_ms,
            invocationId: event.invocation_id,
            command: event.command,
            codeChange: event.code_change ? {
              file: event.code_change.file,
              before: event.code_change.before,
              after: event.code_change.after,
              startLine: event.code_change.start_line,
              endLine: event.code_change.end_line,
            } : undefined,
          };
          if (currentCallbacks.onActivity) currentCallbacks.onActivity(activity);
          break;

        case "setup_required":
          if (currentCallbacks.onSetupRequired) currentCallbacks.onSetupRequired();
          break;
        case "config_loaded":
          if (currentCallbacks.onConfigLoaded) currentCallbacks.onConfigLoaded(event.config);
          break;
        case "config_reloaded":
          if (currentCallbacks.onConfigReloaded) currentCallbacks.onConfigReloaded(event.provider, event.model_name);
          break;
        case "cancelled":
          if (currentCallbacks.onCancelled) currentCallbacks.onCancelled(event.reason);
          if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange("ready", event.reason || "Cancelled");
          break;
        case "session_history":
          if (currentCallbacks.onSessionHistory) {
            currentCallbacks.onSessionHistory(event.title, event.summary || "", event.messages || []);
          }
          break;
        case "project_permission_required":
          if (currentCallbacks.onProjectPermissionRequired) {
            currentCallbacks.onProjectPermissionRequired(event.repo_root);
          }
          break;
      }
    },
    []
  );

  const handleError = useCallback(
    (err: Error) => {
      if (!mountedRef.current) return;
      const currentCallbacks = callbacksRef.current;
      if (currentCallbacks.onError) currentCallbacks.onError(err.message);
      if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange("error", err.message);
    },
    []
  );

  const handleClose = useCallback(() => {
    if (!mountedRef.current) return;
    const currentCallbacks = callbacksRef.current;
    if (currentCallbacks.onStatusChange) currentCallbacks.onStatusChange("disconnected", "Engine connection closed");
  }, []);

  useEffect(() => {
    client.on("event", handleEvent);
    client.on("error", handleError);
    client.on("close", handleClose);

    return () => {
      client.off("event", handleEvent);
      client.off("error", handleError);
      client.off("close", handleClose);
    };
  }, [client, handleEvent, handleError, handleClose]);
}

function mapStatusToUiStatus(status: string): UiStatus {
  switch (status) {
    case "engine_ready":
      return "connecting";
    case "thinking":
      return "thinking";
    case "step_start":
      return "running_tools";
    case "done":
      return "ready";
    case "retry":
    case "budget_exceeded":
      return "running_tools";
    default:
      return "ready";
  }
}

function mapActivityType(type: string): Activity["type"] {
  switch (type) {
    case "thinking":
      return "thinking";
    case "reasoning":
      return "reasoning";
    case "edit":
      return "edit";
    case "tool":
      return "tool";
    default:
      return "tool";
  }
}

function mapActivityStatus(status: string): Activity["status"] {
  switch (status) {
    case "started":
      return "active";
    case "completed":
      return "completed";
    case "failed":
      return "failed";
    default:
      return "active";
  }
}

