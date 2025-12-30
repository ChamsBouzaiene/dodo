package session

import (
	"context"
	"testing"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// MockLLM is a simple mock for the LLMClient interface
type MockLLM struct {
	Response string
}

func (m *MockLLM) Chat(ctx context.Context, model string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (engine.LLMResponse, error) {
	return engine.LLMResponse{
		Assistant: engine.ChatMessage{
			Role:    engine.RoleAssistant,
			Content: m.Response,
		},
	}, nil
}

func (m *MockLLM) Stream(ctx context.Context, model string, messages []engine.ChatMessage, toolSchemas []engine.ToolSchema, opts engine.ChatOptions) (<-chan engine.StreamEvent, <-chan error) {
	return nil, nil
}

func TestSummarizer_GenerateTitle(t *testing.T) {
	mock := &MockLLM{Response: "Refactoring Auth Logic"}
	summarizer := NewSummarizer(mock, "test-model")

	history := []engine.ChatMessage{
		{Role: engine.RoleUser, Content: "I need to fix the login bug"},
	}

	title, err := summarizer.GenerateTitle(context.Background(), history)
	if err != nil {
		t.Fatalf("GenerateTitle failed: %v", err)
	}

	if title != "Refactoring Auth Logic" {
		t.Errorf("Expected title 'Refactoring Auth Logic', got '%s'", title)
	}
}

func TestSummarizer_GenerateSummary(t *testing.T) {
	mock := &MockLLM{Response: "User fixed the login bug by updating the token validation."}
	summarizer := NewSummarizer(mock, "test-model")

	history := []engine.ChatMessage{
		{Role: engine.RoleUser, Content: "I fixed the login bug"},
	}

	summary, err := summarizer.GenerateSummary(context.Background(), history)
	if err != nil {
		t.Fatalf("GenerateSummary failed: %v", err)
	}

	if summary != "User fixed the login bug by updating the token validation." {
		t.Errorf("Expected summary match, got '%s'", summary)
	}
}
