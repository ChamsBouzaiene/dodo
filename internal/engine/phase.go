package engine

func DetectPhase(history []ChatMessage) Phase {
	// naive: look at last tool names embedded in tool messages
	for i := len(history) - 1; i >= 0; i-- {
		m := history[i]
		if m.Role == RoleTool {
			if m.Name == "search_replace" || m.Name == "write" {
				return PhaseEdit
			}
			if m.Name == "run_tests" || m.Name == "run_build" || m.Name == "run_lint" {
				return PhaseValidate
			}
			if m.Name == "grep" || m.Name == "codebase_search" || m.Name == "read_file" || m.Name == "read_span" || m.Name == "list_files" {
				return PhaseDiscoverAndPlan
			}
		}
	}
	return PhaseExplore
}

// Call st.Phase = DetectPhase(st.History) at the beginning of each step. to detect the phase.
