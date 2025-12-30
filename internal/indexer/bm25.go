package indexer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
)

// BM25Result represents a BM25 search result.
type BM25Result struct {
	ChunkID  string
	Score    float64
	FilePath string
	Lang     string
}

// BM25Index provides BM25 keyword search over code chunks.
type BM25Index struct {
	index bleve.Index
	path  string
}

// NewBM25Index creates or opens a BM25 index.
// If the index is corrupted, it will automatically delete and recreate it.
func NewBM25Index(dbPath string) (*BM25Index, error) {
	indexPath := dbPath + ".bleve"
	
	// Try to open existing index
	index, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// Create new index
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create BM25 index: %w", err)
		}
		log.Println("📚 BM25 index created")
	} else if err != nil {
		// Index exists but is corrupted - delete and recreate
		log.Printf("⚠️  BM25 index appears corrupted (error: %v), recreating...", err)
		
		// Close the index if it was partially opened
		if index != nil {
			index.Close()
		}
		
		// Delete the corrupted index directory
		if err := os.RemoveAll(indexPath); err != nil {
			log.Printf("⚠️  Failed to remove corrupted index directory: %v", err)
			// Try to remove just the store directory
			storePath := filepath.Join(indexPath, "store")
			if err := os.RemoveAll(storePath); err != nil {
				log.Printf("⚠️  Failed to remove store directory: %v", err)
			}
		}
		
		// Create new index
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to recreate BM25 index: %w", err)
		}
		log.Println("✅ BM25 index recreated (corrupted index was deleted)")
	}
	
	return &BM25Index{
		index: index,
		path:  indexPath,
	}, nil
}

// buildIndexMapping creates the index mapping for code chunks.
func buildIndexMapping() mapping.IndexMapping {
	// Create a custom mapping
	indexMapping := bleve.NewIndexMapping()
	
	// Document mapping for chunks
	chunkMapping := bleve.NewDocumentMapping()
	
	// Stored fields (not analyzed, just stored)
	chunkIDField := bleve.NewTextFieldMapping()
	chunkIDField.Analyzer = keyword.Name
	chunkIDField.Store = true
	chunkIDField.Index = true
	chunkMapping.AddFieldMappingsAt("chunk_id", chunkIDField)
	
	repoIDField := bleve.NewTextFieldMapping()
	repoIDField.Analyzer = keyword.Name
	repoIDField.Store = true
	repoIDField.Index = true
	chunkMapping.AddFieldMappingsAt("repo_id", repoIDField)
	
	filePathField := bleve.NewTextFieldMapping()
	filePathField.Analyzer = keyword.Name
	filePathField.Store = true
	filePathField.Index = true
	chunkMapping.AddFieldMappingsAt("file_path", filePathField)
	
	langField := bleve.NewTextFieldMapping()
	langField.Analyzer = keyword.Name
	langField.Store = true
	langField.Index = true
	chunkMapping.AddFieldMappingsAt("lang", langField)
	
	// Searchable text fields (analyzed)
	textField := bleve.NewTextFieldMapping()
	textField.Analyzer = standard.Name
	textField.Store = false
	textField.Index = true
	chunkMapping.AddFieldMappingsAt("text", textField)
	
	symbolNameField := bleve.NewTextFieldMapping()
	symbolNameField.Analyzer = standard.Name
	symbolNameField.Store = false
	symbolNameField.Index = true
	chunkMapping.AddFieldMappingsAt("symbol_name", symbolNameField)
	
	signatureField := bleve.NewTextFieldMapping()
	signatureField.Analyzer = standard.Name
	signatureField.Store = false
	signatureField.Index = true
	chunkMapping.AddFieldMappingsAt("signature", signatureField)
	
	// Set default mapping
	indexMapping.DefaultMapping = chunkMapping
	
	return indexMapping
}

// IndexChunk indexes a single chunk for BM25 search.
func (b *BM25Index) IndexChunk(chunk *Chunk, signature string) error {
	doc := map[string]interface{}{
		"chunk_id":    chunk.ChunkID,
		"repo_id":     chunk.RepoID,
		"file_path":   chunk.FilePath,
		"lang":        chunk.Lang,
		"text":        chunk.Text,
		"symbol_name": chunk.SymbolName,
		"signature":   signature,
	}
	
	return b.index.Index(chunk.ChunkID, doc)
}

// DeleteChunk removes a chunk from the BM25 index.
func (b *BM25Index) DeleteChunk(chunkID string) error {
	return b.index.Delete(chunkID)
}

// Search performs a BM25 search and returns top k results.
func (b *BM25Index) Search(query string, repoID string, globs []string, k int) ([]BM25Result, error) {
	// Build query
	q := bleve.NewMatchQuery(query)
	
	// Add repo filter
	repoQuery := bleve.NewTermQuery(repoID)
	repoQuery.SetField("repo_id")
	
	combinedQuery := bleve.NewConjunctionQuery(q, repoQuery)
	
	// Add glob filters if provided
	if len(globs) > 0 {
		// Build disjunction (OR) of all glob patterns
		disjunction := bleve.NewDisjunctionQuery()
		for _, glob := range globs {
			// Convert glob to wildcard query
			pattern := convertGlobToPattern(glob)
			wildcardQuery := bleve.NewWildcardQuery(pattern)
			wildcardQuery.SetField("file_path")
			disjunction.AddQuery(wildcardQuery)
		}
		
		// Combine with main query using conjunction (AND)
		combinedQuery = bleve.NewConjunctionQuery(combinedQuery, disjunction)
	}
	
	// Execute search
	searchRequest := bleve.NewSearchRequest(combinedQuery)
	searchRequest.Size = k
	searchRequest.Fields = []string{"chunk_id", "file_path", "lang"}
	
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("BM25 search failed: %w", err)
	}
	
	// Convert results
	results := make([]BM25Result, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		result := BM25Result{
			ChunkID: hit.ID,
			Score:   hit.Score,
		}
		
		if filePath, ok := hit.Fields["file_path"].(string); ok {
			result.FilePath = filePath
		}
		if lang, ok := hit.Fields["lang"].(string); ok {
			result.Lang = lang
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// Close closes the BM25 index.
func (b *BM25Index) Close() error {
	return b.index.Close()
}

// convertGlobToPattern converts a glob pattern to a Bleve wildcard pattern.
// Examples: "*.go" -> "*.go", "internal/*" -> "internal/*"
func convertGlobToPattern(glob string) string {
	// Bleve wildcards: * matches any sequence, ? matches single char
	// This is already compatible with basic glob patterns
	
	// Handle ** (match any directory depth)
	pattern := strings.ReplaceAll(glob, "**", "*")
	
	// Ensure it matches from start if no wildcard at beginning
	if !strings.HasPrefix(pattern, "*") {
		pattern = "*" + pattern
	}
	
	return pattern
}

// BatchIndex indexes multiple chunks efficiently.
func (b *BM25Index) BatchIndex(chunks []Chunk, signatures map[string]string) error {
	batch := b.index.NewBatch()
	
	for i := range chunks {
		chunk := &chunks[i]
		sig := signatures[chunk.ChunkID]
		
		doc := map[string]interface{}{
			"chunk_id":    chunk.ChunkID,
			"repo_id":     chunk.RepoID,
			"file_path":   chunk.FilePath,
			"lang":        chunk.Lang,
			"text":        chunk.Text,
			"symbol_name": chunk.SymbolName,
			"signature":   sig,
		}
		
		if err := batch.Index(chunk.ChunkID, doc); err != nil {
			return fmt.Errorf("failed to add chunk %s to batch: %w", chunk.ChunkID, err)
		}
	}
	
	return b.index.Batch(batch)
}

// DeleteByFileID removes all chunks for a given file.
func (b *BM25Index) DeleteByFileID(chunks []string) error {
	batch := b.index.NewBatch()
	
	for _, chunkID := range chunks {
		batch.Delete(chunkID)
	}
	
	return b.index.Batch(batch)
}

// GetPath returns the filesystem path of the BM25 index.
func (b *BM25Index) GetPath() string {
	return b.path
}

