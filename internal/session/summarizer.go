package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// Summarizer handles LLM-based summarization for sessions.
type Summarizer struct {
	llm   engine.LLMClient
	model string
}

// NewSummarizer creates a new session summarizer.
func NewSummarizer(llm engine.LLMClient, model string) *Summarizer {
	return &Summarizer{
		llm:   llm,
		model: model,
	}
}

// GenerateTitle generates a short 3-5 word title for the session.
func (s *Summarizer) GenerateTitle(ctx context.Context, history []engine.ChatMessage) (string, error) {
	if len(history) == 0 {
		return "New Session", nil
	}

	systemPrompt := "You are a helpful assistant. Generate a short, concise title (3-5 words) for this session based on the user's intent and work done. Do not use quotes or punctuation."

	// We only need the first few messages to determine the intent for the title
	limit := 10
	if len(history) < limit {
		limit = len(history)
	}

	userPrompt := fmt.Sprintf("History:\n%s\n\nGenerate Title:", engine.RenderForSummary(history[:limit]))

	msgs := []engine.ChatMessage{
		{Role: engine.RoleSystem, Content: systemPrompt},
		{Role: engine.RoleUser, Content: userPrompt},
	}

	resp, err := s.llm.Chat(ctx, s.model, msgs, nil, engine.ChatOptions{
		MaxOutputTokens: 20,
		Temperature:     0.3,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	return strings.TrimSpace(resp.Assistant.Content), nil
}

// GenerateSummary generates a context summary for the next session.
func (s *Summarizer) GenerateSummary(ctx context.Context, history []engine.ChatMessage) (string, error) {
	if len(history) == 0 {
		return "", nil
	}

	systemPrompt := "You represent the memory of an AI coding assistant. Summarize the following session history to preserve context for a future session. Focus on: decisions made, files modified, unresolved errors, and next steps. Be concise."

	userPrompt := fmt.Sprintf("Summarize this session:\n\n%s", engine.RenderForSummary(history))

	msgs := []engine.ChatMessage{
		{Role: engine.RoleSystem, Content: systemPrompt},
		{Role: engine.RoleUser, Content: userPrompt},
	}

	resp, err := s.llm.Chat(ctx, s.model, msgs, nil, engine.ChatOptions{
		MaxOutputTokens: 500,
		Temperature:     0.1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	return strings.TrimSpace(resp.Assistant.Content), nil
}
