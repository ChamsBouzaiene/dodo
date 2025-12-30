package prompts

// PromptVersion represents a version identifier for prompts.
type PromptVersion string

const (
	// PromptV1 is the first version of prompts.
	PromptV1 PromptVersion = "1.0.0"
	// PromptV2 is the second version (for future use).
	PromptV2 PromptVersion = "2.0.0"
)

// Prompt represents a versioned prompt with metadata.
type Prompt struct {
	ID          string        // Unique identifier (e.g., "coding", "interactive")
	Version     PromptVersion // Version of this prompt
	Content     string        // The actual prompt text
	Description string        // Human-readable description
	Tags        []string      // Tags for categorization (e.g., ["coding", "interactive", "strict"])
	Deprecated  bool          // True if this version is deprecated
}
