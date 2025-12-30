package engine

import "strings"

// GetModelLimits returns the budget configuration for a specific model.
// It allows models with larger context windows to utilize them.
func GetModelLimits(model string) BudgetConfig {
	// Default safety limits (16k context)
	defaultConfig := DefaultBudgetConfig()

	// Normalize model name for matching
	modelLower := strings.ToLower(model)

	switch {
	// Kimi K2 (200k context)
	// We leave some buffer for safety (190k hard limit)
	case strings.Contains(modelLower, "kimi"):
		return BudgetConfig{
			SoftLimit:            150000,
			HardLimit:            190000,
			MaxCompressionPasses: 5,
			ReserveTokens:        4000, // Reserve more for larger output potential
		}

	// GPT-4o (128k context)
	case strings.Contains(modelLower, "gpt-4o"):
		return BudgetConfig{
			SoftLimit:            100000,
			HardLimit:            120000,
			MaxCompressionPasses: 5,
			ReserveTokens:        4000,
		}

	// Claude 3.5 Sonnet / Opus (200k context)
	case strings.Contains(modelLower, "claude-3") || strings.Contains(modelLower, "sonnet") || strings.Contains(modelLower, "opus"):
		return BudgetConfig{
			SoftLimit:            150000,
			HardLimit:            190000,
			MaxCompressionPasses: 5,
			ReserveTokens:        4000,
		}

	// DeepSeek (64k or 128k depending on version, assume 64k safe for now if unknown)
	case strings.Contains(modelLower, "deepseek"):
		return BudgetConfig{
			SoftLimit:            50000,
			HardLimit:            60000,
			MaxCompressionPasses: 5,
			ReserveTokens:        3000,
		}
	}

	return defaultConfig
}
