package protocol

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// CommandType enumerates all supported CLI -> engine commands.
type CommandType string

const (
	CommandStartSession      CommandType = "start_session"
	CommandUserMessage       CommandType = "user_message"
	CommandSaveConfig        CommandType = "save_config"
	CommandGetConfig         CommandType = "get_config"
	CommandReloadConfig      CommandType = "reload_config"
	CommandCancelRequest     CommandType = "cancel_request"
	CommandProjectPermission CommandType = "project_permission"
)

// Command is a marker interface implemented by all protocol commands.
type Command interface {
	GetType() CommandType
}

// StartSessionCommand initializes (or resumes) a session.
type StartSessionCommand struct {
	Type      CommandType       `json:"type"`
	SessionID string            `json:"session_id,omitempty"`
	RepoRoot  string            `json:"repo_root,omitempty"`
	Meta      map[string]any    `json:"meta,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
}

// GetType implements Command.
func (c StartSessionCommand) GetType() CommandType { return CommandStartSession }

// UserMessageCommand sends a user instruction to the engine.
type UserMessageCommand struct {
	Type      CommandType `json:"type"`
	SessionID string      `json:"session_id"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
}

// GetType implements Command.
func (c UserMessageCommand) GetType() CommandType { return CommandUserMessage }

// SaveConfigCommand persists user configuration.
type SaveConfigCommand struct {
	Type   CommandType       `json:"type"`
	Config map[string]string `json:"config"`
}

// GetType implements Command.
func (c SaveConfigCommand) GetType() CommandType { return CommandSaveConfig }

type rawCommand struct {
	Type CommandType `json:"type"`
}

// DecodeCommand converts raw JSON into a strongly typed command.
func DecodeCommand(data []byte) (Command, error) {
	var base rawCommand
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("decode command: %w", err)
	}

	switch base.Type {
	case CommandStartSession:
		var cmd StartSessionCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode start_session: %w", err)
		}
		if cmd.Meta == nil {
			cmd.Meta = map[string]any{}
		}
		return cmd, nil
	case CommandSaveConfig:
		var cmd SaveConfigCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode save_config: %w", err)
		}
		return cmd, nil
	case CommandGetConfig:
		var cmd GetConfigCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode get_config: %w", err)
		}
		return cmd, nil
	case CommandReloadConfig:
		var cmd ReloadConfigCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode reload_config: %w", err)
		}
		if cmd.SessionID == "" {
			return nil, errors.New("reload_config requires session_id")
		}
		return cmd, nil
	case CommandUserMessage:
		var cmd UserMessageCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode user_message: %w", err)
		}
		if cmd.SessionID == "" {
			return nil, errors.New("user_message requires session_id")
		}
		if cmd.Message == "" {
			return nil, errors.New("user_message requires message")
		}
		return cmd, nil
	case CommandCancelRequest:
		var cmd CancelRequestCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode cancel_request: %w", err)
		}
		if cmd.SessionID == "" {
			return nil, errors.New("cancel_request requires session_id")
		}
		return cmd, nil
	case CommandProjectPermission:
		var cmd ProjectPermissionCommand
		if err := json.Unmarshal(data, &cmd); err != nil {
			return nil, fmt.Errorf("decode project_permission: %w", err)
		}
		if cmd.SessionID == "" {
			return nil, errors.New("project_permission requires session_id")
		}
		return cmd, nil
	default:
		return nil, fmt.Errorf("unknown command type: %s", base.Type)
	}
}

// NewSessionID generates a new opaque session identifier.
func NewSessionID() string {
	return uuid.NewString()
}

// EventType enumerates engine -> CLI events.
type EventType string

const (
	EventAssistantText             EventType = "assistant_text"
	EventStatus                    EventType = "status"
	EventTool                      EventType = "tool_event"
	EventFilesChanged              EventType = "files_changed"
	EventDone                      EventType = "done"
	EventError                     EventType = "error"
	EventActivity                  EventType = "activity"
	EventTokenUsage                EventType = "token_usage"
	EventProjectPlan               EventType = "project_plan"
	EventToolOutput                EventType = "tool_output"
	EventContext                   EventType = "context"
	EventSetupRequired             EventType = "setup_required"
	EventConfigLoaded              EventType = "config_loaded"
	EventConfigReloaded            EventType = "config_reloaded"
	EventCancelled                 EventType = "cancelled"
	EventSessionHistory            EventType = "session_history"
	EventProjectPermissionRequired EventType = "project_permission_required"
)

// Event is implemented by every outgoing message.
type Event interface {
	isEvent()
	GetType() EventType
}

// MarshalEvent serializes an event into JSON for NDJSON transport.
func MarshalEvent(e Event) ([]byte, error) {
	return json.Marshal(e)
}

type eventBase struct {
	Type      EventType `json:"type"`
	SessionID string    `json:"session_id,omitempty"`
}

func (eventBase) isEvent() {}

// AssistantTextEvent streams assistant text/content back to the CLI.
type AssistantTextEvent struct {
	eventBase
	Content string `json:"content"`
	Final   bool   `json:"final,omitempty"`
	Source  string `json:"source,omitempty"`
}

// NewAssistantTextEvent constructs an assistant_text event.
func NewAssistantTextEvent(sessionID, content, source string, final bool) AssistantTextEvent {
	return AssistantTextEvent{
		eventBase: eventBase{Type: EventAssistantText, SessionID: sessionID},
		Content:   content,
		Source:    source,
		Final:     final,
	}
}

// GetType implements Event.
func (e AssistantTextEvent) GetType() EventType { return e.Type }

// StatusEvent communicates coarse engine state.
type StatusEvent struct {
	eventBase
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// NewStatusEvent constructs a status event.
func NewStatusEvent(sessionID, status, detail string) StatusEvent {
	return StatusEvent{
		eventBase: eventBase{Type: EventStatus, SessionID: sessionID},
		Status:    status,
		Detail:    detail,
	}
}

// GetType implements Event.
func (e StatusEvent) GetType() EventType { return e.Type }

// ToolEvent tracks tool invocation lifecycle.
type ToolEvent struct {
	eventBase
	Tool    string `json:"tool"`
	Phase   string `json:"phase"`
	Success *bool  `json:"success,omitempty"`
	Details string `json:"details,omitempty"`
}

// NewToolEvent constructs a tool_event message.
func NewToolEvent(sessionID, tool, phase string, success *bool, details string) ToolEvent {
	return ToolEvent{
		eventBase: eventBase{Type: EventTool, SessionID: sessionID},
		Tool:      tool,
		Phase:     phase,
		Success:   success,
		Details:   details,
	}
}

// GetType implements Event.
func (e ToolEvent) GetType() EventType { return e.Type }

// FilesChangedEvent communicates file modifications.
type FilesChangedEvent struct {
	eventBase
	Files []string `json:"files"`
}

// NewFilesChangedEvent constructs a files_changed event.
func NewFilesChangedEvent(sessionID string, files []string) FilesChangedEvent {
	return FilesChangedEvent{
		eventBase: eventBase{Type: EventFilesChanged, SessionID: sessionID},
		Files:     files,
	}
}

// GetType implements Event.
func (e FilesChangedEvent) GetType() EventType { return e.Type }

// DoneEvent signals session completion for a request.
type DoneEvent struct {
	eventBase
	Summary      string   `json:"summary,omitempty"`
	FilesChanged []string `json:"files_changed,omitempty"`
}

// NewDoneEvent constructs a done event.
func NewDoneEvent(sessionID, summary string, files []string) DoneEvent {
	return DoneEvent{
		eventBase:    eventBase{Type: EventDone, SessionID: sessionID},
		Summary:      summary,
		FilesChanged: files,
	}
}

// GetType implements Event.
func (e DoneEvent) GetType() EventType { return e.Type }

// ErrorEvent reports recoverable protocol or engine issues.
type ErrorEvent struct {
	eventBase
	Message string `json:"message"`
	Kind    string `json:"kind,omitempty"`
	Details string `json:"details,omitempty"`
}

// NewErrorEvent constructs an error event.
func NewErrorEvent(sessionID, message, kind, details string) ErrorEvent {
	return ErrorEvent{
		eventBase: eventBase{Type: EventError, SessionID: sessionID},
		Message:   message,
		Kind:      kind,
		Details:   details,
	}
}

// GetType implements Event.
func (e ErrorEvent) GetType() EventType { return e.Type }

// CodeChange represents a code modification with before/after content.
type CodeChange struct {
	File      string `json:"file"`
	Before    string `json:"before"`
	After     string `json:"after"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

// ActivityEvent tracks agent activities in real-time (tool calls, thinking, editing).
type ActivityEvent struct {
	eventBase
	ActivityID   string         `json:"activity_id"`
	ActivityType string         `json:"activity_type"` // "thinking", "tool", "edit", "reasoning"
	Tool         string         `json:"tool,omitempty"`
	Target       string         `json:"target,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Status       string         `json:"status"` // "started", "completed", "failed"
	CodeChange   *CodeChange    `json:"code_change,omitempty"`
	InvocationID string         `json:"invocation_id,omitempty"`
	StartTime    string         `json:"start_time,omitempty"`
	EndTime      string         `json:"end_time,omitempty"`
	DurationMs   int64          `json:"duration_ms,omitempty"`
	Command      string         `json:"command,omitempty"`
}

// NewActivityEvent constructs an activity event.
func NewActivityEvent(sessionID, activityID, activityType, tool, target, status string, metadata map[string]any, codeChange *CodeChange) ActivityEvent {
	return ActivityEvent{
		eventBase:    eventBase{Type: EventActivity, SessionID: sessionID},
		ActivityID:   activityID,
		ActivityType: activityType,
		Tool:         tool,
		Target:       target,
		Metadata:     metadata,
		Status:       status,
		CodeChange:   codeChange,
	}
}

// NewActivityEventWithTiming constructs an activity event with timing and command info.
func NewActivityEventWithTiming(sessionID, activityID, invocationID, activityType, tool, target, status, startTime, endTime string, durationMs int64, command string, metadata map[string]any, codeChange *CodeChange) ActivityEvent {
	return ActivityEvent{
		eventBase:    eventBase{Type: EventActivity, SessionID: sessionID},
		ActivityID:   activityID,
		InvocationID: invocationID,
		ActivityType: activityType,
		Tool:         tool,
		Target:       target,
		Status:       status,
		StartTime:    startTime,
		EndTime:      endTime,
		DurationMs:   durationMs,
		Command:      command,
		Metadata:     metadata,
		CodeChange:   codeChange,
	}
}

// GetType implements Event.
func (e ActivityEvent) GetType() EventType { return e.Type }

// TokenUsageEvent reports token consumption.
type TokenUsageEvent struct {
	eventBase
	PromptTokens int `json:"prompt_tokens"`
	Limit        int `json:"limit"`
	Total        int `json:"total"`
}

// NewTokenUsageEvent constructs a token_usage event.
func NewTokenUsageEvent(sessionID string, promptTokens, limit, total int) TokenUsageEvent {
	return TokenUsageEvent{
		eventBase:    eventBase{Type: EventTokenUsage, SessionID: sessionID},
		PromptTokens: promptTokens,
		Limit:        limit,
		Total:        total,
	}
}

// GetType implements Event.
func (e TokenUsageEvent) GetType() EventType { return e.Type }

// ProjectPlanEvent updates the project plan.
type ProjectPlanEvent struct {
	eventBase
	Content string `json:"content"`
	Source  string `json:"source"`
}

// NewProjectPlanEvent constructs a project_plan event.
func NewProjectPlanEvent(sessionID, content, source string) ProjectPlanEvent {
	return ProjectPlanEvent{
		eventBase: eventBase{Type: EventProjectPlan, SessionID: sessionID},
		Content:   content,
		Source:    source,
	}
}

// GetType implements Event.
func (e ProjectPlanEvent) GetType() EventType { return e.Type }

// ToolOutputEvent streams tool output (stdout/stderr).
type ToolOutputEvent struct {
	eventBase
	InvocationID string `json:"invocation_id"`
	Tool         string `json:"tool"`
	Output       string `json:"output"`
	IsError      bool   `json:"is_error"`
	Stream       string `json:"stream"` // stdout, stderr, command, complete
}

// NewToolOutputEvent constructs a tool_output event.
func NewToolOutputEvent(sessionID, invocationID, tool, output string, isError bool, stream string) ToolOutputEvent {
	return ToolOutputEvent{
		eventBase:    eventBase{Type: EventToolOutput, SessionID: sessionID},
		InvocationID: invocationID,
		Tool:         tool,
		Output:       output,
		IsError:      isError,
		Stream:       stream,
	}
}

// GetType implements Event.
func (e ToolOutputEvent) GetType() EventType { return e.Type }

// ContextEvent reports context management actions (compression, summarization).
type ContextEvent struct {
	eventBase
	Kind        string `json:"kind"` // compress, summarize
	Description string `json:"description"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
}

// NewContextEvent constructs a context event.
func NewContextEvent(sessionID, kind, description string, before, after int) ContextEvent {
	return ContextEvent{
		eventBase:   eventBase{Type: EventContext, SessionID: sessionID},
		Kind:        kind,
		Description: description,
		Before:      before,
		After:       after,
	}
}

// GetType implements Event.
func (e ContextEvent) GetType() EventType { return e.Type }

// SetupRequiredEvent signals that the user needs to configure the application.
type SetupRequiredEvent struct {
	eventBase
}

// NewSetupRequiredEvent constructs a setup_required event.
func NewSetupRequiredEvent() SetupRequiredEvent {
	return SetupRequiredEvent{
		eventBase: eventBase{Type: EventSetupRequired},
	}
}

// GetType implements Event.
func (e SetupRequiredEvent) GetType() EventType { return e.Type }

// GetConfigCommand requests the current configuration.
type GetConfigCommand struct {
	Type CommandType `json:"type"`
}

// GetType implements Command.
func (c GetConfigCommand) GetType() CommandType { return CommandGetConfig }

// ReloadConfigCommand triggers a hot-reload of the configuration for an existing session.
type ReloadConfigCommand struct {
	Type      CommandType `json:"type"`
	SessionID string      `json:"session_id"`
}

// GetType implements Command.
func (c ReloadConfigCommand) GetType() CommandType { return CommandReloadConfig }

// ConfigLoadedEvent returns the current configuration.
type ConfigLoadedEvent struct {
	eventBase
	Config map[string]string `json:"config"`
}

// NewConfigLoadedEvent constructs a config_loaded event.
func NewConfigLoadedEvent(config map[string]string) ConfigLoadedEvent {
	return ConfigLoadedEvent{
		eventBase: eventBase{Type: EventConfigLoaded},
		Config:    config,
	}
}

// GetType implements Event.
func (e ConfigLoadedEvent) GetType() EventType { return e.Type }

// ConfigReloadedEvent signals that the configuration was successfully reloaded.
type ConfigReloadedEvent struct {
	eventBase
	Provider  string `json:"provider"`
	ModelName string `json:"model_name"`
}

// NewConfigReloadedEvent constructs a config_reloaded event.
func NewConfigReloadedEvent(sessionID, provider, modelName string) ConfigReloadedEvent {
	return ConfigReloadedEvent{
		eventBase: eventBase{Type: EventConfigReloaded, SessionID: sessionID},
		Provider:  provider,
		ModelName: modelName,
	}
}

// GetType implements Event.
func (e ConfigReloadedEvent) GetType() EventType { return e.Type }

// CancelRequestCommand requests cancellation of the current running task.
type CancelRequestCommand struct {
	Type      CommandType `json:"type"`
	SessionID string      `json:"session_id"`
}

// GetType implements Command.
func (c CancelRequestCommand) GetType() CommandType { return CommandCancelRequest }

// CancelledEvent signals that a task was cancelled by user request.
type CancelledEvent struct {
	eventBase
	Reason string `json:"reason,omitempty"`
}

// NewCancelledEvent constructs a cancelled event.
func NewCancelledEvent(sessionID, reason string) CancelledEvent {
	return CancelledEvent{
		eventBase: eventBase{Type: EventCancelled, SessionID: sessionID},
		Reason:    reason,
	}
}

// GetType implements Event.
func (e CancelledEvent) GetType() EventType { return e.Type }

// HistoryMessage represents a single message in conversation history for UI display.
type HistoryMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// SessionHistoryEvent sends the previous conversation history when resuming a session.
type SessionHistoryEvent struct {
	eventBase
	Title    string           `json:"title"`
	Summary  string           `json:"summary,omitempty"`
	Messages []HistoryMessage `json:"messages"`
}

// NewSessionHistoryEvent constructs a session_history event.
func NewSessionHistoryEvent(sessionID, title, summary string, messages []HistoryMessage) SessionHistoryEvent {
	return SessionHistoryEvent{
		eventBase: eventBase{Type: EventSessionHistory, SessionID: sessionID},
		Title:     title,
		Summary:   summary,
		Messages:  messages,
	}
}

// GetType implements Event.
func (e SessionHistoryEvent) GetType() EventType { return e.Type }

// ProjectPermissionCommand sends user's decision about project indexing.
type ProjectPermissionCommand struct {
	Type            CommandType `json:"type"`
	SessionID       string      `json:"session_id"`
	IndexingEnabled bool        `json:"indexing_enabled"`
}

// GetType implements Command.
func (c ProjectPermissionCommand) GetType() CommandType { return CommandProjectPermission }

// ProjectPermissionRequiredEvent signals that the project needs indexing permission.
type ProjectPermissionRequiredEvent struct {
	eventBase
	RepoRoot string `json:"repo_root"`
}

// NewProjectPermissionRequiredEvent constructs a project_permission_required event.
func NewProjectPermissionRequiredEvent(sessionID, repoRoot string) ProjectPermissionRequiredEvent {
	return ProjectPermissionRequiredEvent{
		eventBase: eventBase{Type: EventProjectPermissionRequired, SessionID: sessionID},
		RepoRoot:  repoRoot,
	}
}

// GetType implements Event.
func (e ProjectPermissionRequiredEvent) GetType() EventType { return e.Type }
