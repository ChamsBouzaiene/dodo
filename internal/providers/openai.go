package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"

	openai "github.com/meguminnnnnnnnn/go-openai"
)

// OpenAIClient implements engine.LLMClient by calling OpenAI SDK directly.
// This avoids any dependency on Eino.
type OpenAIClient struct {
	client  *openai.Client
	model   string
	baseURL string
}

// NewOpenAIClient creates a new OpenAI client for the engine.
func NewOpenAIClient(apiKey, modelName, baseURL string) (*OpenAIClient, error) {
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)

	return &OpenAIClient{
		client:  client,
		model:   modelName,
		baseURL: baseURL,
	}, nil
}

// Chat implements engine.LLMClient.Chat by calling OpenAI API directly.
func (c *OpenAIClient) Chat(ctx context.Context, modelName string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (engine.LLMResponse, error) {
	// Convert engine messages to OpenAI format
	openaiMsgs := make([]openai.ChatCompletionMessage, 0, len(messages))
	var systemMsg string

	// Track if previous assistant message had tool calls
	var prevAssistantHadToolCalls bool

	for i, msg := range messages {
		switch msg.Role {
		case engine.RoleSystem:
			systemMsg = msg.Content
			prevAssistantHadToolCalls = false // Reset after system message
		case engine.RoleUser:
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: msg.Content,
			})
			prevAssistantHadToolCalls = false // Reset after user message
		case engine.RoleAssistant:
			// OpenAI allows empty content for assistant messages with tool calls
			// However, the SDK might serialize empty string as null, causing API errors
			// Use a space character if content is empty to avoid null serialization
			content := msg.Content
			if content == "" {
				// Use a single space instead of empty string to avoid null serialization
				// OpenAI accepts this and it's semantically equivalent to empty
				content = " "
			}

			// Convert tool calls to OpenAI format if present
			var toolCalls []openai.ToolCall
			if len(msg.ToolCalls) > 0 {
				toolCalls = make([]openai.ToolCall, 0, len(msg.ToolCalls))
				for _, tc := range msg.ToolCalls {
					argsJSON, _ := json.Marshal(tc.Args)
					toolCalls = append(toolCalls, openai.ToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openai.FunctionCall{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					})
				}
			}

			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   content,
				ToolCalls: toolCalls, // Include tool_calls if present
			})
			// Track if this assistant message had tool calls
			prevAssistantHadToolCalls = len(msg.ToolCalls) > 0
		case engine.RoleTool:
			// OpenAI requires tool messages to follow an assistant message with tool_calls
			// Skip tool messages if previous assistant didn't have tool calls
			if !prevAssistantHadToolCalls {
				// This shouldn't happen in normal flow, but skip to avoid API error
				continue
			}
			// Ensure content is never empty (OpenAI rejects null/empty content)
			content := msg.Content
			if content == "" {
				content = "{}" // Empty JSON object instead of empty string
			}
			// msg.Name contains the tool_call_id from when we stored the tool result
			openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				ToolCallID: msg.Name, // This is the tool_call_id, not the tool name
				Content:    content,
			})
			// Check if next message is assistant (to reset tracking)
			if i+1 < len(messages) && messages[i+1].Role == engine.RoleAssistant {
				prevAssistantHadToolCalls = false
			}
		}
	}

	// Convert tool schemas to OpenAI format
	var tools []openai.Tool
	for _, ts := range toolSchemas {
		var schemaObj map[string]any
		if err := json.Unmarshal([]byte(ts.JSONSchema), &schemaObj); err != nil {
			return engine.LLMResponse{}, fmt.Errorf("invalid tool schema JSON for %s: %w", ts.Name, err)
		}

		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        ts.Name,
				Description: ts.Description,
				Parameters:  schemaObj,
			},
		})
	}

	// Build request
	req := openai.ChatCompletionRequest{
		Model:    modelName,
		Messages: openaiMsgs,
	}

	// Add system message if present
	if systemMsg != "" {
		req.Messages = append([]openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemMsg,
		}}, req.Messages...)
	}

	// Add tools if present
	if len(tools) > 0 {
		req.Tools = tools
		// Explicitly set tool_choice to "auto" so the model decides when to use tools
		// This can also be "none" (never use tools) or "required" (must use tools)
		req.ToolChoice = "auto"
	}

	// Apply options
	if opts.MaxOutputTokens > 0 {
		req.MaxTokens = opts.MaxOutputTokens
	}
	if opts.Temperature > 0 {
		req.Temperature = &opts.Temperature
	}

	// Call OpenAI API
	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		// Extract HTTP status and Retry-After from error
		httpStatus, retryAfter := extractErrorMetadata(err)
		wrappedErr := engine.WrapLLMError(err, httpStatus, retryAfter)
		return engine.LLMResponse{}, wrappedErr
	}

	if len(resp.Choices) == 0 {
		return engine.LLMResponse{}, fmt.Errorf("empty response from OpenAI")
	}

	choice := resp.Choices[0]

	// Extract assistant message
	// OpenAI may return empty content when only tool calls are present
	// Ensure we always have at least an empty string (not null)
	content := choice.Message.Content
	if content == "" {
		content = "" // Explicitly set to empty string
	}
	assistantMsg := engine.ChatMessage{
		Role:    engine.RoleAssistant,
		Content: content,
	}

	// Parse tool calls
	var toolCalls []engine.ToolCall
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls = make([]engine.ToolCall, 0, len(choice.Message.ToolCalls))
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]any
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					args = make(map[string]any)
				}
			} else {
				args = make(map[string]any)
			}

			toolCalls = append(toolCalls, engine.ToolCall{
				ID:   tc.ID, // Store the provider's tool call ID
				Name: tc.Function.Name,
				Args: args,
			})
		}
		// Store tool calls in assistant message for proper message reconstruction
		assistantMsg.ToolCalls = toolCalls
	}

	// Determine finish reason
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	} else if choice.FinishReason == openai.FinishReasonLength {
		finishReason = "length"
	} else if choice.FinishReason == openai.FinishReasonContentFilter {
		finishReason = "content_filter"
	}

	// Extract usage
	usage := engine.Usage{
		Prompt:     resp.Usage.PromptTokens,
		Completion: resp.Usage.CompletionTokens,
		Total:      resp.Usage.TotalTokens,
	}

	return engine.LLMResponse{
		Assistant:    assistantMsg,
		ToolCalls:    toolCalls,
		Usage:        usage,
		FinishReason: finishReason,
	}, nil
}

// Stream implements engine.LLMClient.Stream with real streaming support.
func (c *OpenAIClient) Stream(ctx context.Context, modelName string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (<-chan engine.StreamEvent, <-chan error) {
	eventCh := make(chan engine.StreamEvent, 10) // Buffered to avoid blocking
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		// Note: errCh is closed after sending nil (or error) to signal completion

		// Convert messages and tools (reuse logic from Chat())
		openaiMsgs := make([]openai.ChatCompletionMessage, 0, len(messages))
		var systemMsg string
		var prevAssistantHadToolCalls bool

		for i, msg := range messages {
			switch msg.Role {
			case engine.RoleSystem:
				systemMsg = msg.Content
				prevAssistantHadToolCalls = false
			case engine.RoleUser:
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: msg.Content,
				})
				prevAssistantHadToolCalls = false
			case engine.RoleAssistant:
				content := msg.Content
				if content == "" {
					content = " "
				}
				var toolCalls []openai.ToolCall
				if len(msg.ToolCalls) > 0 {
					toolCalls = make([]openai.ToolCall, 0, len(msg.ToolCalls))
					for _, tc := range msg.ToolCalls {
						argsJSON, _ := json.Marshal(tc.Args)
						toolCalls = append(toolCalls, openai.ToolCall{
							ID:   tc.ID,
							Type: "function",
							Function: openai.FunctionCall{
								Name:      tc.Name,
								Arguments: string(argsJSON),
							},
						})
					}
				}
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:      openai.ChatMessageRoleAssistant,
					Content:   content,
					ToolCalls: toolCalls,
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
				openaiMsgs = append(openaiMsgs, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					ToolCallID: msg.Name,
					Content:    content,
				})
				if i+1 < len(messages) && messages[i+1].Role == engine.RoleAssistant {
					prevAssistantHadToolCalls = false
				}
			}
		}

		// Convert tool schemas
		var tools []openai.Tool
		for _, ts := range toolSchemas {
			var schemaObj map[string]any
			if err := json.Unmarshal([]byte(ts.JSONSchema), &schemaObj); err != nil {
				errCh <- fmt.Errorf("invalid tool schema JSON for %s: %w", ts.Name, err)
				return
			}
			tools = append(tools, openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        ts.Name,
					Description: ts.Description,
					Parameters:  schemaObj,
				},
			})
		}

		// Build streaming request
		req := openai.ChatCompletionRequest{
			Model:    modelName,
			Messages: openaiMsgs,
			Stream:   true, // Enable streaming
			StreamOptions: &openai.StreamOptions{
				IncludeUsage: true, // Include usage in final chunk
			},
		}

		if systemMsg != "" {
			req.Messages = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemMsg,
			}}, req.Messages...)
		}

		if len(tools) > 0 {
			req.Tools = tools
			req.ToolChoice = "auto"
		}

		if opts.MaxOutputTokens > 0 {
			req.MaxTokens = opts.MaxOutputTokens
		}
		if opts.Temperature > 0 {
			req.Temperature = &opts.Temperature
		}

		// Create stream
		stream, err := c.client.CreateChatCompletionStream(ctx, req)
		if err != nil {
			httpStatus, retryAfter := extractErrorMetadata(err)
			wrappedErr := engine.WrapLLMError(err, httpStatus, retryAfter)
			errCh <- wrappedErr
			return
		}
		defer stream.Close()

		// Track tool calls being accumulated (OpenAI sends deltas per field)
		// We need to track both the ToolCall and the raw JSON string for arguments
		type toolCallAccumulator struct {
			toolCall *engine.ToolCall
			argsJSON strings.Builder // Accumulate raw JSON string (may be partial)
			index    int             // Index of this tool call (for ordering)
		}
		toolCallAccum := make(map[string]*toolCallAccumulator) // Map by tool call ID
		toolCallIndex := 0
		var finalUsage engine.Usage

		// Read from stream
		for {
			response, err := stream.Recv()
			if err != nil {
				// Check if it's EOF (normal completion)
				// Check for io.EOF or wrapped EOF errors
				if errors.Is(err, io.EOF) {
					// Stream completed normally
				} else {
					// Also check error string in case SDK wraps it differently
					errStr := err.Error()
					if !(strings.Contains(errStr, "EOF") || strings.Contains(errStr, "end of file")) {
						// Real error, not EOF
						httpStatus, retryAfter := extractErrorMetadata(err)
						wrappedErr := engine.WrapLLMError(err, httpStatus, retryAfter)
						errCh <- wrappedErr
						close(errCh)
						return
					}
				}
				// Handle EOF - stream completed normally
				{
					// Stream completed normally, emit completed tool calls first
					// Sort by index to maintain order
					type indexedCall struct {
						index int
						tc    *engine.ToolCall
					}
					var sortedCalls []indexedCall

					for _, acc := range toolCallAccum {
						// Parse accumulated JSON arguments
						if acc.argsJSON.Len() > 0 {
							var args map[string]any
							argsStr := acc.argsJSON.String()

							// Try to parse JSON
							if err := json.Unmarshal([]byte(argsStr), &args); err == nil {
								acc.toolCall.Args = args
								// Log successful completion with summary (to stderr, not stdout)
								fmt.Fprintf(os.Stderr, "✅ Tool call completed: %s (ID: %s) with %d args (%d bytes JSON)\n",
									acc.toolCall.Name, acc.toolCall.ID, len(args), acc.argsJSON.Len())
								sortedCalls = append(sortedCalls, indexedCall{
									index: acc.index,
									tc:    acc.toolCall,
								})
							} else {
								// JSON parsing failed - likely incomplete JSON
								// Check if JSON looks incomplete (doesn't end with })
								trimmed := strings.TrimSpace(argsStr)
								isIncomplete := !strings.HasSuffix(trimmed, "}") && !strings.HasSuffix(trimmed, "]")

								if isIncomplete {
									// Show last 100 chars for context
									preview := trimmed
									if len(preview) > 100 {
										preview = "..." + preview[len(preview)-100:]
									}
									fmt.Fprintf(os.Stderr, "❌ Tool call %s (ID: %s) has INCOMPLETE JSON (%d bytes) - stream ended prematurely\n",
										acc.toolCall.Name, acc.toolCall.ID, acc.argsJSON.Len())
									fmt.Fprintf(os.Stderr, "   Last 100 chars: %q\n", preview)
									// Set error field on ToolCall - engine will handle it
									acc.toolCall.Error = fmt.Sprintf("Stream ended prematurely (%d bytes received). Arguments incomplete. This usually indicates MaxOutputTokens is too low.", acc.argsJSON.Len())
									sortedCalls = append(sortedCalls, indexedCall{
										index: acc.index,
										tc:    acc.toolCall,
									})
								} else {
									fmt.Fprintf(os.Stderr, "❌ Tool call %s (ID: %s) has INVALID JSON: %v (%d bytes)\n",
										acc.toolCall.Name, acc.toolCall.ID, err, acc.argsJSON.Len())
									// Set error field on ToolCall - engine will handle it
									acc.toolCall.Error = fmt.Sprintf("Invalid JSON in arguments: %v. Please check syntax and retry.", err)
									sortedCalls = append(sortedCalls, indexedCall{
										index: acc.index,
										tc:    acc.toolCall,
									})
								}
							}
						} else {
							// No arguments were accumulated
							if acc.toolCall.Name != "" {
								fmt.Fprintf(os.Stderr, "⚠️  Tool call %s (ID: %s) has no arguments - LLM sent incomplete tool call\n",
									acc.toolCall.Name, acc.toolCall.ID)
								// Set error field on ToolCall - engine will handle it
								acc.toolCall.Error = "No arguments received. Please retry with complete arguments."
								acc.toolCall.Args = make(map[string]any) // Empty args
								sortedCalls = append(sortedCalls, indexedCall{
									index: acc.index,
									tc:    acc.toolCall,
								})
							} else {
								// Tool call with no name - skip it
								continue
							}
						}
					}
					// Sort by index
					for i := 0; i < len(sortedCalls)-1; i++ {
						for j := i + 1; j < len(sortedCalls); j++ {
							if sortedCalls[i].index > sortedCalls[j].index {
								sortedCalls[i], sortedCalls[j] = sortedCalls[j], sortedCalls[i]
							}
						}
					}
					// Emit all tool calls in order (engine will handle errors via ToolCall.Error field)
					for _, ic := range sortedCalls {
						select {
						case eventCh <- engine.StreamEvent{
							Type:     "tool_call",
							ToolCall: *ic.tc,
						}:
						case <-ctx.Done():
							return
						}
					}
					// Emit final usage if available
					if finalUsage.Total > 0 {
						select {
						case eventCh <- engine.StreamEvent{Type: "usage", Usage: finalUsage}:
						case <-ctx.Done():
							return
						}
					}
					// Signal successful completion by sending nil to errCh
					// This allows step_stream.go to detect completion
					select {
					case errCh <- nil:
					case <-ctx.Done():
						return
					}
					// Close errCh after sending nil to signal completion
					close(errCh)
					return
				}
			}

			// Process stream response
			// Note: Final chunk may have usage but no choices (when stream_options.include_usage is true)
			// So we check usage first before checking choices
			if response.Usage != nil && response.Usage.TotalTokens > 0 {
				finalUsage = engine.Usage{
					Prompt:     response.Usage.PromptTokens,
					Completion: response.Usage.CompletionTokens,
					Total:      response.Usage.TotalTokens,
				}
			}

			// Process choices if present
			if len(response.Choices) == 0 {
				continue
			}

			choice := response.Choices[0]
			delta := choice.Delta

			// Handle text content delta
			if delta.Content != "" {
				select {
				case eventCh <- engine.StreamEvent{
					Type: "text_delta",
					Text: delta.Content,
				}:
				case <-ctx.Done():
					return
				}
			}

			// Handle tool call deltas (accumulate by ID)
			if len(delta.ToolCalls) > 0 {
				for _, tcDelta := range delta.ToolCalls {
					// Tool calls may arrive without ID initially (index-based)
					// We need to handle both cases: by ID and by index
					var acc *toolCallAccumulator
					var found bool

					if tcDelta.ID != "" {
						// Use ID if available
						acc, found = toolCallAccum[tcDelta.ID]
						if !found {
							acc = &toolCallAccumulator{
								toolCall: &engine.ToolCall{
									ID:   tcDelta.ID,
									Args: make(map[string]any),
								},
								index: toolCallIndex,
							}
							toolCallAccum[tcDelta.ID] = acc
							toolCallIndex++
						}
					} else if tcDelta.Index != nil {
						// Fallback to index if ID not available yet
						// Find accumulator by index (less reliable, but works)
						for _, existingAcc := range toolCallAccum {
							if existingAcc.index == *tcDelta.Index {
								acc = existingAcc
								found = true
								break
							}
						}
						if !found {
							// Create new accumulator with temporary ID
							tempID := fmt.Sprintf("temp_%d", *tcDelta.Index)
							acc = &toolCallAccumulator{
								toolCall: &engine.ToolCall{
									ID:   tempID,
									Args: make(map[string]any),
								},
								index: *tcDelta.Index,
							}
							toolCallAccum[tempID] = acc
						}
					} else {
						continue // Skip if neither ID nor index available
					}

					// Update ID if we get it later
					if tcDelta.ID != "" && acc.toolCall.ID != tcDelta.ID {
						// Update the ID and move accumulator to new key
						oldID := acc.toolCall.ID
						acc.toolCall.ID = tcDelta.ID
						if oldID != tcDelta.ID {
							delete(toolCallAccum, oldID)
							toolCallAccum[tcDelta.ID] = acc
						}
					}

					// Accumulate function name
					if tcDelta.Function.Name != "" {
						acc.toolCall.Name = tcDelta.Function.Name
					}

					// Accumulate function arguments (JSON string, may be partial)
					if tcDelta.Function.Arguments != "" {
						// Track if this is the first chunk for this tool call
						isFirstChunk := acc.argsJSON.Len() == 0
						acc.argsJSON.WriteString(tcDelta.Function.Arguments)
						// Only log when tool call started (first chunk) - to stderr, not stdout
						if isFirstChunk && acc.toolCall.Name != "" {
							fmt.Fprintf(os.Stderr, "🔧 Tool call started: %s (ID: %s)\n", acc.toolCall.Name, acc.toolCall.ID)
						}
					}
				}
			}

			// Note: Usage is handled above before checking choices
			// This ensures we capture usage even if the final chunk has no choices
		}
	}()

	return eventCh, errCh
}

// extractErrorMetadata extracts HTTP status code and Retry-After header from an error.
// This is a helper function to extract metadata from SDK errors.
func extractErrorMetadata(err error) (int, string) {
	if err == nil {
		return 0, ""
	}

	errStr := err.Error()
	var httpStatus int
	var retryAfter string

	// Try to extract HTTP status code from error message
	// Common patterns: "429", "status code 429", "HTTP 429", etc.
	if strings.Contains(errStr, "429") {
		httpStatus = http.StatusTooManyRequests
	} else if strings.Contains(errStr, "500") {
		httpStatus = http.StatusInternalServerError
	} else if strings.Contains(errStr, "502") {
		httpStatus = http.StatusBadGateway
	} else if strings.Contains(errStr, "503") {
		httpStatus = http.StatusServiceUnavailable
	} else if strings.Contains(errStr, "504") {
		httpStatus = http.StatusGatewayTimeout
	} else if strings.Contains(errStr, "401") {
		httpStatus = http.StatusUnauthorized
	} else if strings.Contains(errStr, "403") {
		httpStatus = http.StatusForbidden
	} else if strings.Contains(errStr, "400") {
		httpStatus = http.StatusBadRequest
	} else if strings.Contains(errStr, "402") {
		httpStatus = http.StatusPaymentRequired
	}

	// Try to extract Retry-After from error message
	// Common patterns: "Retry-After: 60", "retry after 60", etc.
	if idx := strings.Index(strings.ToLower(errStr), "retry-after"); idx != -1 {
		// Try to extract the value after "retry-after"
		remaining := errStr[idx+11:] // Skip "retry-after"
		parts := strings.Fields(remaining)
		if len(parts) > 0 {
			retryAfter = parts[0]
		}
	} else if idx := strings.Index(strings.ToLower(errStr), "retry after"); idx != -1 {
		remaining := errStr[idx+12:] // Skip "retry after"
		parts := strings.Fields(remaining)
		if len(parts) > 0 {
			retryAfter = parts[0]
		}
	}

	return httpStatus, retryAfter
}
