package providers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ChamsBouzaiene/dodo/internal/engine"

	anthropic "github.com/liushuangls/go-anthropic/v2"
)

// AnthropicClient implements engine.LLMClient by calling Anthropic SDK directly.
// This avoids any dependency on Eino.
type AnthropicClient struct {
	client *anthropic.Client
	model  string
}

// NewAnthropicClient creates a new Anthropic client for the engine.
func NewAnthropicClient(apiKey, modelName string) (*AnthropicClient, error) {
	client := anthropic.NewClient(apiKey)

	return &AnthropicClient{
		client: client,
		model:  modelName,
	}, nil
}

// Chat implements engine.LLMClient.Chat by calling Anthropic API directly.
func (c *AnthropicClient) Chat(ctx context.Context, modelName string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (engine.LLMResponse, error) {
	// Convert engine messages to Anthropic format
	var systemParts []anthropic.MessageSystemPart
	var anthropicMsgs []anthropic.Message

	// Track if previous assistant message had tool calls (for proper message ordering)
	var prevAssistantHadToolCalls bool

	for i, msg := range messages {
		switch msg.Role {
		case engine.RoleSystem:
			systemParts = append(systemParts, anthropic.MessageSystemPart{
				Type: "text",
				Text: msg.Content,
			})
			prevAssistantHadToolCalls = false // Reset after system message
		case engine.RoleUser:
			anthropicMsgs = append(anthropicMsgs, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{anthropic.NewTextMessageContent(msg.Content)},
			})
			prevAssistantHadToolCalls = false // Reset after user message
		case engine.RoleAssistant:
			// Assistant messages can have text and/or tool calls
			var content []anthropic.MessageContent
			if msg.Content != "" && msg.Content != " " {
				content = append(content, anthropic.NewTextMessageContent(msg.Content))
			}

			// Add tool_use blocks if this assistant message had tool calls
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Args)
					toolUse := anthropic.NewToolUseMessageContent(
						tc.ID,
						tc.Name,
						json.RawMessage(argsJSON),
					)
					content = append(content, toolUse)
				}
			}

			anthropicMsgs = append(anthropicMsgs, anthropic.Message{
				Role:    anthropic.RoleAssistant,
				Content: content,
			})
			// Track if this assistant message had tool calls
			prevAssistantHadToolCalls = len(msg.ToolCalls) > 0
		case engine.RoleTool:
			// Anthropic requires tool results to follow an assistant message with tool_use
			// Skip tool messages if previous assistant didn't have tool calls
			if !prevAssistantHadToolCalls {
				// This shouldn't happen in normal flow, but skip to avoid API error
				continue
			}
			// Tool results
			// msg.Name contains the tool_use_id from when we stored the tool result
			// Ensure content is never empty (Anthropic may reject empty content)
			content := msg.Content
			if content == "" {
				content = "{}" // Empty JSON object instead of empty string
			}
			toolResult := anthropic.NewToolResultMessageContent(
				msg.Name, // This is the tool_use_id, not the tool name
				content,
				false, // isError
			)
			anthropicMsgs = append(anthropicMsgs, anthropic.Message{
				Role:    anthropic.RoleUser,
				Content: []anthropic.MessageContent{toolResult},
			})
			// Check if next message is assistant (to reset tracking)
			if i+1 < len(messages) && messages[i+1].Role == engine.RoleAssistant {
				prevAssistantHadToolCalls = false
			}
		}
	}

	// Convert tool schemas to Anthropic format
	var toolDefs []anthropic.ToolDefinition
	for _, ts := range toolSchemas {
		var schemaObj map[string]any
		if err := json.Unmarshal([]byte(ts.JSONSchema), &schemaObj); err != nil {
			return engine.LLMResponse{}, fmt.Errorf("invalid tool schema JSON for %s: %w", ts.Name, err)
		}

		toolDefs = append(toolDefs, anthropic.ToolDefinition{
			Name:        ts.Name,
			Description: ts.Description,
			InputSchema: schemaObj,
		})
	}

	// Build request
	maxTokens := 4096
	if opts.MaxOutputTokens > 0 {
		maxTokens = opts.MaxOutputTokens
	}

	temperature := float32(0.1)
	if opts.Temperature > 0 {
		temperature = opts.Temperature
	}

	req := anthropic.MessagesRequest{
		Model:       anthropic.Model(modelName),
		Messages:    anthropicMsgs,
		MaxTokens:   maxTokens,
		Temperature: &temperature,
	}

	// Add system messages if present
	if len(systemParts) > 0 {
		req.MultiSystem = systemParts
	}

	// Add tools if present
	if len(toolDefs) > 0 {
		req.Tools = toolDefs
	}

	// Call Anthropic API
	resp, err := c.client.CreateMessages(ctx, req)
	if err != nil {
		// Extract HTTP status and Retry-After from error
		httpStatus, retryAfter := extractErrorMetadata(err)
		wrappedErr := engine.WrapLLMError(err, httpStatus, retryAfter)
		return engine.LLMResponse{}, wrappedErr
	}

	// Extract text content and tool calls
	var textContent string
	var toolCalls []engine.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case anthropic.MessagesContentTypeText:
			if block.Text != nil {
				textContent += *block.Text
			}
		case "tool_use":
			if block.MessageContentToolUse != nil && block.ID != "" && block.Name != "" {
				var args map[string]any
				if len(block.Input) > 0 {
					if err := json.Unmarshal(block.Input, &args); err != nil {
						args = make(map[string]any)
					}
				} else {
					args = make(map[string]any)
				}

				toolCalls = append(toolCalls, engine.ToolCall{
					ID:   block.ID, // Store Anthropic's tool use ID
					Name: block.Name,
					Args: args,
				})
			}
		}
	}

	// Determine finish reason
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	} else if resp.StopReason == "max_tokens" {
		finishReason = "length"
	} else if resp.StopReason == "content_filtered" {
		finishReason = "content_filter"
	}

	// Extract usage
	usage := engine.Usage{
		Prompt:     resp.Usage.InputTokens,
		Completion: resp.Usage.OutputTokens,
		Total:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}

	assistantMsg := engine.ChatMessage{
		Role:      engine.RoleAssistant,
		Content:   textContent,
		ToolCalls: toolCalls, // Store tool calls for proper message reconstruction
	}

	return engine.LLMResponse{
		Assistant:    assistantMsg,
		ToolCalls:    toolCalls,
		Usage:        usage,
		FinishReason: finishReason,
	}, nil
}

// Stream implements engine.LLMClient.Stream with real streaming support.
// Note: Anthropic SDK uses callback-based streaming, which we adapt to channels.
func (c *AnthropicClient) Stream(ctx context.Context, modelName string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (<-chan engine.StreamEvent, <-chan error) {
	eventCh := make(chan engine.StreamEvent, 10) // Buffered to avoid blocking
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		// Convert messages and tools (reuse logic from Chat())
		var systemParts []anthropic.MessageSystemPart
		var anthropicMsgs []anthropic.Message
		var prevAssistantHadToolCalls bool

		for i, msg := range messages {
			switch msg.Role {
			case engine.RoleSystem:
				systemParts = append(systemParts, anthropic.MessageSystemPart{
					Type: "text",
					Text: msg.Content,
				})
				prevAssistantHadToolCalls = false
			case engine.RoleUser:
				anthropicMsgs = append(anthropicMsgs, anthropic.Message{
					Role:    anthropic.RoleUser,
					Content: []anthropic.MessageContent{anthropic.NewTextMessageContent(msg.Content)},
				})
				prevAssistantHadToolCalls = false
			case engine.RoleAssistant:
				var content []anthropic.MessageContent
				if msg.Content != "" && msg.Content != " " {
					content = append(content, anthropic.NewTextMessageContent(msg.Content))
				}
				if len(msg.ToolCalls) > 0 {
					for _, tc := range msg.ToolCalls {
						argsJSON, _ := json.Marshal(tc.Args)
						toolUse := anthropic.NewToolUseMessageContent(
							tc.ID,
							tc.Name,
							json.RawMessage(argsJSON),
						)
						content = append(content, toolUse)
					}
				}
				anthropicMsgs = append(anthropicMsgs, anthropic.Message{
					Role:    anthropic.RoleAssistant,
					Content: content,
				})
				prevAssistantHadToolCalls = len(msg.ToolCalls) > 0
			case engine.RoleTool:
				if !prevAssistantHadToolCalls {
					continue
				}
				content := msg.Content
				if content == "" {
					content = "{}"
				}
				toolResult := anthropic.NewToolResultMessageContent(
					msg.Name,
					content,
					false,
				)
				anthropicMsgs = append(anthropicMsgs, anthropic.Message{
					Role:    anthropic.RoleUser,
					Content: []anthropic.MessageContent{toolResult},
				})
				if i+1 < len(messages) && messages[i+1].Role == engine.RoleAssistant {
					prevAssistantHadToolCalls = false
				}
			}
		}

		// Convert tool schemas
		var toolDefs []anthropic.ToolDefinition
		for _, ts := range toolSchemas {
			var schemaObj map[string]any
			if err := json.Unmarshal([]byte(ts.JSONSchema), &schemaObj); err != nil {
				errCh <- fmt.Errorf("invalid tool schema JSON for %s: %w", ts.Name, err)
				return
			}
			toolDefs = append(toolDefs, anthropic.ToolDefinition{
				Name:        ts.Name,
				Description: ts.Description,
				InputSchema: schemaObj,
			})
		}

		// Build streaming request
		maxTokens := 4096
		if opts.MaxOutputTokens > 0 {
			maxTokens = opts.MaxOutputTokens
		}
		temperature := float32(0.1)
		if opts.Temperature > 0 {
			temperature = opts.Temperature
		}

		req := anthropic.MessagesStreamRequest{
			MessagesRequest: anthropic.MessagesRequest{
				Model:       anthropic.Model(modelName),
				Messages:    anthropicMsgs,
				MaxTokens:   maxTokens,
				Temperature: &temperature,
			},
		}

		if len(systemParts) > 0 {
			req.MultiSystem = systemParts
		}
		if len(toolDefs) > 0 {
			req.Tools = toolDefs
		}

		// Set up streaming callbacks
		req.OnError = func(errResp anthropic.ErrorResponse) {
			errCh <- fmt.Errorf("anthropic streaming error: %s", errResp.Error.Message)
		}

		req.OnContentBlockDelta = func(delta anthropic.MessagesEventContentBlockDeltaData) {
			// Handle text deltas
			if delta.Delta.Type == "text_delta" && delta.Delta.Text != nil {
				select {
				case eventCh <- engine.StreamEvent{
					Type: "text_delta",
					Text: *delta.Delta.Text,
				}:
				case <-ctx.Done():
					return
				}
			}
		}

		req.OnContentBlockStop = func(stop anthropic.MessagesEventContentBlockStopData, content anthropic.MessageContent) {
			// Handle completed tool use blocks
			if content.Type == "tool_use" && content.MessageContentToolUse != nil {
				tc := content.MessageContentToolUse
				var args map[string]any
				if len(tc.Input) > 0 {
					if err := json.Unmarshal(tc.Input, &args); err != nil {
						args = make(map[string]any)
					}
				} else {
					args = make(map[string]any)
				}

				toolCall := engine.ToolCall{
					ID:   tc.ID,
					Name: tc.Name,
					Args: args,
				}

				// Emit tool call event
				select {
				case eventCh <- engine.StreamEvent{
					Type:     "tool_call",
					ToolCall: toolCall,
				}:
				case <-ctx.Done():
					return
				}
			}
		}

		// Track usage separately (Anthropic may provide it in response, not callbacks)
		var finalUsage engine.Usage
		req.OnMessageStop = func(stop anthropic.MessagesEventMessageStopData) {
			// Message stop event - stream is complete
			// Note: Usage information may be available in the response object returned by CreateMessagesStream
			// For now, we'll emit a zero usage (can be enhanced if SDK provides usage in callbacks)
		}

		// Call streaming API
		// Note: CreateMessagesStream returns a MessagesResponse which may contain usage info
		resp, err := c.client.CreateMessagesStream(ctx, req)
		if err != nil {
			// Extract HTTP status and Retry-After from error
			httpStatus, retryAfter := extractErrorMetadata(err)
			wrappedErr := engine.WrapLLMError(err, httpStatus, retryAfter)
			errCh <- wrappedErr
			return
		}

		// Extract usage from response if available
		if resp.Usage.InputTokens > 0 {
			finalUsage = engine.Usage{
				Prompt:     resp.Usage.InputTokens,
				Completion: resp.Usage.OutputTokens,
				Total:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			}
			// Emit usage event
			select {
			case eventCh <- engine.StreamEvent{
				Type:  "usage",
				Usage: finalUsage,
			}:
			case <-ctx.Done():
				return
			}
		}

		// Note: Anthropic's streaming API uses callbacks, so callbacks are invoked
		// during CreateMessagesStream execution. Events are sent via callbacks above.
	}()

	return eventCh, errCh
}
