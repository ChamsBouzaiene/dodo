package codebeacon

import (
	"context"
	"fmt"
	"strings"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// NewCodeBeaconTool creates the CodeBeacon analysis tool for brain agent.
// This tool invokes a separate analysis agent to investigate the codebase.
// This is defined in brain package to avoid circular import issues.
func NewCodeBeaconTool(beaconAgent *CodeBeaconAgent) engine.Tool {
	return engine.Tool{
		Name: "code_beacon",
		Description: `Invoke CodeBeacon scout to investigate codebase and answer specific questions.

CodeBeacon is a READ-ONLY scout that finds relevant code and provides a map.
It does NOT read every detail - you can follow up with read_span if needed.

INPUT:
- investigation_goal: Specific question you want answered
- scope: "focused" | "moderate" | "comprehensive" (optional, defaults to "moderate")
- focus_areas: Optional list of directories to focus on

SCOPE GUIDANCE:
- "focused": Narrow question about single feature (5-8K tokens)
  Example: "How does JWT validation work?"
  
- "moderate": Question about related components (10-15K tokens)
  Example: "How are CLI commands implemented and registered?"
  
- "comprehensive": Full architecture overview (20-30K tokens)
  Example: "Explain the complete architecture of this application"

Choose scope based on:
1. Question specificity (narrow = focused, broad = comprehensive)
2. Project size (check workspace context - large project = be more focused)
3. Token budget (how much context can you afford?)

EXAMPLE:
{
  "investigation_goal": "How are middleware functions implemented and registered?",
  "scope": "focused",
  "focus_areas": ["internal/middleware", "internal/server"]
}

OUTPUT: 
- Relevant files with descriptions
- Key patterns and structures  
- Line ranges for important code
- Recommendations
- List of files already read (don't re-read these)

After receiving report:
- Use it as a MAP to guide your work
- Use read_span for additional details if needed
- Trust the findings - don't re-read files to verify`,
		SchemaJSON: `{
			"type": "object",
			"properties": {
				"investigation_goal": {
					"type": "string",
					"description": "Clear description of what you want to understand about the codebase"
				},
				"scope": {
					"type": "string",
					"enum": ["focused", "moderate", "comprehensive"],
					"description": "Investigation scope: focused (5-8K tokens), moderate (10-15K tokens), comprehensive (20-30K tokens). Defaults to moderate."
				},
				"focus_areas": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Optional: directories or modules to focus investigation on"
				}
			},
			"required": ["investigation_goal"]
		}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			// Parse investigation_goal
			goal, ok := args["investigation_goal"].(string)
			if !ok || goal == "" {
				return "", fmt.Errorf("investigation_goal must be a non-empty string")
			}

			// Parse scope (optional, defaults to "moderate")
			scope := "moderate"
			if scopeRaw, ok := args["scope"].(string); ok && scopeRaw != "" {
				scope = scopeRaw
			}

			// Parse focus_areas (optional)
			var focusAreas []string
			if areasRaw, ok := args["focus_areas"].([]interface{}); ok {
				for _, area := range areasRaw {
					if areaStr, ok := area.(string); ok {
						focusAreas = append(focusAreas, areaStr)
					}
				}
			}

			// Call CodeBeacon agent (with caching)
			report, fromCache, err := beaconAgent.Investigate(ctx, goal, scope, focusAreas)
			if err != nil {
				return "", fmt.Errorf("CodeBeacon investigation failed: %w", err)
			}

			// Format report for brain agent with concise findings block
			return formatBeaconToolOutput(report, fromCache), nil
		},
		Retryable: true,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "analysis",
			Tags:     []string{"read-only", "analysis", "codebase-understanding"},
		},
	}
}

func formatBeaconToolOutput(report *BeaconReport, fromCache bool) string {
	var sb strings.Builder

	sb.WriteString("[CODEBEACON FINDINGS]\n")
	if fromCache {
		sb.WriteString("Source: cached report (recent)\n")
	} else {
		sb.WriteString("Source: fresh investigation\n")
	}
	sb.WriteString(fmt.Sprintf("Goal: %s\n", report.InvestigationGoal))

	if summary := strings.TrimSpace(report.Summary); summary != "" {
		firstLine := summary
		if idx := strings.Index(summary, "\n"); idx >= 0 {
			firstLine = summary[:idx]
		}
		sb.WriteString(fmt.Sprintf("Summary: %s\n", strings.TrimSpace(firstLine)))
	}

	if len(report.RelevantFiles) > 0 {
		sb.WriteString("Key files:\n")
		limit := 3
		if len(report.RelevantFiles) < limit {
			limit = len(report.RelevantFiles)
		}
		for i := 0; i < limit; i++ {
			file := report.RelevantFiles[i]
			sb.WriteString(fmt.Sprintf("- %s — %s\n", file.Path, file.Relevance))
		}
	}

	// Add list of files already analyzed by CodeBeacon
	if len(report.FilesRead) > 0 {
		sb.WriteString("\n⚠️  CodeBeacon already read these files (DO NOT re-read):\n")
		for _, path := range report.FilesRead {
			sb.WriteString(fmt.Sprintf("  - %s\n", path))
		}
		sb.WriteString("✅ Use the findings above instead of re-reading.\n")
	}

	sb.WriteString("[/CODEBEACON FINDINGS]\n\n")

	sb.WriteString(report.FormatForAgent())
	return sb.String()
}
