package prompts

import (
	"fmt"
	"strings"
)

// PromptBuilder helps compose prompts from fragments and variables.
type PromptBuilder struct {
	basePrompt *Prompt
	fragments  []string
	variables  map[string]string
}

// NewPromptBuilder creates a new prompt builder based on a registered prompt.
func NewPromptBuilder(registry *PromptRegistry, id string, version PromptVersion) (*PromptBuilder, error) {
	basePrompt, err := registry.Get(id, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get base prompt: %w", err)
	}

	return &PromptBuilder{
		basePrompt: basePrompt,
		fragments:  []string{basePrompt.Content},
		variables:  make(map[string]string),
	}, nil
}

// AddFragment appends a fragment to the prompt.
func (b *PromptBuilder) AddFragment(text string) *PromptBuilder {
	b.fragments = append(b.fragments, text)
	return b
}

// SetVariable sets a variable for template substitution.
func (b *PromptBuilder) SetVariable(key, value string) *PromptBuilder {
	b.variables[key] = value
	return b
}

// Build constructs the final prompt string.
func (b *PromptBuilder) Build() (string, error) {
	// Join fragments
	result := strings.Join(b.fragments, "\n\n")

	// Replace variables (simple {{key}} substitution)
	for key, value := range b.variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// BuildWithWorkspaceContext is a convenience method that adds workspace context.
func (b *PromptBuilder) BuildWithWorkspaceContext(workspaceContext string) (string, error) {
	if workspaceContext != "" {
		b.AddFragment(workspaceContext)
		b.AddFragment(`WORKSPACE SNAPSHOT: The <workspace_context> above is a complete snapshot of the workspace.
Check <project_layout> FIRST before calling 'list_files' - you already know all directories and files.
Only use 'list_files' if you need hidden files or special metadata.`)
	}
	return b.Build()
}
