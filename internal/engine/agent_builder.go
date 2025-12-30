package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/prompts"
)

// AgentBuilder helps construct an Agent with a fluent API.
// COPY builder pattern from agent/interactive.go
type AgentBuilder struct {
	config      AgentConfig
	llm         LLMClient
	tools       ToolRegistry
	hooks       Hooks
	prompt      *prompts.Prompt
	customRules string
}

// NewAgentBuilder creates a new agent builder with default configuration.
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		config: DefaultAgentConfig(),
	}
}

// WithModel sets the model name.
func (b *AgentBuilder) WithModel(model string) *AgentBuilder {
	b.config.Model = model
	return b
}

// WithLLM sets the LLM client.
func (b *AgentBuilder) WithLLM(llm LLMClient) *AgentBuilder {
	b.llm = llm
	return b
}

// WithMaxSteps sets the maximum number of steps.
func (b *AgentBuilder) WithMaxSteps(maxSteps int) *AgentBuilder {
	b.config.MaxSteps = maxSteps
	return b
}

// WithMaxOutputTokens sets the maximum output tokens for LLM responses.
// If not set, defaults to 8192. Set to 0 to use the default.
func (b *AgentBuilder) WithMaxOutputTokens(tokens int) *AgentBuilder {
	b.config.MaxOutputTokens = tokens
	return b
}

// WithBudget sets the budget configuration.
func (b *AgentBuilder) WithBudget(budget BudgetConfig) *AgentBuilder {
	b.config.Budget = budget
	return b
}

// WithRetryConfig sets the retry configuration.
func (b *AgentBuilder) WithRetryConfig(retryConfig *RetryConfig) *AgentBuilder {
	b.config.RetryConfig = retryConfig
	return b
}

// WithCompressionConfig sets the compression configuration.
func (b *AgentBuilder) WithCompressionConfig(compressionConfig *CompressionConfig) *AgentBuilder {
	b.config.CompressionConfig = compressionConfig
	return b
}

// WithToolRegistry allows callers to provide a fully constructed tool registry.
// This is useful when higher layers need to augment the default tool set.
func (b *AgentBuilder) WithToolRegistry(reg ToolRegistry, repoRoot string, retrieval indexer.Retrieval, set ToolSet) *AgentBuilder {
	b.tools = reg
	b.config.RepoRoot = repoRoot
	b.config.Retrieval = retrieval
	b.config.ToolSet = set
	return b
}

// WithPlanningEnforcement toggles whether edit tools remain blocked until a plan is created.
func (b *AgentBuilder) WithPlanningEnforcement(enforce bool) *AgentBuilder {
	b.config.EnforcePlanning = enforce
	return b
}

// WithPrompt sets the prompt ID and version.
func (b *AgentBuilder) WithPrompt(id string, version prompts.PromptVersion) (*AgentBuilder, error) {
	registry := prompts.DefaultRegistry()
	prompt, err := registry.Get(id, version)
	if err != nil {
		return nil, err
	}
	b.prompt = prompt
	b.config.PromptID = id
	b.config.PromptVersion = version
	return b, nil
}

// WithStreaming enables or disables streaming mode.
func (b *AgentBuilder) WithStreaming(streaming bool) *AgentBuilder {
	b.config.Streaming = streaming
	return b
}

// WithHooks sets custom hooks.
func (b *AgentBuilder) WithHooks(hooks Hooks) *AgentBuilder {
	b.hooks = hooks
	return b
}

// WithWorkspaceContext sets the workspace context.
func (b *AgentBuilder) WithWorkspaceContext(ctx *indexer.WorkspaceContext) *AgentBuilder {
	b.config.WorkspaceCtx = ctx
	return b
}

// WithCustomRules sets custom agent rules from project's .dodo/rules file.
func (b *AgentBuilder) WithCustomRules(rules string) *AgentBuilder {
	b.customRules = rules
	return b
}

// Build constructs the Agent instance.
func (b *AgentBuilder) Build(ctx context.Context) (*Agent, error) {
	// Create LLM client if not set
	if b.llm == nil {
		return nil, fmt.Errorf("LLM client not configured: use WithLLM")
	}

	// Create tools if not set
	if b.tools == nil {
		return nil, fmt.Errorf("tools not configured: use WithToolRegistry")
	}

	// Apply dynamic budget limits if not explicitly configured
	if b.config.Budget.HardLimit == 0 {
		b.config.Budget = GetModelLimits(b.config.Model)
		log.Printf("💰 Applied dynamic budget for model %s: %d tokens", b.config.Model, b.config.Budget.HardLimit)
	}

	// Get prompt if not set
	if b.prompt == nil {
		registry := prompts.DefaultRegistry()
		prompt, err := registry.GetLatest(b.config.PromptID)
		if err != nil {
			return nil, err
		}
		b.prompt = prompt
	}

	// Build prompt with workspace context if available
	if b.config.WorkspaceCtx != nil && b.prompt != nil {
		registry := prompts.DefaultRegistry()
		builder, err := prompts.NewPromptBuilder(registry, b.config.PromptID, b.config.PromptVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to create prompt builder: %w", err)
		}
		contextXML := b.config.WorkspaceCtx.FormatAsXML()
		log.Printf("📋 Injecting workspace context (%d bytes)", len(contextXML))
		finalPrompt, err := builder.BuildWithWorkspaceContext(contextXML)
		if err != nil {
			return nil, fmt.Errorf("failed to build prompt: %w", err)
		}
		// Create a temporary prompt with the final content
		b.prompt = &prompts.Prompt{
			ID:          b.prompt.ID,
			Version:     b.prompt.Version,
			Content:     finalPrompt,
			Description: b.prompt.Description,
			Tags:        b.prompt.Tags,
			Deprecated:  b.prompt.Deprecated,
		}
	}

	// Inject custom rules if provided
	if b.customRules != "" {
		rulesSection := fmt.Sprintf("\n\n[PROJECT CUSTOM RULES]\nThe following rules are defined in .dodo/rules for this project. Follow them strictly:\n\n%s\n\n[END PROJECT CUSTOM RULES]", b.customRules)
		b.prompt = &prompts.Prompt{
			ID:          b.prompt.ID,
			Version:     b.prompt.Version,
			Content:     b.prompt.Content + rulesSection,
			Description: b.prompt.Description,
			Tags:        b.prompt.Tags,
			Deprecated:  b.prompt.Deprecated,
		}
		log.Printf("📜 Injected custom rules (%d bytes) from .dodo/rules", len(b.customRules))
	}

	// Create default hooks if not set
	if b.hooks == nil {
		b.hooks = Hooks{
			LoggerHook{L: log.Default()},
			NewResponseHook(),
		}
	}

	// Log initial configuration (system prompt, tools, token counts)
	logInitialConfiguration(b.prompt, b.tools, b.config.Model)

	return &Agent{
		llm:    b.llm,
		tools:  b.tools,
		config: b.config,
		hooks:  b.hooks,
		prompt: b.prompt,
	}, nil
}

// logInitialConfiguration logs a simplified summary: prompt ID/version, token breakdown, and tool tags.
func logInitialConfiguration(prompt *prompts.Prompt, tools ToolRegistry, model string) {
	if prompt == nil {
		return
	}

	logger := log.Default()

	// Count tokens for system prompt
	tokenizer := GetTokenizerForModel(model)
	systemPromptTokens, _ := tokenizer.CountTokens(prompt.Content, model)

	// Log prompt ID and version
	logger.Printf("📋 PROMPT: %s@%s", prompt.ID, prompt.Version)

	// Count tokens for tools and collect tags
	schemas := tools.Schemas()
	totalToolTokens := 0
	toolTags := make(map[string]bool)       // Use map to deduplicate tags
	toolCategories := make(map[string]bool) // Use map to deduplicate categories

	for _, schema := range schemas {
		// Count tokens for tool schema
		toolDescTokens, _ := tokenizer.CountTokens(schema.Description, model)
		schemaTokens, _ := tokenizer.CountTokens(schema.JSONSchema, model)
		nameTokens, _ := tokenizer.CountTokens(schema.Name, model)
		toolTokens := nameTokens + toolDescTokens + schemaTokens + 10 // +10 for overhead
		totalToolTokens += toolTokens

		// Get tool metadata for tags/categories
		if tool, ok := tools[schema.Name]; ok {
			if tool.Metadata.Category != "" {
				toolCategories[tool.Metadata.Category] = true
			}
			for _, tag := range tool.Metadata.Tags {
				toolTags[tag] = true
			}
		}
	}

	// Log token breakdown
	logger.Printf("💰 TOKEN BREAKDOWN: system=~%d tokens, tools=~%d tokens, TOTAL=~%d tokens",
		systemPromptTokens, totalToolTokens, systemPromptTokens+totalToolTokens)

	// Log tool tags/categories
	if len(schemas) > 0 {
		categories := make([]string, 0, len(toolCategories))
		for cat := range toolCategories {
			categories = append(categories, cat)
		}
		tags := make([]string, 0, len(toolTags))
		for tag := range toolTags {
			tags = append(tags, tag)
		}
		logger.Printf("🔧 TOOLS: %d available [categories: %v, tags: %v]", len(schemas), categories, tags)
	} else {
		logger.Printf("🔧 TOOLS: none")
	}
}
