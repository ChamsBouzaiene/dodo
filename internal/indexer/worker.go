package indexer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// IndexingWorker processes files in the background.
type IndexingWorker struct {
	indexer  *Indexer
	chunker  Chunker
	embedder Embedder
	bm25     *BM25Index
	repoID   string
	repoRoot string

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	batchSize    int
	tickInterval time.Duration
}

// NewIndexingWorker creates a new background indexing worker.
func NewIndexingWorker(indexer *Indexer, chunker Chunker, embedder Embedder, bm25 *BM25Index, repoID, repoRoot string) *IndexingWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &IndexingWorker{
		indexer:      indexer,
		chunker:      chunker,
		embedder:     embedder,
		bm25:         bm25,
		repoID:       repoID,
		repoRoot:     repoRoot,
		ctx:          ctx,
		cancel:       cancel,
		batchSize:    20,              // Process up to 20 files per tick
		tickInterval: 5 * time.Second, // Check for work every 5 seconds
	}
}

// Start begins the background indexing loop.
func (w *IndexingWorker) Start() {
	w.wg.Add(1)
	go w.indexingLoop()
}

// Stop stops the background indexing worker.
func (w *IndexingWorker) Stop() {
	w.cancel()
	w.wg.Wait()
}

// indexingLoop continuously processes pending files.
func (w *IndexingWorker) indexingLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()

	log.Printf("🔄 Background indexing worker started (batch size: %d, interval: %v)", w.batchSize, w.tickInterval)

	for {
		select {
		case <-w.ctx.Done():
			log.Println("🛑 Background indexing worker stopped")
			return

		case <-ticker.C:
			w.processBatch()
		}
	}
}

// processBatch processes a batch of pending files.
func (w *IndexingWorker) processBatch() {
	// Get pending files
	files, err := w.indexer.GetFilesNeedingIndex(w.ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get pending files: %v", err)
		return
	}

	if len(files) == 0 {
		return // Nothing to do
	}

	// Limit to batch size
	if len(files) > w.batchSize {
		files = files[:w.batchSize]
	}

	log.Printf("📦 Processing batch of %d files", len(files))

	// Process each file
	for _, file := range files {
		if err := w.processFile(file); err != nil {
			log.Printf("❌ Failed to index %s: %v", file.Path, err)
		}
	}
}

// processFile processes a single file: chunk, embed, and store.
func (w *IndexingWorker) processFile(file FileRecord) error {
	// Mark as indexing
	if err := w.indexer.MarkIndexing(w.ctx, file.Path); err != nil {
		return fmt.Errorf("failed to mark as indexing: %w", err)
	}

	// Read file content
	fullPath := filepath.Join(w.repoRoot, file.Path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// File might have been deleted
		if os.IsNotExist(err) {
			// Mark as deleted in DB
			w.indexer.db.MarkDeleted(w.ctx, w.repoID, file.Path)
			return nil
		}
		return w.markFailed(file.Path, fmt.Errorf("failed to read file: %w", err))
	}

	// Create FileInfo for chunking
	fileInfo := FileInfo{
		Path:      file.Path,
		Lang:      Language(file.Lang),
		Hash:      file.Hash,
		SizeBytes: file.SizeBytes,
		MtimeUnix: file.MtimeUnix,
	}

	// Chunk the file
	chunks, symbols, err := w.chunker.Chunk(w.ctx, fileInfo, content)
	if err != nil {
		return w.markFailed(file.Path, fmt.Errorf("failed to chunk file: %w", err))
	}

	// Delete old chunks/symbols/embeddings/BM25 entries for this file
	// Get old chunk IDs before deleting
	oldChunks, err := w.indexer.db.GetChunksByFile(w.ctx, file.FileID)
	if err != nil {
		log.Printf("⚠️  Failed to get old chunks: %v", err)
	} else if len(oldChunks) > 0 && w.bm25 != nil {
		// Delete from BM25 index
		oldChunkIDs := make([]string, len(oldChunks))
		for i, c := range oldChunks {
			oldChunkIDs[i] = c.ChunkID
		}
		if err := w.bm25.DeleteByFileID(oldChunkIDs); err != nil {
			log.Printf("⚠️  Failed to delete old chunks from BM25: %v", err)
		}
	}

	if err := w.indexer.db.DeleteChunksByFile(w.ctx, file.FileID); err != nil {
		log.Printf("⚠️  Failed to delete old chunks: %v", err)
	}
	if err := w.indexer.db.DeleteSymbolsByFile(w.ctx, file.FileID); err != nil {
		log.Printf("⚠️  Failed to delete old symbols: %v", err)
	}
	if err := w.indexer.db.DeleteEmbeddingsByFile(w.ctx, file.FileID); err != nil {
		log.Printf("⚠️  Failed to delete old embeddings: %v", err)
	}

	// Insert symbols
	for i := range symbols {
		symbols[i].RepoID = w.repoID
		symbols[i].FileID = file.FileID
		if err := w.indexer.db.InsertSymbol(w.ctx, &symbols[i]); err != nil {
			log.Printf("⚠️  Failed to insert symbol %s: %v", symbols[i].Name, err)
		}
	}

	// Insert chunks and generate embeddings
	if len(chunks) > 0 {
		// Collect chunk texts for batch embedding
		chunkTexts := make([]string, len(chunks))
		for i, chunk := range chunks {
			chunkTexts[i] = chunk.Text
		}

		// Generate embeddings in batch
		embeddings, dim, err := w.embedder.EmbedBatch(w.ctx, chunkTexts)
		if err != nil {
			log.Printf("⚠️  Failed to generate embeddings for %s: %v", file.Path, err)
			// Continue without embeddings
			embeddings = nil
		}

		// Build signature map for BM25 indexing
		signatureMap := make(map[string]string)
		for i := range symbols {
			signatureMap[symbols[i].SymbolID] = symbols[i].Signature
		}

		// Insert chunks and embeddings
		for i := range chunks {
			chunks[i].RepoID = w.repoID
			chunks[i].FileID = file.FileID

			if err := w.indexer.db.InsertChunk(w.ctx, &chunks[i]); err != nil {
				log.Printf("⚠️  Failed to insert chunk: %v", err)
				continue
			}

			// Index in BM25
			if w.bm25 != nil {
				signature := signatureMap[chunks[i].SymbolID]
				if err := w.bm25.IndexChunk(&chunks[i], signature); err != nil {
					log.Printf("⚠️  Failed to index chunk in BM25: %v", err)
				}
			}

			// Insert embedding if available
			if embeddings != nil && i < len(embeddings) {
				embedding := Embedding{
					ChunkID: chunks[i].ChunkID,
					RepoID:  w.repoID,
					Dim:     dim,
					Vector:  embeddings[i],
				}
				if err := w.indexer.db.InsertEmbedding(w.ctx, &embedding); err != nil {
					log.Printf("⚠️  Failed to insert embedding: %v", err)
				}
			}
		}
	}

	// Mark as successfully indexed
	if err := w.indexer.MarkIndexed(w.ctx, file.Path); err != nil {
		return fmt.Errorf("failed to mark as indexed: %w", err)
	}

	log.Printf("✅ Indexed %s (%d symbols, %d chunks)", file.Path, len(symbols), len(chunks))
	return nil
}

// markFailed marks a file as failed to index.
func (w *IndexingWorker) markFailed(path string, err error) error {
	errMsg := err.Error()
	if len(errMsg) > 500 {
		errMsg = errMsg[:500] // Truncate long error messages
	}
	return w.indexer.MarkFailed(w.ctx, path, errMsg)
}

// RunIndexingBatch processes up to N pending files immediately.
// This is used for quick freshness before starting the agent.
func (w *IndexingWorker) RunIndexingBatch(ctx context.Context, maxFiles int) error {
	// Get pending files
	files, err := w.indexer.GetFilesNeedingIndex(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending files: %w", err)
	}

	if len(files) == 0 {
		return nil // Nothing to do
	}

	// Limit to maxFiles
	if len(files) > maxFiles {
		files = files[:maxFiles]
	}

	log.Printf("📦 Quick indexing batch: %d files", len(files))

	// Process each file
	for _, file := range files {
		if err := w.processFile(file); err != nil {
			log.Printf("❌ Failed to index %s: %v", file.Path, err)
			// Continue with next file
		}
	}

	return nil
}
