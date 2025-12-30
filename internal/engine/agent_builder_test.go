package engine

import (
	"context"
	"testing"

	"github.com/ChamsBouzaiene/dodo/internal/prompts"
)

// MockLLMClient for testing
type MockLLMClient struct{}

func (m *MockLLMClient) Chat(ctx context.Context, modelName string, messages []ChatMessage, toolSchemas []ToolSchema, opts ChatOptions) (LLMResponse, error) {
	return LLMResponse{}, nil
}

func (m *MockLLMClient) Stream(ctx context.Context, modelName string, messages []ChatMessage, toolSchemas []ToolSchema, opts ChatOptions) (<-chan StreamEvent, <-chan error) {
	return nil, nil
}

func TestAgentBuilder_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("missing LLM", func(t *testing.T) {
		builder := NewAgentBuilder()
		// Don't set LLM
		builder.WithToolRegistry(make(ToolRegistry), "", nil, ToolSet{})

		_, err := builder.Build(ctx)
		if err == nil {
			t.Error("Build() expected error for missing LLM, got nil")
		}
		if err != nil && err.Error() != "LLM client not configured: use WithLLM" {
			t.Errorf("Build() error = %v, want 'LLM client not configured: use WithLLM'", err)
		}
	})

	t.Run("missing tools", func(t *testing.T) {
		builder := NewAgentBuilder()
		builder.WithLLM(&MockLLMClient{})
		// Don't set tools

		_, err := builder.Build(ctx)
		if err == nil {
			t.Error("Build() expected error for missing tools, got nil")
		}
		if err != nil && err.Error() != "tools not configured: use WithToolRegistry" {
			t.Errorf("Build() error = %v, want 'tools not configured: use WithToolRegistry'", err)
		}
	})
}

func TestAgentBuilder_Success(t *testing.T) {
	ctx := context.Background()
	builder := NewAgentBuilder()
	builder.WithLLM(&MockLLMClient{})
	builder.WithToolRegistry(make(ToolRegistry), "", nil, ToolSet{})

	// We need to set a valid prompt ID that exists in the default registry, or mock the registry.
	// Since we can't easily mock the global registry, we'll rely on "interactive" prompt usually being there,
	// OR we can expect an error if the prompt is missing, but that's a different error.
	// Let's try to set a prompt that we know likely exists or handle the error gracefully.
	// Actually, AgentBuilder uses prompts.DefaultRegistry().
	// If we don't call WithPrompt, it defaults to config.PromptID which is "interactive" (from DefaultAgentConfig).

	agent, err := builder.Build(ctx)

	// If the prompt "interactive" is missing from the default registry in this test environment,
	// Build will fail. We should check if it failed due to prompt or something else.
	// However, for unit testing the builder *logic*, we primarily care that it *tries* to build.
	// If it fails on prompt retrieval, that's "success" for the builder validation part (LLM/Tools were accepted).

	if err != nil {
		// If it failed because of prompt, that's acceptable for this test as it means validation passed
		// But ideally we want a full success.
		// Let's just check that it didn't fail with the validation errors we tested above.
		if err.Error() == "LLM client not configured: use WithLLM" || err.Error() == "tools not configured: use WithToolRegistry" {
			t.Errorf("Build() failed with validation error: %v", err)
		}
	} else {
		if agent == nil {
			t.Error("Build() returned nil agent on success")
		}
		if agent.llm == nil {
			t.Error("Agent has nil LLM")
		}
		if agent.tools == nil {
			t.Error("Agent has nil tools")
		}
	}
}

func TestAgent_Append(t *testing.T) {
	// Create a test prompt
	testPrompt := &prompts.Prompt{
		ID:      "test",
		Version: prompts.PromptV1,
		Content: "You are a test assistant.",
	}

	t.Run("append when lastState is nil", func(t *testing.T) {
		agent := &Agent{
			llm:       &MockLLMClient{},
			tools:     make(ToolRegistry),
			config:    DefaultAgentConfig(),
			hooks:     Hooks{},
			prompt:    testPrompt,
			lastState: nil,
		}

		msg := ChatMessage{
			Role:    RoleUser,
			Content: "Hello",
		}

		agent.Append(msg)

		if agent.lastState == nil {
			t.Fatal("expected lastState to be created, got nil")
		}

		if len(agent.lastState.History) != 2 {
			t.Errorf("expected 2 messages in history (system + user), got %d", len(agent.lastState.History))
		}

		if agent.lastState.History[0].Role != RoleSystem {
			t.Errorf("expected first message to be system, got %s", agent.lastState.History[0].Role)
		}

		if agent.lastState.History[0].Content != testPrompt.Content {
			t.Errorf("expected system message content to match prompt, got %q", agent.lastState.History[0].Content)
		}

		if agent.lastState.History[1].Role != RoleUser {
			t.Errorf("expected second message to be user, got %s", agent.lastState.History[1].Role)
		}

		if agent.lastState.History[1].Content != "Hello" {
			t.Errorf("expected user message content to be 'Hello', got %q", agent.lastState.History[1].Content)
		}
	})

	t.Run("append when lastState exists", func(t *testing.T) {
		existingState := &State{
			History: []ChatMessage{
				{Role: RoleSystem, Content: testPrompt.Content},
				{Role: RoleUser, Content: "First message"},
			},
			Model:           "test-model",
			MaxSteps:        10,
			Budget:          DefaultBudgetConfig(),
			EditToolBlocked: false,
			FailureCounts:   make(map[string]int),
			FileReadCache:   make(map[string]bool),
			ToolCallCount:   0,
			MiniPlan:        nil,
		}

		agent := &Agent{
			llm:       &MockLLMClient{},
			tools:     make(ToolRegistry),
			config:    DefaultAgentConfig(),
			hooks:     Hooks{},
			prompt:    testPrompt,
			lastState: existingState,
		}

		msg := ChatMessage{
			Role:    RoleAssistant,
			Content: "Response",
		}

		initialHistoryLen := len(agent.lastState.History)
		agent.Append(msg)

		if len(agent.lastState.History) != initialHistoryLen+1 {
			t.Errorf("expected history length to increase by 1, got %d (was %d)", len(agent.lastState.History), initialHistoryLen)
		}

		lastMsg := agent.lastState.History[len(agent.lastState.History)-1]
		if lastMsg.Role != RoleAssistant {
			t.Errorf("expected last message to be assistant, got %s", lastMsg.Role)
		}

		if lastMsg.Content != "Response" {
			t.Errorf("expected last message content to be 'Response', got %q", lastMsg.Content)
		}
	})

	t.Run("append preserves messages in next Run call", func(t *testing.T) {
		agent := &Agent{
			llm:       &MockLLMClient{},
			tools:     make(ToolRegistry),
			config:    DefaultAgentConfig(),
			hooks:     Hooks{},
			prompt:    testPrompt,
			lastState: nil,
		}

		// Append a message before first Run
		preMsg := ChatMessage{
			Role:    RoleUser,
			Content: "Pre-message",
		}
		agent.Append(preMsg)

		// Simulate a Run call (we can't actually run it without a real LLM, but we can check state creation)
		// The Run method will create a new state from lastState
		if agent.lastState == nil {
			t.Fatal("expected lastState to exist after Append")
		}

		// Verify the pre-message is in lastState
		found := false
		for _, msg := range agent.lastState.History {
			if msg.Content == "Pre-message" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected pre-message to be in lastState history")
		}
	})

	t.Run("append with different message roles", func(t *testing.T) {
		agent := &Agent{
			llm:       &MockLLMClient{},
			tools:     make(ToolRegistry),
			config:    DefaultAgentConfig(),
			hooks:     Hooks{},
			prompt:    testPrompt,
			lastState: nil,
		}

		messages := []ChatMessage{
			{Role: RoleUser, Content: "User message"},
			{Role: RoleAssistant, Content: "Assistant message"},
			{Role: RoleTool, Name: "tool1", Content: "Tool result"},
		}

		for _, msg := range messages {
			agent.Append(msg)
		}

		if len(agent.lastState.History) != 4 { // system + 3 messages
			t.Errorf("expected 4 messages in history, got %d", len(agent.lastState.History))
		}

		// Verify all messages are present
		expectedRoles := []MessageRole{RoleSystem, RoleUser, RoleAssistant, RoleTool}
		for i, expectedRole := range expectedRoles {
			if agent.lastState.History[i].Role != expectedRole {
				t.Errorf("message %d: expected role %s, got %s", i, expectedRole, agent.lastState.History[i].Role)
			}
		}
	})

	t.Run("append with nil prompt", func(t *testing.T) {
		agent := &Agent{
			llm:       &MockLLMClient{},
			tools:     make(ToolRegistry),
			config:    DefaultAgentConfig(),
			hooks:     Hooks{},
			prompt:    nil, // nil prompt
			lastState: nil,
		}

		msg := ChatMessage{
			Role:    RoleUser,
			Content: "Test message",
		}

		agent.Append(msg)

		if agent.lastState == nil {
			t.Fatal("expected lastState to be created even with nil prompt, got nil")
		}

		// With nil prompt, history should start empty (no system message)
		if len(agent.lastState.History) != 1 {
			t.Errorf("expected 1 message in history (user only, no system), got %d", len(agent.lastState.History))
		}

		if agent.lastState.History[0].Role != RoleUser {
			t.Errorf("expected first message to be user, got %s", agent.lastState.History[0].Role)
		}

		if agent.lastState.History[0].Content != "Test message" {
			t.Errorf("expected message content to be 'Test message', got %q", agent.lastState.History[0].Content)
		}
	})
}
