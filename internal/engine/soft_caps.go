package engine

import (
	"fmt"
	"strings"
)

// Soft cap constants
const (
	MaxToolCallsPerRun       = 40 // Maximum total tool calls before suggesting planner mode
	MaxBuildFailures         = 3  // Maximum build failures before stopping
	MaxSearchReplaceFailures = 3  // Maximum search_replace failures per file
)

// SoftCapError indicates that a soft limit was reached during execution.
// Unlike hard errors, soft caps are meant to gracefully stop execution
// with helpful guidance rather than failing abruptly.
type SoftCapError struct {
	Type    string // Type of cap reached: "tool_call_limit", "build_failure_limit", "edit_failure_limit"
	Message string // Helpful message explaining what happened and what to do
}

func (e *SoftCapError) Error() string {
	return fmt.Sprintf("soft cap reached (%s): %s", e.Type, e.Message)
}

// IsSoftCapError checks if an error is a SoftCapError.
func IsSoftCapError(err error) bool {
	_, ok := err.(*SoftCapError)
	return ok
}

// checkSoftCaps checks if any soft limits have been reached.
// Returns a SoftCapError if a limit is exceeded, nil otherwise.
func checkSoftCaps(st *State) error {
	// Initialize FailureCounts if needed
	if st.FailureCounts == nil {
		st.FailureCounts = make(map[string]int)
	}

	// Check tool call count
	if st.ToolCallCount >= MaxToolCallsPerRun {
		return &SoftCapError{
			Type:    "tool_call_limit",
			Message: fmt.Sprintf("Reached soft cap: %d tool calls. This task is complex and may require manual intervention or a different approach.", MaxToolCallsPerRun),
		}
	}

	// Check build failures
	if st.FailureCounts["run_build"] >= MaxBuildFailures {
		return &SoftCapError{
			Type:    "build_failure_limit",
			Message: fmt.Sprintf("Build failed %d times. Try a different approach, ask the user for guidance, or revise your plan if you have one.", MaxBuildFailures),
		}
	}

	// Check search_replace failures per file
	for key, count := range st.FailureCounts {
		if strings.HasPrefix(key, "search_replace:") && count >= MaxSearchReplaceFailures {
			file := strings.TrimPrefix(key, "search_replace:")
			return &SoftCapError{
				Type:    "edit_failure_limit",
				Message: fmt.Sprintf("Failed to edit %s multiple times. Consider using write_file to rewrite the entire file, or revise your plan with a different approach.", file),
			}
		}
	}

	return nil
}
