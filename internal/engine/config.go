package engine

import "time"

// CompressionConfig defines how message history should be compressed.
type CompressionConfig struct {
	Enabled            bool
	SummarizeThreshold int    // Keep last N before summarizing (default: 12)
	KeepRecentCount    int    // Always keep last N messages (default: 8)
	TruncateToolsAt    int    // Max chars for tool output (default: 4000)
	Strategy           string // "balanced", "aggressive", "conservative"
}

// DefaultCompressionConfig returns sensible default compression configuration.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Enabled:            true,
		SummarizeThreshold: 30, // Summarize after 30 messages (more headroom before compression)
		KeepRecentCount:    40,  // Keep last 40 messages (was 24) - preserve more context to prevent file re-reads
		TruncateToolsAt:    4000,
		Strategy:           "balanced",
	}
}

// EngineConfig holds all engine configuration options.
type EngineConfig struct {
	RetryConfig       *RetryConfig
	CompressionConfig *CompressionConfig
	// Add other config options here as needed
}

// DefaultEngineConfig returns a default engine configuration.
func DefaultEngineConfig() EngineConfig {
	retryConfig := DefaultRetryConfig()
	compressionConfig := DefaultCompressionConfig()
	return EngineConfig{
		RetryConfig:       &retryConfig,
		CompressionConfig: &compressionConfig,
	}
}

// DefaultRetryConfig returns sensible default retry policies.
// This function is kept here for backward compatibility and centralized config management.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		LLMPolicy: RetryPolicy{
			MaxRetries:   3,
			InitialDelay: 1 * time.Second,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
		},
		ToolPolicy: RetryPolicy{
			MaxRetries:   2,
			InitialDelay: 500 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
		},
	}
}
