package engine

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int // Approximate expectation
	}{
		{
			name: "empty",
			text: "",
			want: 0,
		},
		{
			name: "short word",
			text: "hello",
			want: 1, // 5 chars / 4 = 1
		},
		{
			name: "sentence",
			text: "hello world this is a test",
			want: 6, // 26 chars / 4 = 6 + whitespace/6 ~ 0 = 6
		},
		{
			name: "code snippet",
			text: "func main() { fmt.Println(\"hello\") }",
			want: 9, // 36 chars / 4 = 9 + whitespace/6 ~ 0 = 9
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			// Since it's an estimation, allow a small margin of error if needed,
			// but for now we test exact values based on the formula implementation.
			// Formula: (len(runes) / 4) + (whitespace / 6)
			// If estimated < 1 and len > 0, return 1.

			if got != tt.want {
				t.Errorf("EstimateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCountTokensForMessages(t *testing.T) {
	tokenizer := DefaultTokenizer{}
	model := "test-model"

	tests := []struct {
		name     string
		messages []ChatMessage
		minWant  int // Minimum expected tokens (to account for overhead)
	}{
		{
			name: "single message",
			messages: []ChatMessage{
				{Role: RoleUser, Content: "hello"},
			},
			// Role(user=4/4=1) + Content(hello=5/4=1) + Overhead(4) = 6
			minWant: 6,
		},
		{
			name: "with tool calls",
			messages: []ChatMessage{
				{
					Role:    RoleAssistant,
					Content: "calling tool",
					ToolCalls: []ToolCall{
						{Name: "test_tool", Args: map[string]any{"key": "val"}},
					},
				},
			},
			// Role(assistant=9/4=2) + Content(12/4=3) + ToolName(9/4=2) + Args(~15/4=3) + Overhead(4) = ~14
			minWant: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CountTokensForMessages(tokenizer, tt.messages, model)
			if err != nil {
				t.Errorf("CountTokensForMessages() error = %v", err)
				return
			}
			if got < tt.minWant {
				t.Errorf("CountTokensForMessages() = %v, want >= %v", got, tt.minWant)
			}
		})
	}
}

func TestGetTokenizerForModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{"openai gpt-4", "gpt-4"},
		{"openai o1", "o1-preview"},
		{"anthropic", "claude-3"},
		{"other", "llama-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTokenizerForModel(tt.model)
			if got == nil {
				t.Error("GetTokenizerForModel() returned nil")
			}
			// Currently all return DefaultTokenizer, verify it works
			count, err := got.CountTokens("test", tt.model)
			if err != nil {
				t.Errorf("Tokenizer.CountTokens error = %v", err)
			}
			if count <= 0 {
				t.Errorf("Tokenizer.CountTokens returned invalid count: %d", count)
			}
		})
	}
}
