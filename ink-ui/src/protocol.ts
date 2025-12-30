export type StartSessionCommand = {
  type: "start_session";
  session_id?: string;
  repo_root?: string;
  meta?: Record<string, unknown>;
  config?: Record<string, string>;
};

export type UserMessageCommand = {
  type: "user_message";
  session_id: string;
  message: string;
  request_id?: string;
};

export type SaveConfigCommand = {
  type: "save_config";
  config: Record<string, string>;
};

export type GetConfigCommand = {
  type: "get_config";
};

export type ReloadConfigCommand = {
  type: "reload_config";
  session_id: string;
};

export type CancelRequestCommand = {
  type: "cancel_request";
  session_id: string;
};

export type ProjectPermissionCommand = {
  type: "project_permission";
  session_id: string;
  indexing_enabled: boolean;
};

export type Command = StartSessionCommand | UserMessageCommand | SaveConfigCommand | GetConfigCommand | ReloadConfigCommand | CancelRequestCommand | ProjectPermissionCommand;

export type StatusEvent = {
  type: "status";
  session_id?: string;
  status: string;
  detail?: string;
};

export type AssistantTextEvent = {
  type: "assistant_text";
  session_id: string;
  content: string;
  final?: boolean;
  source?: "delta" | "assistant" | "respond.summary" | string;
};

export type ToolEvent = {
  type: "tool_event";
  session_id: string;
  tool: string;
  phase: string;
  success?: boolean;
  details?: string;
};

export type FilesChangedEvent = {
  type: "files_changed";
  session_id: string;
  files: string[];
};

export type DoneEvent = {
  type: "done";
  session_id: string;
  summary?: string;
  files_changed?: string[];
};

export type ErrorEvent = {
  type: "error";
  session_id?: string;
  message: string;
  kind?: string;
  details?: string;
};

export type CodeChangeData = {
  file: string;
  before: string;
  after: string;
  start_line?: number;
  end_line?: number;
};

export type ActivityEvent = {
  type: "activity";
  session_id: string;
  activity_id: string;
  activity_type: string;
  tool?: string;
  target?: string;
  metadata?: Record<string, any>;
  status: "started" | "completed" | "failed";
  code_change?: CodeChangeData;
  invocation_id?: string;
  start_time?: string;
  end_time?: string;
  duration_ms?: number;
  command?: string;
};

export type TokenUsageEvent = {
  type: "token_usage";
  session_id: string;
  prompt_tokens: number;
  limit: number;
  total: number;
};

export type ProjectPlanEvent = {
  type: "project_plan";
  session_id: string;
  content: string;
  source: string;
};

export type ToolOutputEvent = {
  type: "tool_output";
  session_id: string;
  invocation_id: string;
  tool: string;
  output: string;
  is_error: boolean;
  stream: "stdout" | "stderr" | "command" | "complete";
};

export type ContextEvent = {
  type: "context";
  session_id: string;
  kind: string;
  description: string;
  before: number;
  after: number;
};

export type SetupRequiredEvent = {
  type: "setup_required";
  session_id?: string;
};

export type ConfigLoadedEvent = {
  type: "config_loaded";
  session_id?: string;
  config: Record<string, string>;
};

export type ConfigReloadedEvent = {
  type: "config_reloaded";
  session_id: string;
  provider: string;
  model_name: string;
};

export type CancelledEvent = {
  type: "cancelled";
  session_id: string;
  reason?: string;
};

export type HistoryMessage = {
  role: string;
  content: string;
};

export type SessionHistoryEvent = {
  type: "session_history";
  session_id: string;
  title: string;
  summary?: string;
  messages: HistoryMessage[];
};

export type ProjectPermissionRequiredEvent = {
  type: "project_permission_required";
  session_id?: string;
  repo_root: string;
};

export type Event =
  | StatusEvent
  | AssistantTextEvent
  | ToolEvent
  | FilesChangedEvent
  | DoneEvent
  | ErrorEvent
  | ActivityEvent
  | TokenUsageEvent
  | ProjectPlanEvent
  | ToolOutputEvent
  | ContextEvent
  | SetupRequiredEvent
  | ConfigLoadedEvent
  | ConfigReloadedEvent
  | CancelledEvent
  | SessionHistoryEvent
  | ProjectPermissionRequiredEvent;

export const serializeCommand = (command: Command): string => {
  return JSON.stringify(command);
};
