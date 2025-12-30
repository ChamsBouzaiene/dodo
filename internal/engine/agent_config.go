package engine

import (
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/prompts"
)

// AgentConfig holds configuration for an agent instance.
// COPY patterns from agent/interactive.go
type AgentConfig struct {
	Model             string
	MaxSteps          int
	Budget            BudgetConfig
	RetryConfig       *RetryConfig
	CompressionConfig *CompressionConfig
	ToolSet           ToolSet
	PromptID          string
	PromptVersion     prompts.PromptVersion
	Streaming         bool
	RepoRoot          string
	Retrieval         indexer.Retrieval
	WorkspaceCtx      *indexer.WorkspaceContext
	EnforcePlanning   bool // When true, edit tools remain blocked until plan tool runs
	MaxOutputTokens   int  // Maximum tokens for LLM output (0 = use default)
}

// DefaultAgentConfig returns a default agent configuration.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Model:    "gpt-4o-mini",
		MaxSteps: 30,
		// Budget is now resolved dynamically in Builder based on model if not set
		ToolSet:         ToolSet{Filesystem: true, Search: true},
		PromptID:        "coding",
		Streaming:       false,
		EnforcePlanning: false,
		MaxOutputTokens: 8192, // Default: 8192 tokens to support large tool calls (file writes, etc.)
	}
}

// DefaultInteractiveMaxSteps is the default step limit for interactive sessions.
const DefaultInteractiveMaxSteps = 60

// DefaultInteractiveCompressionConfig returns the standard compression settings for interactive mode.
func DefaultInteractiveCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Enabled:            true,
		KeepRecentCount:    18,
		SummarizeThreshold: 30,
		TruncateToolsAt:    3000,
		Strategy:           "balanced",
	}
}
