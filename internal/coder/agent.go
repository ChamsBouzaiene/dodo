package coder

import (
	"context"
	"fmt"
	"log"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/project"
	"github.com/ChamsBouzaiene/dodo/internal/providers"

	"github.com/ChamsBouzaiene/dodo/internal/prompts"
	toolsengine "github.com/ChamsBouzaiene/dodo/internal/tools"
	"github.com/ChamsBouzaiene/dodo/internal/tools/reasoning"
)

// CoderAgent is the main coding agent that plans and executes tasks.
// It wraps the generic agent engine with specific configuration for coding tasks.
type CoderAgent struct {
	*engine.Agent
}

// Option configures the CoderAgent.
type Option func(*engine.ToolRegistry)

// WithTool adds a custom tool to the agent's registry.
func WithTool(name string, tool engine.Tool) Option {
	return func(registry *engine.ToolRegistry) {
		(*registry)[name] = tool
	}
}

// NewAgent creates a fully configured CoderAgent.
func NewAgent(ctx context.Context, repoRoot string, retrieval indexer.Retrieval, workspaceCtx *indexer.WorkspaceContext, streaming bool, muteResponse bool, extraHooks []engine.Hook, opts ...Option) (*CoderAgent, error) {
	standardToolSet := engine.ToolSet{
		Filesystem: true,
		Search:     true,
		Semantic:   retrieval != nil,
		Execution:  true,
		Editing:    true,
		Meta:       true,
	}

	baseRegistry, err := toolsengine.NewToolRegistry(repoRoot, retrieval, standardToolSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	// Always expose internal planning tools
	baseRegistry["plan"] = reasoning.NewPlanTool()
	baseRegistry["revise_plan"] = reasoning.NewRevisePlanTool()
	baseRegistry["project_plan"] = reasoning.NewProjectPlanTool(repoRoot)

	// Apply options (inject custom tools like code_beacon)
	for _, opt := range opts {
		opt(&baseRegistry)
	}

	// Create LLM client
	llm, modelName, err := providers.NewLLMClientFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	builder := engine.NewAgentBuilder().
		WithLLM(llm).
		WithModel(modelName).
		WithStreaming(streaming).
		WithMaxSteps(engine.DefaultInteractiveMaxSteps).
		WithPlanningEnforcement(true).
		WithToolRegistry(baseRegistry, repoRoot, retrieval, standardToolSet)

	builder, err = builder.WithPrompt("interactive", prompts.PromptV2)
	if err != nil {
		return nil, fmt.Errorf("failed to configure brain prompt: %w", err)
	}

	// Load custom rules from .dodo/rules if they exist
	if rules, err := project.LoadRules(repoRoot); err == nil && rules != "" {
		builder = builder.WithCustomRules(rules)
	}

	if workspaceCtx != nil {
		builder = builder.WithWorkspaceContext(workspaceCtx)
	}

	builder = builder.WithCompressionConfig(engine.DefaultInteractiveCompressionConfig())

	var defaultHooks engine.Hooks
	// In stdio mode (muteResponse=true), skip LoggerHook to keep stdout clean for NDJSON
	// Logs will still appear on stderr from other log.Printf calls
	if !muteResponse {
		defaultHooks = append(defaultHooks, engine.LoggerHook{L: log.Default()})
		defaultHooks = append(defaultHooks, engine.NewResponseHook())
	}
	if len(extraHooks) > 0 {
		defaultHooks = append(defaultHooks, extraHooks...)
	}
	if len(defaultHooks) > 0 {
		builder = builder.WithHooks(defaultHooks)
	}

	agent, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build coder agent: %w", err)
	}

	return &CoderAgent{Agent: agent}, nil
}
