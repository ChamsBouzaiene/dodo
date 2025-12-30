package codebeacon

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
	"github.com/ChamsBouzaiene/dodo/internal/providers"

	"github.com/ChamsBouzaiene/dodo/internal/prompts"
	toolsengine "github.com/ChamsBouzaiene/dodo/internal/tools"
)

const defaultBeaconCacheTTL = 10 * time.Minute

// CodeBeaconAgent is a read-only analysis agent that investigates codebases.
// It explores the codebase using read-only tools and produces structured reports
// to help the brain agent understand architecture and make better decisions.
type CodeBeaconAgent struct {
	repoRoot     string
	retrieval    indexer.Retrieval
	workspaceCtx *indexer.WorkspaceContext

	mu       sync.Mutex
	cache    map[string]cachedBeaconReport
	cacheTTL time.Duration
}

type cachedBeaconReport struct {
	report *BeaconReport
	stored time.Time
}

// NewCodeBeaconAgent wires up a factory for short-lived CodeBeacon sessions.
func NewCodeBeaconAgent(ctx context.Context, repoRoot string, retrieval indexer.Retrieval, workspaceCtx *indexer.WorkspaceContext) (*CodeBeaconAgent, error) {
	return &CodeBeaconAgent{
		repoRoot:     repoRoot,
		retrieval:    retrieval,
		workspaceCtx: workspaceCtx,
		cache:        make(map[string]cachedBeaconReport),
		cacheTTL:     defaultBeaconCacheTTL,
	}, nil
}

// Investigate runs a focused analysis (or reuses a cached report).
// Returns the report, whether it was served from cache, and an error (if any).
func (b *CodeBeaconAgent) Investigate(ctx context.Context, goal string, scope string, focusAreas []string) (*BeaconReport, bool, error) {
	// Default to moderate if scope is empty
	if scope == "" {
		scope = "moderate"
	}

	key := b.cacheKey(goal, focusAreas)
	if report, ok := b.loadFromCache(key); ok {
		log.Printf("♻️  Reusing CodeBeacon report from cache (goal=%s, scope=%s)", goal, scope)
		return report, true, nil
	}

	report, parsed, err := b.runInvestigation(ctx, goal, scope, focusAreas, false)
	if err != nil {
		return nil, false, err
	}

	// If parsing failed, don't retry the entire investigation (too expensive).
	// Instead, use the fallback report which extracts useful info from the conversation.
	if !parsed {
		log.Printf("⚠️  CodeBeacon report was not valid JSON; using fallback report (goal=%s, scope=%s)", goal, scope)
		log.Printf("💡 Tip: The investigation completed but formatting was incorrect. Fallback report contains useful findings.")
	}

	b.saveToCache(key, report)
	return report, false, nil
}

func (b *CodeBeaconAgent) runInvestigation(ctx context.Context, goal string, scope string, focusAreas []string, strictJSON bool) (*BeaconReport, bool, error) {
	session, err := b.newSession(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create CodeBeacon session: %w", err)
	}

	prompt := buildInvestigationPrompt(goal, scope, focusAreas, strictJSON)

	log.Printf("🔍 CodeBeacon investigating (scope=%s): %s", scope, goal)

	if err := session.Run(ctx, prompt); err != nil {
		return nil, false, fmt.Errorf("investigation failed: %w", err)
	}

	state := session.LastState()
	if state == nil {
		return nil, false, fmt.Errorf("CodeBeacon session produced no state")
	}

	report, err := ExtractReportFromHistory(state.History)
	if err != nil {
		log.Printf("⚠️  Failed to parse CodeBeacon report: %v (falling back to raw findings)", err)
		report = CreateFallbackReport(goal, state.History)
		return report, false, nil
	}

	log.Printf("✅ CodeBeacon investigation complete (scope=%s, files=%d, types=%d, deps=%d)",
		scope, len(report.RelevantFiles), len(report.KeyTypes), len(report.Dependencies))
	return report, true, nil
}

func (b *CodeBeaconAgent) newSession(ctx context.Context) (*engine.Agent, error) {
	readOnlyTools := engine.ToolSet{
		Filesystem: true,
		Search:     true,
		Semantic:   b.retrieval != nil,
		Meta:       true,
	}

	// Create LLM client
	llm, modelName, err := providers.NewLLMClientFromEnv(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	builder := engine.NewAgentBuilder().
		WithLLM(llm).
		WithModel(modelName).
		WithStreaming(false).
		WithMaxSteps(30)

	builder, err = builder.WithPrompt("code_beacon", prompts.PromptV1)
	if err != nil {
		return nil, fmt.Errorf("failed to set prompt: %w", err)
	}

	reg, err := toolsengine.NewToolRegistry(b.repoRoot, b.retrieval, readOnlyTools)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry: %w", err)
	}

	builder = builder.WithToolRegistry(reg, b.repoRoot, b.retrieval, readOnlyTools)

	if b.workspaceCtx != nil {
		builder = builder.WithWorkspaceContext(b.workspaceCtx)
	}

	return builder.Build(ctx)
}

func (b *CodeBeaconAgent) cacheKey(goal string, focusAreas []string) string {
	normalized := make([]string, len(focusAreas))
	copy(normalized, focusAreas)
	sort.Strings(normalized)
	return strings.Join(append([]string{goal}, normalized...), "|")
}

func (b *CodeBeaconAgent) loadFromCache(key string) (*BeaconReport, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry, ok := b.cache[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.stored) > b.cacheTTL {
		delete(b.cache, key)
		return nil, false
	}
	return entry.report, true
}

func (b *CodeBeaconAgent) saveToCache(key string, report *BeaconReport) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cache[key] = cachedBeaconReport{
		report: report,
		stored: time.Now(),
	}
}

// buildInvestigationPrompt constructs the prompt for CodeBeacon based on goal and focus areas.
func buildInvestigationPrompt(goal string, scope string, focusAreas []string, strictJSON bool) string {
	var sb strings.Builder

	sb.WriteString("INVESTIGATION REQUEST:\n")
	sb.WriteString(fmt.Sprintf("Goal: %s\n", goal))
	sb.WriteString(fmt.Sprintf("Scope: %s\n\n", scope))

	if len(focusAreas) > 0 {
		sb.WriteString(fmt.Sprintf("Focus Areas: %s\n\n", strings.Join(focusAreas, ", ")))
	}

	// Add scope-specific guidance
	switch scope {
	case "focused":
		sb.WriteString(`FOCUSED INVESTIGATION (5-8K tokens, 3-5 steps):
- ONE codebase_search with narrow query
- Read 2-4 files with read_span (targeted sections only)
- Output concise report

`)
	case "comprehensive":
		sb.WriteString(`COMPREHENSIVE INVESTIGATION (20-30K tokens, 8-10 steps):
- ONE very broad codebase_search
- Read 6-10 file outlines (read_file on large files returns outline)
- Use read_span for small files or critical sections
- Output detailed architectural report

`)
	default: // moderate
		sb.WriteString(`MODERATE INVESTIGATION (10-15K tokens, 5-7 steps):
- ONE broad codebase_search
- Read 4-6 files with read_span (key sections)
- May do second search for specific details
- Output structured report

`)
	}

	sb.WriteString(`Please investigate the codebase and provide a structured analysis report.

Your investigation should:
1. Use codebase_search and grep to find relevant code
2. Read key files to understand implementation details
3. Identify patterns, interfaces, and dependencies
4. Note any risks or important considerations
5. Provide specific, actionable recommendations

At the end of your investigation, use the 'respond' tool with a JSON report in this structure:
{
  "investigation_goal": "...",
  "summary": "2-3 paragraph overview of your findings",
  "relevant_files": [
    {"path": "...", "relevance": "...", "key_symbols": ["..."]}
  ],
  "key_types": [
    {"name": "...", "kind": "interface|struct|function", "location": "...", "implementations": ["..."]}
  ],
  "dependencies": [
    {"from": "...", "to": "...", "type": "calls|implements|uses"}
  ],
  "patterns": [
    {"name": "...", "description": "...", "examples": ["..."]}
  ],
  "risks": ["..."],
  "recommendations": ["..."]
}`)

	if strictJSON {
		sb.WriteString("\nIMPORTANT: Your previous response was invalid JSON. Respond using ONLY the JSON object above with no commentary, prefixes, or suffixes.\n")
	} else {
		sb.WriteString("\nReminder: Respond with ONLY the JSON object above (no additional text).\n")
	}

	return sb.String()
}
