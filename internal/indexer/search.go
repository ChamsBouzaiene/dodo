package indexer

import "context"

// Span represents a code span (line range) in a file with metadata.
type Span struct {
	Path    string  `json:"path"`    // Relative path from repo root
	Start   int     `json:"start"`   // Start line (1-indexed)
	End     int     `json:"end"`     // End line (1-indexed, inclusive)
	Lang    string  `json:"lang"`    // Language (go, python, etc.)
	Snippet string  `json:"snippet"` // Preview of the code
	Score   float64 `json:"score"`   // Relevance score (0-1)
	Reason  string  `json:"reason"`  // How it was found (embedding_only, rrf(bm25+vec), etc.)
}

// Retrieval provides semantic code search and span reading.
type Retrieval interface {
	// Search finds the top k most relevant code spans for a query.
	// globs can be used to filter by file patterns (e.g., []string{"*.go", "internal/*"}).
	// Returns spans sorted by relevance (highest score first).
	Search(ctx context.Context, query string, globs []string, k int) ([]Span, error)
	
	// ReadSpan reads the source code for a specific span.
	// Returns the text content from start line to end line (inclusive).
	ReadSpan(ctx context.Context, path string, start, end int) (string, error)
}

