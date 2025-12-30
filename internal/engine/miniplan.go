package engine

import (
	"fmt"
	"strings"
	"time"
)

// StepStatus represents the status of a mini-plan step.
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusCompleted StepStatus = "completed"
	StepStatusSkipped   StepStatus = "skipped"
)

// MiniPlanStep represents a single step in the internal plan.
type MiniPlanStep struct {
	ID          string     `json:"id"`           // "step-1", "step-2", etc.
	Description string     `json:"description"`  // What needs to be done
	TargetFiles []string   `json:"target_files"` // Files this step will touch
	Status      StepStatus `json:"status"`       // pending, completed, skipped
}

// PlanRevision represents a modification to the plan after initial creation.
type PlanRevision struct {
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason"`  // Why the plan was revised
	Changes   string    `json:"changes"` // What changed
}

// MiniPlan is the internal execution plan for the brain agent.
// It is lightweight and exists only in memory during a single run.
type MiniPlan struct {
	TaskSummary string         `json:"task_summary"` // 1-2 sentence summary
	Steps       []MiniPlanStep `json:"steps"`        // 3-6 steps
	TargetAreas []string       `json:"target_areas"` // Modules/directories involved
	Risks       []string       `json:"risks"`        // Potential issues
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Revisions   []PlanRevision `json:"revisions"` // History of changes
}

// FormatForPrompt returns a compact representation for context injection.
// This is inserted into the agent's context so it can track progress.
func (p *MiniPlan) FormatForPrompt() string {
	var sb strings.Builder

	sb.WriteString("[INTERNAL_PLAN]\n")
	sb.WriteString(fmt.Sprintf("Task: %s\n", p.TaskSummary))

	// Format steps with status indicators
	sb.WriteString("Steps: ")
	for i, step := range p.Steps {
		if i > 0 {
			sb.WriteString(" | ")
		}

		var statusIcon string
		switch step.Status {
		case StepStatusCompleted:
			statusIcon = "✓"
		case StepStatusPending:
			statusIcon = " "
		case StepStatusSkipped:
			statusIcon = "⊘"
		}

		// Truncate description for compact display
		desc := step.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}

		sb.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, statusIcon, desc))
	}
	sb.WriteString("\n")

	// Add target areas if present
	if len(p.TargetAreas) > 0 {
		sb.WriteString(fmt.Sprintf("Target Areas: %s\n", strings.Join(p.TargetAreas, ", ")))
	}

	// Add risks if present
	if len(p.Risks) > 0 {
		sb.WriteString("Risks: ")
		for i, risk := range p.Risks {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(risk)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("[/INTERNAL_PLAN]")

	return sb.String()
}

// MarkStepCompleted marks a step as completed.
func (p *MiniPlan) MarkStepCompleted(stepID string) error {
	for i := range p.Steps {
		if p.Steps[i].ID == stepID {
			p.Steps[i].Status = StepStatusCompleted
			p.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", stepID)
}

// MarkStepSkipped marks a step as skipped.
func (p *MiniPlan) MarkStepSkipped(stepID string) error {
	for i := range p.Steps {
		if p.Steps[i].ID == stepID {
			p.Steps[i].Status = StepStatusSkipped
			p.UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", stepID)
}

// GetNextPendingStep returns the next pending step, or nil if all steps are done.
func (p *MiniPlan) GetNextPendingStep() *MiniPlanStep {
	for i := range p.Steps {
		if p.Steps[i].Status == StepStatusPending {
			return &p.Steps[i]
		}
	}
	return nil
}

// AllStepsCompleted returns true if all steps are completed or skipped.
func (p *MiniPlan) AllStepsCompleted() bool {
	for _, step := range p.Steps {
		if step.Status == StepStatusPending {
			return false
		}
	}
	return true
}
