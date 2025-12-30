package reasoning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// NewPlanTool creates the internal planning tool.
// This tool allows the agent to create a lightweight execution plan before making edits.
func NewPlanTool() engine.Tool {
	return engine.Tool{
		Name: "plan",
		Description: `Create an internal execution plan for non-trivial code changes.

WHEN TO USE:
- Multi-file changes (2+ files)
- Refactoring or architectural changes
- New features spanning multiple modules
- Bug fixes requiring investigation across files

OUTPUT:
- task_summary: 1-2 sentence description of what you'll accomplish
- steps: Array of 3-6 concrete steps (include file names, function names)
- target_areas: List of modules/directories involved
- risks: Array of potential issues to watch for

EXAMPLE:
{
  "task_summary": "Add JWT authentication middleware to REST API",
  "steps": [
    {"id": "step-1", "description": "Read existing middleware pattern from logger.go", "target_files": ["internal/middleware/logger.go"]},
    {"id": "step-2", "description": "Create auth.go with JWT validation using auth.ValidateToken", "target_files": ["internal/middleware/auth.go"]},
    {"id": "step-3", "description": "Register middleware in main.go router", "target_files": ["cmd/server/main.go"]},
    {"id": "step-4", "description": "Update handlers to use context user info", "target_files": ["internal/handlers/user.go", "internal/handlers/product.go"]},
    {"id": "step-5", "description": "Add tests for auth success and failure cases", "target_files": ["internal/middleware/auth_test.go"]}
  ],
  "target_areas": ["internal/middleware", "internal/handlers", "cmd/server"],
  "risks": ["Breaking change: endpoints will return 401 without tokens", "Need to update existing tests"]
}`,
		SchemaJSON: `{
			"type": "object",
			"properties": {
				"task_summary": {"type": "string", "description": "1-2 sentence summary of what will be accomplished"},
				"steps": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"id": {"type": "string", "description": "step-1, step-2, etc."},
							"description": {"type": "string", "description": "What this step does (be specific, mention files/functions)"},
							"target_files": {"type": "array", "items": {"type": "string"}, "description": "Files this step will modify"}
						},
						"required": ["id", "description"]
					},
					"minItems": 3,
					"maxItems": 6
				},
				"target_areas": {"type": "array", "items": {"type": "string"}, "description": "Modules/directories involved"},
				"risks": {"type": "array", "items": {"type": "string"}, "description": "Potential issues to watch for"}
			},
			"required": ["task_summary", "steps", "target_areas"]
		}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			// Extract State from context if available (will be added in step.go)
			st, ok := ctx.Value("engine_state").(*engine.State)
			if !ok || st == nil {
				return "", fmt.Errorf("internal error: engine state not available in context")
			}

			// Parse task_summary
			taskSummary, ok := args["task_summary"].(string)
			if !ok {
				return "", fmt.Errorf("task_summary must be a string")
			}

			// Parse steps
			stepsRaw, ok := args["steps"].([]interface{})
			if !ok {
				return "", fmt.Errorf("steps must be an array")
			}

			if len(stepsRaw) < 3 || len(stepsRaw) > 6 {
				return "", fmt.Errorf("steps must contain 3-6 items, got %d", len(stepsRaw))
			}

			// Note: We use engine.MiniPlan struct directly
			steps := make([]map[string]interface{}, len(stepsRaw))
			for i, stepRaw := range stepsRaw {
				stepMap, ok := stepRaw.(map[string]interface{})
				if !ok {
					return "", fmt.Errorf("step %d is not a valid object", i+1)
				}

				id, _ := stepMap["id"].(string)
				description, _ := stepMap["description"].(string)

				if id == "" {
					return "", fmt.Errorf("step %d missing id", i+1)
				}
				if description == "" {
					return "", fmt.Errorf("step %d missing description", i+1)
				}

				// Parse target_files (optional)
				var targetFiles []string
				if filesRaw, ok := stepMap["target_files"].([]interface{}); ok {
					for _, fileRaw := range filesRaw {
						if file, ok := fileRaw.(string); ok {
							targetFiles = append(targetFiles, file)
						}
					}
				}

				steps[i] = map[string]interface{}{
					"id":           id,
					"description":  description,
					"target_files": targetFiles,
					"status":       "pending",
				}
			}

			// Parse target_areas
			targetAreasRaw, ok := args["target_areas"].([]interface{})
			if !ok {
				return "", fmt.Errorf("target_areas must be an array")
			}
			targetAreas := make([]string, 0, len(targetAreasRaw))
			for _, area := range targetAreasRaw {
				if areaStr, ok := area.(string); ok {
					targetAreas = append(targetAreas, areaStr)
				}
			}

			// Parse risks (optional)
			var risks []string
			if risksRaw, ok := args["risks"].([]interface{}); ok {
				for _, risk := range risksRaw {
					if riskStr, ok := risk.(string); ok {
						risks = append(risks, riskStr)
					}
				}
			}

			// Create MiniPlan struct
			now := time.Now()

			// Convert steps to MiniPlanStep structs
			miniPlanSteps := make([]engine.MiniPlanStep, len(steps))
			for i, stepMap := range steps {
				miniPlanSteps[i] = engine.MiniPlanStep{
					ID:          stepMap["id"].(string),
					Description: stepMap["description"].(string),
					TargetFiles: stepMap["target_files"].([]string),
					Status:      engine.StepStatusPending,
				}
			}

			miniPlan := &engine.MiniPlan{
				TaskSummary: taskSummary,
				Steps:       miniPlanSteps,
				TargetAreas: targetAreas,
				Risks:       risks,
				CreatedAt:   now,
				UpdatedAt:   now,
				Revisions:   []engine.PlanRevision{},
			}

			// Store in State
			st.MiniPlan = miniPlan

			// Unblock editing tools
			st.EditToolBlocked = false

			// Format confirmation message
			return fmt.Sprintf(`✅ Internal execution plan created successfully!

Plan Summary:
%s

%d steps defined:
%s

You can now proceed with edits following this plan. Use 'revise_plan' if you need to adjust the plan.`,
				taskSummary,
				len(steps),
				formatStepsSummary(miniPlanSteps),
			), nil
		},
		Retryable: false,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "planning",
			Tags:     []string{"planning", "internal-planning"},
		},
	}
}

// NewRevisePlanTool creates the plan revision tool.
// This tool allows the agent to update the plan when reality conflicts with expectations.
func NewRevisePlanTool() engine.Tool {
	return engine.Tool{
		Name: "revise_plan",
		Description: `Revise the internal execution plan when reality conflicts with it.

WHEN TO USE:
- Discovered the codebase structure is different than expected
- A step failed and needs different approach
- Found additional steps are needed
- Want to skip/reorder steps

Provide the complete revised plan (not a diff).`,
		SchemaJSON: `{
			"type": "object",
			"properties": {
				"reason": {"type": "string", "description": "Why you're revising the plan"},
				"revised_steps": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"id": {"type": "string"},
							"description": {"type": "string"},
							"target_files": {"type": "array", "items": {"type": "string"}}
						},
						"required": ["id", "description"]
					}
				},
				"updated_risks": {"type": "array", "items": {"type": "string"}}
			},
			"required": ["reason", "revised_steps"]
		}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			// Extract State from context
			st, ok := ctx.Value("engine_state").(*engine.State)
			if !ok || st == nil {
				return "", fmt.Errorf("internal error: engine state not available in context")
			}

			// Check if plan exists
			if st.MiniPlan == nil {
				return "", fmt.Errorf("no plan to revise. Call 'plan' tool first to create a plan")
			}

			// Get plan as struct
			miniPlan := st.MiniPlan

			// Parse reason
			reason, ok := args["reason"].(string)
			if !ok || reason == "" {
				return "", fmt.Errorf("reason must be a non-empty string")
			}

			// Parse revised_steps
			stepsRaw, ok := args["revised_steps"].([]interface{})
			if !ok {
				return "", fmt.Errorf("revised_steps must be an array")
			}

			revisedSteps := make([]engine.MiniPlanStep, len(stepsRaw))
			for i, stepRaw := range stepsRaw {
				stepMap, ok := stepRaw.(map[string]interface{})
				if !ok {
					return "", fmt.Errorf("step %d is not a valid object", i+1)
				}

				id, _ := stepMap["id"].(string)
				description, _ := stepMap["description"].(string)

				if id == "" {
					return "", fmt.Errorf("step %d missing id", i+1)
				}
				if description == "" {
					return "", fmt.Errorf("step %d missing description", i+1)
				}

				// Parse target_files (optional)
				var targetFiles []string
				if filesRaw, ok := stepMap["target_files"].([]interface{}); ok {
					for _, fileRaw := range filesRaw {
						if file, ok := fileRaw.(string); ok {
							targetFiles = append(targetFiles, file)
						}
					}
				}

				// Preserve status from old plan if step ID matches
				status := engine.StepStatusPending
				for _, oldStep := range miniPlan.Steps {
					if oldStep.ID == id {
						status = oldStep.Status
						break
					}
				}

				revisedSteps[i] = engine.MiniPlanStep{
					ID:          id,
					Description: description,
					TargetFiles: targetFiles,
					Status:      status,
				}
			}

			// Parse updated_risks (optional)
			var updatedRisks []string
			if risksRaw, ok := args["updated_risks"].([]interface{}); ok {
				for _, risk := range risksRaw {
					if riskStr, ok := risk.(string); ok {
						updatedRisks = append(updatedRisks, riskStr)
					}
				}
			} else {
				// Keep existing risks if not provided
				updatedRisks = miniPlan.Risks
			}

			// Get old steps count
			oldStepsCount := len(miniPlan.Steps)

			// Record revision
			revision := engine.PlanRevision{
				Timestamp: time.Now(),
				Reason:    reason,
				Changes:   fmt.Sprintf("Revised from %d steps to %d steps", oldStepsCount, len(revisedSteps)),
			}

			// Update plan
			miniPlan.Steps = revisedSteps
			miniPlan.Risks = updatedRisks
			miniPlan.UpdatedAt = time.Now()
			miniPlan.Revisions = append(miniPlan.Revisions, revision)

			return fmt.Sprintf(`✅ Plan revised successfully!

Revision Reason: %s

Updated Plan:
- Steps: %d
%s

Risks: %s

You can now continue with the revised plan.`,
				reason,
				len(revisedSteps),
				formatStepsSummary(revisedSteps),
				formatRisks(updatedRisks),
			), nil
		},
		Retryable: false,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "planning",
			Tags:     []string{"brain", "internal-planning"},
		},
	}
}

// Helper functions

func formatStepsSummary(steps []engine.MiniPlanStep) string {
	var result string
	for i, step := range steps {
		statusIcon := " "
		if step.Status == engine.StepStatusCompleted {
			statusIcon = "✓"
		} else if step.Status == engine.StepStatusSkipped {
			statusIcon = "⊘"
		}
		result += fmt.Sprintf("  %d. [%s] %s\n", i+1, statusIcon, step.Description)
	}
	return result
}

func formatRisks(risks []string) string {
	if len(risks) == 0 {
		return "None specified"
	}
	var result string
	for i, risk := range risks {
		if i > 0 {
			result += "; "
		}
		result += risk
	}
	return result
}

// NewProjectPlanTool creates the project plan tool.
func NewProjectPlanTool(repoRoot string) engine.Tool {
	return engine.Tool{
		Name: "project_plan",
		Description: `Read or update the high-level project plan (.dodo/plan.md).

WHEN TO USE:
- At the start of a session to understand the big picture
- When a major milestone is completed
- When the overall strategy changes

MODES:
- "read": Returns the current plan content
- "update": Overwrites the plan with new content (markdown supported)
- "append": Appends a new item to the plan

This is different from the 'plan' tool (MiniPlan), which is for the *current* session's immediate steps.`,
		SchemaJSON: `{
			"type": "object",
			"properties": {
				"mode": {
					"type": "string",
					"enum": ["read", "update", "append"],
					"description": "Operation mode"
				},
				"content": {
					"type": "string",
					"description": "Content for update/append modes"
				}
			},
			"required": ["mode"]
		}`,
		Fn: func(ctx context.Context, args map[string]any) (string, error) {
			mode, _ := args["mode"].(string)
			content, _ := args["content"].(string)

			planPath := filepath.Join(repoRoot, ".dodo", "plan.md")

			// Ensure .dodo directory exists
			if err := os.MkdirAll(filepath.Dir(planPath), 0755); err != nil {
				return "", fmt.Errorf("failed to create .dodo directory: %w", err)
			}

			switch mode {
			case "read":
				data, err := os.ReadFile(planPath)
				if os.IsNotExist(err) {
					return "No project plan found. Create one using mode='update'.", nil
				}
				if err != nil {
					return "", fmt.Errorf("failed to read plan: %w", err)
				}
				return string(data), nil

			case "update":
				if content == "" {
					return "", fmt.Errorf("content is required for update mode")
				}

				// Add timestamp header
				timestamp := time.Now().Format(time.RFC1123)
				header := fmt.Sprintf("# Project Plan\n> Last Updated: %s\n\n", timestamp)

				// If content already has the header, strip it to avoid duplication (simple heuristic)
				if strings.HasPrefix(content, "# Project Plan") {
					lines := strings.Split(content, "\n")
					if len(lines) > 2 && strings.Contains(lines[1], "Last Updated:") {
						content = strings.Join(lines[3:], "\n")
					}
				}

				fullContent := header + content

				if err := os.WriteFile(planPath, []byte(fullContent), 0644); err != nil {
					return "", fmt.Errorf("failed to write plan: %w", err)
				}
				return fmt.Sprintf("✅ Project plan updated successfully (Timestamp: %s).", timestamp), nil

			case "append":
				if content == "" {
					return "", fmt.Errorf("content is required for append mode")
				}

				f, err := os.OpenFile(planPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
				if err != nil {
					return "", fmt.Errorf("failed to open plan: %w", err)
				}
				defer f.Close()

				if _, err := f.WriteString("\n" + content + "\n"); err != nil {
					return "", fmt.Errorf("failed to append to plan: %w", err)
				}
				return "✅ Content appended to project plan.", nil

			default:
				return "", fmt.Errorf("invalid mode: %s", mode)
			}
		},
		Retryable: false,
		Metadata: engine.ToolMetadata{
			Version:  "1.0.0",
			Category: "planning",
			Tags:     []string{"planning", "persistent"},
		},
	}
}
