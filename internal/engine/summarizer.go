package engine

import (
	"context"
	"strings"
)

const summarizeSystem = `You compress prior chat history for a coding assistant. Preserve decisions, file paths, function names, errors, and TODOs. Omit pleasantries and redundant logs.`

func SummarizeOld(ctx context.Context, llm LLMClient, st *State, window []ChatMessage) (ChatMessage, error) {
	msgs := []ChatMessage{
		{Role: RoleSystem, Content: summarizeSystem},
		{Role: RoleUser, Content: "Summarize the following history in <= 200 tokens, preserve facts and decisions:\n\n" + RenderForSummary(window)},
	}
	resp, err := llm.Chat(ctx, st.Model, msgs, nil, ChatOptions{MaxOutputTokens: 256})
	if err != nil {
		return ChatMessage{}, err
	}
	return ChatMessage{Role: RoleSystem, Content: "<history_summary>\n" + resp.Assistant.Content + "\n</history_summary>"}, nil
}

func RenderForSummary(ms []ChatMessage) string {
	var b strings.Builder
	for _, m := range ms {
		b.WriteString("[" + string(m.Role) + "] ")
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}
