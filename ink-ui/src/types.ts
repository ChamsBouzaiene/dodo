export type CodeChange = {
  file: string;
  before: string;
  after: string;
  startLine?: number;
  endLine?: number;
};

export type Activity = {
  id: string;
  type: "thinking" | "tool" | "edit" | "reasoning";
  tool?: string;
  target?: string;
  metadata?: Record<string, any>;
  status: "active" | "completed" | "failed";
  timestamp: Date;
  startTime?: Date;
  endTime?: Date;
  durationMs?: number;
  invocationId?: string;
  command?: string;
  codeChange?: CodeChange;
};

export type TokenUsage = {
  used: number;
  total: number;
  percentage: number;
  sessionTotal?: number;
};

export type TimelineStep = {
  id: string; // invocation_id
  toolName: string;
  label: string;
  status: "pending" | "running" | "done" | "failed";
  startedAt: Date;
  finishedAt?: Date;
  durationMs?: number;
  metadata?: Record<string, any>;
  command?: string; // For execution tools
  type?: "tool" | "context_event"; // Distinguish between regular tools and context events
  contextEventType?: "compress" | "summarize"; // Specific context event type
};

export type LogLine = {
  id: string;
  timestamp: Date;
  source: "tool" | "command" | "system";
  toolName?: string;
  invocationId?: string;
  text: string;
  level?: "info" | "error" | "success";
};

export type Turn = {
  id: string;
  user: string;
  assistant: string;
  summary?: string;
  done: boolean;
  collapsed: boolean;
  activities: Activity[];
  timelineSteps: TimelineStep[];
  logLines: LogLine[];
};

export type DisplayToolEvent = {
  tool: string;
  phase: string;
  success?: boolean;
  details?: string;
  timestamp: Date;
};

export type PlanStep = {
  id?: string;
  description?: string;
  target_files?: string[];
  status?: "pending" | "completed" | "skipped";
};

export type UiStatus =
  | "booting"
  | "connecting"
  | "ready"
  | "thinking"
  | "running_tools"
  | "done"
  | "error"
  | "disconnected";

