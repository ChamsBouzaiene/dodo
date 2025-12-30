package engine

import (
	"context"

	"github.com/ChamsBouzaiene/dodo/internal/prompts"
)

// Agent represents an agent instance that can run conversations.
type Agent struct {
	llm       LLMClient
	tools     ToolRegistry
	config    AgentConfig
	hooks     Hooks
	prompt    *prompts.Prompt
	lastState *State
}

// Run executes a single user message through the agent.
// It maintains conversation history across multiple calls.
func (a *Agent) Run(ctx context.Context, userMessage string) error {
	var st *State

	// If we have previous state, reuse it (preserving conversation history)
	if a.lastState != nil && len(a.lastState.History) > 0 {
		// Create new state but preserve history
		st = &State{
			History:         make([]ChatMessage, len(a.lastState.History)),
			Model:           a.config.Model,
			MaxSteps:        a.config.MaxSteps,
			Budget:          a.config.Budget,
			Totals:          a.lastState.Totals, // Preserve accumulated token usage
			EditToolBlocked: a.config.EnforcePlanning,
			FailureCounts:   make(map[string]int),
			FileReadCache:   make(map[string]bool),
			ToolCallCount:   0,
			MiniPlan:        nil,
		}
		// Copy history from previous state
		copy(st.History, a.lastState.History)

		// Preserve file read cache and failure counts from previous run
		if a.lastState.FileReadCache != nil {
			st.FileReadCache = make(map[string]bool)
			for k, v := range a.lastState.FileReadCache {
				st.FileReadCache[k] = v
			}
		}
		if a.lastState.FailureCounts != nil {
			st.FailureCounts = make(map[string]int)
			for k, v := range a.lastState.FailureCounts {
				st.FailureCounts[k] = v
			}
		}
	} else {
		// First run: create new state with system prompt
		st = &State{
			History: []ChatMessage{
				{Role: RoleSystem, Content: a.prompt.Content},
			},
			Model:           a.config.Model,
			MaxSteps:        a.config.MaxSteps,
			Budget:          a.config.Budget,
			EditToolBlocked: a.config.EnforcePlanning,
			FailureCounts:   make(map[string]int),
			FileReadCache:   make(map[string]bool),
			ToolCallCount:   0,
			MiniPlan:        nil,
		}
	}

	// Add user message
	st.Append(ChatMessage{
		Role:    RoleUser,
		Content: userMessage,
	})

	// Build options
	maxOutputTokens := a.config.MaxOutputTokens
	if maxOutputTokens == 0 {
		maxOutputTokens = 8192 // Default fallback if not configured
	}
	opts := ChatOptions{
		MaxOutputTokens:   maxOutputTokens,
		RetryConfig:       a.config.RetryConfig,
		CompressionConfig: a.config.CompressionConfig,
		Stream:            a.config.Streaming,
	}

	// Run engine
	if a.config.Streaming {
		err := RunStream(ctx, a.llm, a.tools, st, a.hooks, opts)
		a.lastState = st
		return err
	}
	err := Run(ctx, a.llm, a.tools, st, a.hooks, opts)
	a.lastState = st
	return err
}

// Append adds a message to the agent's conversation history.
// This allows for multi-turn conversations and external message injection.
// Messages appended here will be preserved in the next Run() call.
func (a *Agent) Append(msg ChatMessage) {
	if a.lastState == nil {
		// Create new state with system prompt
		if a.prompt == nil {
			// Safety check: if prompt is nil, create empty state
			// This shouldn't happen if agent is built correctly, but handle gracefully
			a.lastState = &State{
				History:         []ChatMessage{},
				Model:           a.config.Model,
				MaxSteps:        a.config.MaxSteps,
				Budget:          a.config.Budget,
				EditToolBlocked: a.config.EnforcePlanning,
				FailureCounts:   make(map[string]int),
				FileReadCache:   make(map[string]bool),
				ToolCallCount:   0,
				MiniPlan:        nil,
			}
		} else {
			systemMsg := ChatMessage{
				Role:    RoleSystem,
				Content: a.prompt.Content,
			}
			a.lastState = &State{
				History:         []ChatMessage{systemMsg},
				Model:           a.config.Model,
				MaxSteps:        a.config.MaxSteps,
				Budget:          a.config.Budget,
				EditToolBlocked: a.config.EnforcePlanning,
				FailureCounts:   make(map[string]int),
				FileReadCache:   make(map[string]bool),
				ToolCallCount:   0,
				MiniPlan:        nil,
			}
		}
	}

	// Append message to existing history
	a.lastState.Append(msg)
}

// LastState returns the most recent conversation state after Run completes.
// Callers should treat the returned state as read-only.
func (a *Agent) LastState() *State {
	return a.lastState
}

// SetLLM replaces the agent's LLM client and model name at runtime.
// This allows hot-swapping the LLM provider/model without creating a new agent.
// Conversation history is preserved across the swap.
// This method is safe to call even while the agent is running.
func (a *Agent) SetLLM(client LLMClient, modelName string) {
	a.llm = client
	a.config.Model = modelName

	// Update model in lastState if it exists
	if a.lastState != nil {
		a.lastState.Model = modelName
	}
}
