package indexer

import (
	"context"
)

// Chunker splits file content into searchable chunks.
// Different languages may have different chunking strategies
// (e.g., function-level for Go, paragraph for Markdown).
type Chunker interface {
	// Chunk splits a file into semantic chunks.
	// Returns a list of chunks with their metadata.
	Chunk(ctx context.Context, file FileInfo, content []byte) ([]Chunk, []Symbol, error)
}

// Embedder generates vector embeddings for text chunks.
// This abstracts the embedding model (OpenAI, local model, etc.).
type Embedder interface {
	// Embed generates a vector embedding for a text chunk.
	// Returns the embedding vector as a byte slice.
	Embed(ctx context.Context, text string) ([]byte, int, error) // vector, dimension, error
	
	// EmbedBatch generates embeddings for multiple chunks efficiently.
	// Returns embeddings in the same order as input texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]byte, int, error)
	
	// Dimension returns the dimension of the embedding vectors.
	Dimension() int
}

// ProcessingResult contains the result of processing a file.
type ProcessingResult struct {
	FileID     int64
	Symbols    []Symbol
	Chunks     []Chunk
	Embeddings []Embedding
	Error      error
}

