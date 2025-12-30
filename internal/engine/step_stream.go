package engine

import (
	"context"
	"fmt"
	"strings"
)

// stepOnceStream executes one step of the ReAct loop with streaming support.
// It processes streaming LLM responses and executes tools with retry logic.
func stepOnceStream(ctx context.Context, llm LLMClient, reg ToolRegistry, st *State, hooks Hooks, opts ChatOptions) error {
	// Detect and update phase
	st.Phase = DetectPhase(st.History)
	hooks.OnStepStart(ctx, st)

	// Prepare messages
	msgs, err := prepareMessages(ctx, st, llm, hooks, nil) // Use default compression config for streaming
	if err != nil {
		return err
	}

	// Get retry config
	retryConfig := getRetryConfig(opts)

	// Get tool schemas
	toolSchemas := reg.Schemas()

	// Log full prompt and tools before LLM call
	hooks.OnBeforeLLM(ctx, st, msgs, toolSchemas)

	// Stream LLM response
	deltaCh, errCh := llm.Stream(ctx, st.Model, msgs, toolSchemas, opts)
	var assistantBuffer strings.Builder
	var respUsage Usage
	var toolCalls []ToolCall

	for {
		select {
		case ev, ok := <-deltaCh:
			if !ok {
				deltaCh = nil
				// If deltaCh closed, check if we should break
				if errCh == nil {
					break
				}
				continue
			}
			switch ev.Type {
			case "text_delta":
				assistantBuffer.WriteString(ev.Text)
				hooks.OnStreamDelta(ctx, st, ev.Text)
			case "tool_call":
				toolCalls = append(toolCalls, ev.ToolCall)
			case "usage":
				respUsage = ev.Usage
			}
		case err, ok := <-errCh:
			if !ok {
				// errCh closed, set to nil and check if we should break
				errCh = nil
				if deltaCh == nil {
					break
				}
				continue
			}
			if err != nil {
				handleRetryExhaustion(hooks, ctx, st, err)
				return err
			}
			// Received nil - successful completion
			errCh = nil
		}
		if deltaCh == nil && errCh == nil {
			break
		}
	}

	assistant := ChatMessage{Role: RoleAssistant, Content: assistantBuffer.String()}
	resp := LLMResponse{Assistant: assistant, ToolCalls: toolCalls, Usage: respUsage}

	// Process LLM response
	processLLMResponse(resp, st, hooks, ctx)

	// Check if we're done (no tool calls)
	if len(toolCalls) == 0 {
		st.Done = true
		return nil
	}

	// Separate tool calls into valid and failed (those with Error field set by provider)
	var validToolCalls []ToolCall
	var failedToolCalls []ToolCall
	
	for _, call := range toolCalls {
		if call.Error != "" {
			failedToolCalls = append(failedToolCalls, call)
		} else {
			validToolCalls = append(validToolCalls, call)
		}
	}

	// Handle failed tool calls - add error messages to history so LLM can see and retry
	if len(failedToolCalls) > 0 {
		for _, call := range failedToolCalls {
			errorMsg := fmt.Sprintf("ERROR: Tool %s failed - %s", call.Name, call.Error)
			st.Append(ChatMessage{Role: RoleTool, Name: call.ID, Content: errorMsg})
			hooks.OnToolResult(ctx, st, call, errorMsg, fmt.Errorf("%s", call.Error))
		}
		hooks.OnHistoryChanged(ctx, st)
	}

	// Execute valid tool calls with retry logic
	if len(validToolCalls) > 0 {
		if err := executeToolCalls(ctx, validToolCalls, reg, retryConfig, hooks, st); err != nil {
			return err
		}

		// Check if 'respond' tool was called - this signals task completion
		for _, call := range validToolCalls {
			if call.Name == "respond" {
				st.Done = true
				break
			}
		}
	}
	// If all tool calls failed (Error field set), we return nil to continue the loop
	// The error messages in history will prompt the LLM to retry with better arguments

	return nil // next loop iteration will continue the ReAct flow
}
