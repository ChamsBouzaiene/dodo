package engine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// getRetryConfig returns the retry configuration, using defaults if not provided.
func getRetryConfig(opts ChatOptions) *RetryConfig {
	if opts.RetryConfig != nil {
		return opts.RetryConfig
	}
	defaultConfig := DefaultRetryConfig()
	return &defaultConfig
}

// callRetryHooks calls OnRetryAttempt on all hooks.
func callRetryHooks(hooks Hooks, ctx context.Context, st *State, attempt, maxAttempts int, delay time.Duration, err error) {
	for _, hook := range hooks {
		hook.OnRetryAttempt(ctx, st, attempt, maxAttempts, delay, err)
	}
}

// handleRetryExhaustion calls OnRetryExhausted on all hooks if the error indicates retries were exhausted.
func handleRetryExhaustion(hooks Hooks, ctx context.Context, st *State, err error) {
	if IsRetryExhausted(err) {
		for _, hook := range hooks {
			hook.OnRetryExhausted(ctx, st, err)
		}
	}
}

// toolResult represents the result of executing a tool call.
type toolResult struct {
	idx     int
	content string
	err     error
	call    ToolCall
}

// executeToolsWithRetry executes tool calls with retry logic and returns results in order.
func executeToolsWithRetry(ctx context.Context, calls []ToolCall, reg ToolRegistry, retryConfig *RetryConfig, hooks Hooks, st *State) ([]toolResult, error) {
	if len(calls) == 0 {
		return nil, nil
	}

	var wg sync.WaitGroup
	results := make([]toolResult, len(calls))

	for i, call := range calls {
		wg.Add(1)
		go func(i int, c ToolCall) {
			defer wg.Done()

			// Check cancellation before execution
			select {
			case <-ctx.Done():
				results[i] = toolResult{
					idx:  i,
					err:  ctx.Err(),
					call: c,
				}
				return
			default:
			}

			hooks.OnToolCall(ctx, st, c)

			// Wrap tool call with retry logic
			res, err := RetryToolCall(
				ctx,
				retryConfig.ToolPolicy,
				c,
				reg,
				func(attempt int, delay time.Duration, retryErr error) {
					callRetryHooks(hooks, ctx, st, attempt, retryConfig.ToolPolicy.MaxRetries, delay, retryErr)
				},
			)
			handleRetryExhaustion(hooks, ctx, st, err)
			results[i] = toolResult{idx: i, content: res, err: err, call: c}
		}(i, call)
	}

	wg.Wait()
	return results, nil
}

// prepareMessages builds and processes messages for an LLM call.
// It applies processors and enforces budget limits with compression if needed.
func prepareMessages(ctx context.Context, st *State, llm LLMClient, hooks Hooks, compressionCfg *CompressionConfig) ([]ChatMessage, error) {
	msgs := append([]ChatMessage(nil), st.History...)

	// Use provided config or default
	cfg := compressionCfg
	if cfg == nil {
		defaultCfg := DefaultCompressionConfig()
		cfg = &defaultCfg
	}

	// Apply processors based on config
	if cfg.Enabled {
		var err error
		var processors []Processor
		if cfg.KeepRecentCount > 0 {
			processors = append(processors, KeepRecentToolCalls(cfg.KeepRecentCount))
		}
		processors = append(processors, SummarizeOlderThanN(llm, cfg.SummarizeThreshold))
		if cfg.KeepRecentCount > 0 {
			processors = append(processors, KeepLastN(cfg.KeepRecentCount))
		}
		processors = append(processors, TruncateLongTools(cfg.TruncateToolsAt))

		msgs, err = ApplyProcessors(ctx, st, msgs, processors...)
		if err != nil {
			return nil, err
		}
	}

	// Check and enforce budget if configured
	if st.Budget.HardLimit > 0 {
		tokenizer := GetTokenizerForModel(st.Model)

		// Count tokens before compression
		beforeTokens, err := CountTokensForMessages(tokenizer, msgs, st.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to count tokens: %w", err)
		}

		// Check if we're over soft limit (warn but don't fail)
		if st.Budget.SoftLimit > 0 && beforeTokens > st.Budget.SoftLimit {
			hooks.OnBudgetExceeded(ctx, st, beforeTokens, st.Budget.SoftLimit, st.Budget.HardLimit)
		}

		// Compress if over hard limit
		effectiveHardLimit := st.Budget.HardLimit - st.Budget.ReserveTokens
		if beforeTokens > effectiveHardLimit {
			compressedMsgs, _, err := compressUntilUnderBudget(
				ctx, llm, st, msgs, st.Budget, tokenizer,
				func(before, after int, strategy CompressionStrategy) {
					// Log each compression step
					hooks.OnBudgetCompression(ctx, st, before, after, strategy)
				},
			)
			if err != nil {
				return nil, fmt.Errorf("budget compression failed: %w", err)
			}
			msgs = compressedMsgs
		}
	}

	return msgs, nil
}

// buildMessagesForCall is deprecated, use prepareMessages instead.
func buildMessagesForCall(ctx context.Context, st *State, llm LLMClient, processors []Processor) ([]ChatMessage, error) {
	// Create empty hooks for backward compatibility
	return prepareMessages(ctx, st, llm, Hooks{}, nil)
}

func executeTool(ctx context.Context, call ToolCall, reg ToolRegistry) (string, error) {
	t, ok := reg[call.Name]
	if !ok {
		return "", fmt.Errorf("tool not found: %s (available tools: %v)", call.Name, getToolNames(reg))
	}

	// Validate args against schema
	if err := t.ValidateArgs(call.Args); err != nil {
		return "", fmt.Errorf("validation failed for tool %s: %w", call.Name, err)
	}

	result, err := t.Fn(ctx, call.Args)
	if err != nil {
		return "", fmt.Errorf("execution failed for tool %s: %w", call.Name, err)
	}

	return result, nil
}

// getToolNames returns a list of available tool names for error messages.
func getToolNames(reg ToolRegistry) []string {
	names := make([]string, 0, len(reg))
	for name := range reg {
		names = append(names, name)
	}
	return names
}

// callLLMWithRetry calls the LLM with retry logic and returns the response.
func callLLMWithRetry(ctx context.Context, llm LLMClient, model string, msgs []ChatMessage, schemas []ToolSchema, opts ChatOptions, retryConfig *RetryConfig, hooks Hooks, st *State) (LLMResponse, error) {
	resp, err := RetryLLMCall(
		ctx,
		retryConfig.LLMPolicy,
		llm,
		model,
		msgs,
		schemas,
		opts,
		func(attempt int, delay time.Duration, retryErr error) {
			callRetryHooks(hooks, ctx, st, attempt, retryConfig.LLMPolicy.MaxRetries, delay, retryErr)
		},
	)
	if err != nil {
		handleRetryExhaustion(hooks, ctx, st, err)
		return LLMResponse{}, err
	}
	return resp, nil
}

// processLLMResponse processes the LLM response: updates state, appends to history, and tracks usage.
func processLLMResponse(resp LLMResponse, st *State, hooks Hooks, ctx context.Context) {
	hooks.OnAfterLLM(ctx, st, resp)

	st.Totals.Prompt += resp.Usage.Prompt
	st.Totals.Completion += resp.Usage.Completion
	st.Totals.Total += resp.Usage.Total

	// Always append assistant message with tool calls (if any)
	assistantMsg := resp.Assistant
	assistantMsg.ToolCalls = resp.ToolCalls // Store tool calls for proper message reconstruction
	st.Append(assistantMsg)
	hooks.OnHistoryChanged(ctx, st)
}

// executeToolCalls executes tool calls and appends results to history.
// It enforces edit tool blocking if internal planning is enabled.
func executeToolCalls(ctx context.Context, calls []ToolCall, reg ToolRegistry, retryConfig *RetryConfig, hooks Hooks, st *State) error {
	if len(calls) == 0 {
		return nil
	}

	// Check if any tool call is blocked by planning enforcement
	if st.EditToolBlocked {
		var blockedCalls []ToolCall
		var allowedCalls []ToolCall

		for _, call := range calls {
			if isEditTool(call.Name) {
				blockedCalls = append(blockedCalls, call)
			} else {
				allowedCalls = append(allowedCalls, call)
			}
		}

		// Handle blocked calls - append error messages
		for _, call := range blockedCalls {
			errorMsg := fmt.Sprintf("ERROR: Planning required before edits. Call 'plan' tool first to create an execution plan, then proceed with edits.\n\nBlocked tool: %s\n\nYou must call the 'plan' tool with your execution plan before using edit tools (search_replace, write_file, write, delete_file).", call.Name)
			st.Append(ChatMessage{
				Role:    RoleTool,
				Name:    call.ID,
				Content: errorMsg,
			})
			hooks.OnToolResult(ctx, st, call, errorMsg, fmt.Errorf("edit blocked: no plan"))
		}

		// Update calls list to only include allowed calls
		calls = allowedCalls

		// If all calls were blocked, return early after appending history
		if len(calls) == 0 {
			hooks.OnHistoryChanged(ctx, st)
			return nil
		}
	}

	// Add State to context so tools can access it
	ctx = context.WithValue(ctx, "engine_state", st)

	// Execute tools with retry logic
	results, err := executeToolsWithRetry(ctx, calls, reg, retryConfig, hooks, st)
	if err != nil {
		return err
	}

	// Append tool results in order
	// Use tool call ID (not name) for proper message ordering with providers
	for _, o := range results {
		if o.err != nil {
			o.content = "ERROR: " + o.err.Error()
		}
		// Use tool call ID as Name field (providers use this to match tool messages to tool calls)
		toolCallID := o.call.ID
		if toolCallID == "" {
			// Fallback to tool name if ID not set (shouldn't happen with proper providers)
			toolCallID = o.call.Name
		}
		st.Append(ChatMessage{Role: RoleTool, Name: toolCallID, Content: o.content})
		hooks.OnToolResult(ctx, st, o.call, o.content, o.err)
	}
	hooks.OnHistoryChanged(ctx, st)

	return nil
}

// isEditTool returns true if the tool modifies files.
// These tools are subject to planning enforcement when EditToolBlocked is true.
func isEditTool(toolName string) bool {
	editTools := map[string]bool{
		"search_replace": true,
		"write_file":     true,
		"write":          true,
		"delete_file":    true,
	}
	return editTools[toolName]
}

func stepOnce(ctx context.Context, llm LLMClient, reg ToolRegistry, st *State, hooks Hooks, opts ChatOptions) error {
	// Detect and update phase
	st.Phase = DetectPhase(st.History)
	hooks.OnStepStart(ctx, st)

	// Prepare messages
	msgs, err := prepareMessages(ctx, st, llm, hooks, opts.CompressionConfig)
	if err != nil {
		return WrapWithContext(err, st, "message_preparation", "")
	}

	// Get retry config
	retryConfig := getRetryConfig(opts)

	// Get tool schemas
	toolSchemas := reg.Schemas()

	// Log full prompt and tools before LLM call
	hooks.OnBeforeLLM(ctx, st, msgs, toolSchemas)

	// Call LLM with retry logic
	resp, err := callLLMWithRetry(ctx, llm, st.Model, msgs, toolSchemas, opts, retryConfig, hooks, st)
	if err != nil {
		return WrapWithContext(err, st, "llm_call", "")
	}

	// Process LLM response
	processLLMResponse(resp, st, hooks, ctx)

	// Check if we're done (no tool calls)
	if len(resp.ToolCalls) == 0 {
		st.Done = true
		return nil
	}

	// Execute tool calls
	if err := executeToolCalls(ctx, resp.ToolCalls, reg, retryConfig, hooks, st); err != nil {
		return WrapWithContext(err, st, "tool_execution", "")
	}

	// Check if 'respond' tool was called - this signals task completion
	for _, call := range resp.ToolCalls {
		if call.Name == "respond" {
			st.Done = true
			break
		}
	}

	return nil // next loop iteration will continue the ReAct flow
}
