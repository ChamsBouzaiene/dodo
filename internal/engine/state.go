// Package engine provides agent orchestration functionality.
// See README.md for architecture overview.

package engine

// Phase represents the current phase of the agent's work.
type Phase string

const (
	PhaseExplore         Phase = "explore"
	PhaseDiscoverAndPlan Phase = "discover_and_plan"
	PhaseEdit            Phase = "edit"
	PhaseValidate        Phase = "validate"
)

type State struct {
	History  []ChatMessage // Conversation history
	Step     int           // Current step (increments only on success)
	Retries  int           // Retry attempts (tracked separately from steps)
	Done     bool          // True when LLM provides final answer (no tool calls)
	Phase    Phase         // Current phase (explore, discover_and_plan, edit, validate)
	Model    string        // LLM model name
	MaxSteps int           // Maximum steps before stopping
	Budget   BudgetConfig  // Token budget configuration (zero value = unlimited)
	Totals   Usage         // Accumulated token usage across all calls

	// Brain agent enhancements for SOTA behavior
	MiniPlan        *MiniPlan       // Internal plan
	ToolCallCount   int             // Total tool calls this run (for soft caps)
	FileReadCache   map[string]bool // Track which files have been read (avoid redundant reads)
	EditToolBlocked bool            // True if edits blocked (no plan yet)
	FailureCounts   map[string]int  // Track failures per tool/file (for soft caps)
}

func (s *State) Append(msg ChatMessage) { s.History = append(s.History, msg) }
