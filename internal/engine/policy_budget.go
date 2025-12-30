package engine

import "context"

// Deprecated: BuildMessagesWithBudget is deprecated. Use prepareMessages() instead, which
// integrates budget management with the new BudgetConfig system.
// This function is kept for backward compatibility but should not be used in new code.
func BuildMessagesWithBudget(ctx context.Context, st *State, llm LLMClient, estimate func([]ChatMessage) int, processors []Processor) ([]ChatMessage, error) {
	msgs := append([]ChatMessage(nil), st.History...)
	msgs, err := ApplyProcessors(ctx, st, msgs, processors...)
	if err != nil {
		return nil, err
	}
	
	// Use new budget system if configured
	if st.Budget.HardLimit > 0 {
		// Create empty hooks for backward compatibility
		hooks := Hooks{}
		return prepareMessages(ctx, st, llm, hooks, nil)
	}
	
	// Fallback to old behavior if no budget configured
	return msgs, nil
}
