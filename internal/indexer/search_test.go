package indexer

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// MockEmbedder for testing
type MockEmbedder struct {
	vectors map[string][]float32
}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]byte, int, error) {
	vec, ok := m.vectors[text]
	if !ok {
		// Return zero vector if not found
		vec = make([]float32, 4)
	}
	return encodeVectorTest(vec), len(vec), nil
}

func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]byte, int, error) {
	var results [][]byte
	dim := 0
	for _, text := range texts {
		vec, ok := m.vectors[text]
		if !ok {
			vec = make([]float32, 4)
		}
		results = append(results, encodeVectorTest(vec))
		dim = len(vec)
	}
	return results, dim, nil
}

func (m *MockEmbedder) Dimension() int {
	return 4
}

func encodeVectorTest(vector []float32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, vector)
	return buf.Bytes()
}

func TestManager_Search(t *testing.T) {
	// Setup temp directory
	tmpDir, err := os.MkdirTemp("", "indexer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "index.db")

	// Setup mock embedder
	// We'll use 4D vectors for simplicity
	// Query: "search" -> [1, 0, 0, 0]
	// Doc1: "search result" -> [0.9, 0.1, 0, 0] (High similarity)
	// Doc2: "other text" -> [0, 1, 0, 0] (Low similarity)
	mockEmbedder := &MockEmbedder{
		vectors: map[string][]float32{
			"search":        {1.0, 0.0, 0.0, 0.0},
			"search result": {0.9, 0.1, 0.0, 0.0},
			"other text":    {0.0, 1.0, 0.0, 0.0},
		},
	}

	// Create Manager
	config := ManagerConfig{
		DBPath:   dbPath,
		RepoID:   "test-repo",
		RepoRoot: tmpDir,
		Embedder: mockEmbedder,
	}

	ctx := context.Background()
	mgr, err := NewManager(ctx, config)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr.Stop()

	// Insert test data directly into DB and BM25
	// We need to access the underlying components

	// 1. Insert into DB
	chunk1 := Chunk{
		ChunkID:   "chunk1",
		RepoID:    "test-repo",
		FilePath:  "file1.go",
		Lang:      "go",
		Text:      "search result",
		StartLine: 1,
		EndLine:   5,
	}
	chunk2 := Chunk{
		ChunkID:   "chunk2",
		RepoID:    "test-repo",
		FilePath:  "file2.txt",
		Lang:      "text",
		Text:      "other text",
		StartLine: 1,
		EndLine:   5,
	}

	// Insert chunks
	err = mgr.indexer.db.InsertChunk(ctx, &chunk1)
	if err != nil {
		t.Fatalf("Failed to insert chunk1: %v", err)
	}
	err = mgr.indexer.db.InsertChunk(ctx, &chunk2)
	if err != nil {
		t.Fatalf("Failed to insert chunk2: %v", err)
	}

	// Insert embeddings
	vec1 := encodeVectorTest(mockEmbedder.vectors["search result"])
	vec2 := encodeVectorTest(mockEmbedder.vectors["other text"])

	// We need to manually insert embeddings as InsertChunk doesn't do it
	// The DB interface doesn't expose InsertEmbedding directly, but Indexer does?
	// Wait, Indexer.db is *DB. Let's check DB methods.
	// DB has InsertEmbedding? I need to check internal/indexer/db.go.
	// Assuming it does or I can execute raw SQL.
	// Manager.GetDB() returns *DB.

	_, err = mgr.GetDB().db.ExecContext(ctx, `INSERT INTO embeddings (chunk_id, repo_id, vector, dim) VALUES (?, ?, ?, ?)`, chunk1.ChunkID, "test-repo", vec1, 4)
	if err != nil {
		t.Fatalf("Failed to insert embedding 1: %v", err)
	}
	_, err = mgr.GetDB().db.ExecContext(ctx, `INSERT INTO embeddings (chunk_id, repo_id, vector, dim) VALUES (?, ?, ?, ?)`, chunk2.ChunkID, "test-repo", vec2, 4)
	if err != nil {
		t.Fatalf("Failed to insert embedding 2: %v", err)
	}

	// 2. Insert into BM25
	// chunk1 has "search" keyword, chunk2 does not.
	err = mgr.bm25.IndexChunk(&chunk1, "")
	if err != nil {
		t.Fatalf("Failed to index chunk1 in BM25: %v", err)
	}
	err = mgr.bm25.IndexChunk(&chunk2, "")
	if err != nil {
		t.Fatalf("Failed to index chunk2 in BM25: %v", err)
	}

	// Test Search
	// Query "search" should match chunk1 via BM25 (keyword) and Embedding (high similarity).
	// chunk2 should have low score.

	results, err := mgr.Search(ctx, "search", nil, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected results, got none")
	}

	// Check top result
	if results[0].Path != "file1.go" {
		t.Errorf("Top result should be file1.go, got %s", results[0].Path)
	}

	// Check that we have a score
	if results[0].Score <= 0 {
		t.Errorf("Expected positive score, got %f", results[0].Score)
	}

	// Check reason (should be RRF)
	if results[0].Reason != "rrf(bm25+vec)" {
		t.Errorf("Expected reason rrf(bm25+vec), got %s", results[0].Reason)
	}
}
