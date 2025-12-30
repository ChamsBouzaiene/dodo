package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Manager orchestrates the entire indexing system:
// - Git integration for change detection
// - File watching for real-time updates
// - Scheduled safety scans
// - Background indexing worker
type Manager struct {
	// Core components
	indexer  *Indexer
	worker   *IndexingWorker
	watcher  *FileWatcher
	chunker  Chunker
	embedder Embedder
	bm25     *BM25Index

	// Repository info
	repoID   string
	repoRoot string
	gitInfo  GitInfo

	// Configuration
	config ManagerConfig

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex

	started bool

	// Workspace context caching
	cachedContext *WorkspaceContext
	contextMu     sync.RWMutex
}

// ManagerConfig configures the manager behavior.
type ManagerConfig struct {
	// Database path
	DBPath string

	// Repo identification
	RepoID   string
	RepoRoot string

	// Components (optional, will use defaults if nil)
	Chunker  Chunker
	Embedder Embedder

	// File watching
	EnableFileWatcher bool

	// Safety scan interval (periodic full scan as backup)
	SafetyScanInterval time.Duration

	// Indexing worker config
	WorkerBatchSize    int
	WorkerTickInterval time.Duration

	// Search scoring configuration
	CodeFileBoost float64 // Boost factor for code files vs documentation (default: 1.2)
}

// NewManager creates a new indexing manager.
func NewManager(ctx context.Context, config ManagerConfig) (*Manager, error) {
	// Validate config
	if config.DBPath == "" {
		return nil, fmt.Errorf("DBPath is required")
	}
	if config.RepoID == "" {
		return nil, fmt.Errorf("RepoID is required")
	}
	if config.RepoRoot == "" {
		return nil, fmt.Errorf("RepoRoot is required")
	}

	// Set defaults
	if config.Chunker == nil {
		config.Chunker = NewDefaultChunker()
	}
	if config.Embedder == nil {
		config.Embedder = NewNoOpEmbedder(384) // Default to no-op embedder
	}
	if config.SafetyScanInterval == 0 {
		config.SafetyScanInterval = 10 * time.Minute
	}
	if config.WorkerBatchSize == 0 {
		config.WorkerBatchSize = 20
	}
	if config.WorkerTickInterval == 0 {
		config.WorkerTickInterval = 5 * time.Second
	}
	if config.CodeFileBoost == 0 {
		config.CodeFileBoost = 1.2 // Default: 20% boost for code files
	}

	// Detect git
	gitInfo := DetectGit(ctx, config.RepoRoot)
	log.Printf("🔍 Repository: %s (git: %v)", config.RepoRoot, gitInfo.IsGit)

	// Create indexer
	log.Println("[DEBUG] Creating indexer...")
	indexer, err := NewIndexer(ctx, config.DBPath, config.RepoID, config.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexer: %w", err)
	}
	log.Println("[DEBUG] Indexer created")

	// Store repo info in database
	log.Println("[DEBUG] Upserting repo info...")
	if err := indexer.db.UpsertRepo(ctx, config.RepoID, config.RepoRoot, gitInfo.IsGit, gitInfo.GitRoot); err != nil {
		log.Printf("⚠️  Failed to store repo info: %v", err)
	}
	log.Println("[DEBUG] Repo info upserted")

	// Create BM25 index
	log.Println("[DEBUG] Creating BM25 index...")
	bm25, err := NewBM25Index(config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create BM25 index: %w", err)
	}
	log.Println("📚 BM25 index ready")

	// Create worker
	worker := NewIndexingWorker(indexer, config.Chunker, config.Embedder, bm25, config.RepoID, config.RepoRoot)
	worker.batchSize = config.WorkerBatchSize
	worker.tickInterval = config.WorkerTickInterval

	mgrCtx, cancel := context.WithCancel(context.Background())

	m := &Manager{
		indexer:  indexer,
		worker:   worker,
		chunker:  config.Chunker,
		embedder: config.Embedder,
		bm25:     bm25,
		repoID:   config.RepoID,
		repoRoot: config.RepoRoot,
		gitInfo:  gitInfo,
		config:   config,
		ctx:      mgrCtx,
		cancel:   cancel,
	}

	// Create file watcher if enabled
	if config.EnableFileWatcher {
		walker, err := NewWalker(config.RepoRoot)
		if err == nil {
			watcher, err := NewFileWatcher(config.RepoRoot, NewDefaultLanguageDetector(), walker.ignoreMatcher)
			if err != nil {
				log.Printf("⚠️  Failed to create file watcher: %v", err)
			} else {
				m.watcher = watcher
				// Set up change callbacks
				watcher.OnChange(m.handleFileChanges)
				watcher.OnStructureChange(m.InvalidateWorkspaceContext)
			}
		}
	}

	return m, nil
}

// Start begins all background processes:
// - File watcher (if enabled)
// - Safety scan ticker
// - Indexing worker
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("manager already started")
	}

	log.Println("🚀 Starting indexing manager")

	// Reset any stuck files from previous crashes
	count, err := m.indexer.ResetStuckIndexing(m.ctx, 1*time.Hour)
	if err != nil {
		log.Printf("⚠️  Failed to reset stuck files: %v", err)
	} else if count > 0 {
		log.Printf("🔄 Reset %d stuck files from previous run", count)
	}

	// Start file watcher
	if m.watcher != nil {
		if err := m.watcher.Start(); err != nil {
			log.Printf("⚠️  Failed to start file watcher: %v", err)
		} else {
			log.Println("👀 File watcher started")
		}
	}

	// Start safety scan loop
	m.wg.Add(1)
	go m.safetyScanLoop()

	// Start indexing worker
	m.worker.Start()

	m.started = true
	log.Println("✅ Indexing manager started")

	return nil
}

// Stop stops all background processes.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	log.Println("🛑 Stopping indexing manager")

	// Stop file watcher
	if m.watcher != nil {
		m.watcher.Stop()
	}

	// Stop worker
	m.worker.Stop()

	// Stop safety scan loop
	m.cancel()
	m.wg.Wait()

	// Close BM25 index
	if m.bm25 != nil {
		m.bm25.Close()
	}

	// Close indexer
	m.indexer.Close()

	m.started = false
	log.Println("✅ Indexing manager stopped")

	return nil
}

// InitialIndex performs the initial full indexing of the repository.
// This should be called once on first use to build the complete index.
func (m *Manager) InitialIndex(ctx context.Context) error {
	log.Println("🔨 Starting initial full indexing")

	// Step 1: Discover all files
	result, err := m.discoverFiles(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	log.Printf("📊 Discovery complete: %d files discovered, %d need indexing", result.TotalDiscovered, len(result.FilesNeedingIndex))

	// Step 2: Index all pending files
	// Get all pending files (not just from scan result, in case there are old pending files)
	pending, err := m.indexer.GetFilesNeedingIndex(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending files: %w", err)
	}

	if len(pending) == 0 {
		log.Println("✅ No files need indexing")
		return nil
	}

	log.Printf("⚙️  Indexing %d files...", len(pending))

	// Process all files
	for i, file := range pending {
		if i%100 == 0 {
			log.Printf("Progress: %d/%d files", i, len(pending))
		}
		if err := m.worker.processFile(file); err != nil {
			log.Printf("❌ Failed to index %s: %v", file.Path, err)
		}
	}

	log.Println("✅ Initial indexing complete")
	return nil
}

// QuickFreshness performs a quick freshness check and indexes up to N pending files.
// This is called before starting the agent to ensure index is reasonably fresh.
func (m *Manager) QuickFreshness(ctx context.Context, maxFiles int) error {
	// Step 1: Quick change detection
	changed, err := m.detectChanges(ctx)
	if err != nil {
		log.Printf("⚠️  Change detection failed: %v", err)
	} else if len(changed) > 0 {
		log.Printf("📝 Detected %d changed files", len(changed))
		// Invalidate workspace context if files changed (may include structural changes)
		m.InvalidateWorkspaceContext()
	}

	// Step 2: Index up to maxFiles pending files
	return m.worker.RunIndexingBatch(ctx, maxFiles)
}

// discoverFiles discovers all files in the repository.
func (m *Manager) discoverFiles(ctx context.Context) (*ScanResult, error) {
	if m.gitInfo.IsGit {
		return m.discoverGitFiles(ctx)
	}
	return m.discoverNonGitFiles(ctx)
}

// discoverGitFiles uses git ls-files for git repositories.
func (m *Manager) discoverGitFiles(ctx context.Context) (*ScanResult, error) {
	// Get git-tracked files
	gitFiles, err := GetGitTrackedFiles(ctx, m.gitInfo.GitRoot)
	if err != nil {
		log.Printf("⚠️  git ls-files failed, falling back to filesystem walk: %v", err)
		return m.discoverNonGitFiles(ctx)
	}

	// Filter to only files we care about
	walker, err := NewWalker(m.repoRoot)
	if err != nil {
		return nil, err
	}

	langDetector := NewDefaultLanguageDetector()
	var filesToIndex []FileInfo

	for _, relPath := range gitFiles {
		fullPath := filepath.Join(m.repoRoot, relPath)

		// Detect language
		lang := langDetector.Detect(fullPath)
		if lang == "" {
			continue // Skip files we don't index
		}

		// Get file metadata
		info, err := walker.getFileInfo(fullPath, relPath, lang)
		if err != nil {
			log.Printf("⚠️  Failed to get info for %s: %v", relPath, err)
			continue
		}

		filesToIndex = append(filesToIndex, *info)
	}

	// Update database with discovered files
	return m.indexer.Scan(ctx)
}

// discoverNonGitFiles uses filesystem walk for non-git repositories.
func (m *Manager) discoverNonGitFiles(ctx context.Context) (*ScanResult, error) {
	return m.indexer.Scan(ctx)
}

// detectChanges detects file changes using git (if available) or filesystem walk.
func (m *Manager) detectChanges(ctx context.Context) ([]string, error) {
	if m.gitInfo.IsGit {
		return m.detectGitChanges(ctx)
	}
	return m.detectNonGitChanges(ctx)
}

// detectGitChanges uses git status --porcelain for change detection.
func (m *Manager) detectGitChanges(ctx context.Context) ([]string, error) {
	changes, err := GetGitChanges(ctx, m.gitInfo.GitRoot)
	if err != nil {
		return nil, err
	}

	var changedPaths []string
	for _, change := range changes {
		changedPaths = append(changedPaths, change.Path)

		// Update database based on change type
		if change.Status == "D" {
			// Deleted file
			m.indexer.db.MarkDeleted(ctx, m.repoID, change.Path)
		} else {
			// Added or modified file - check if hash changed
			fullPath := filepath.Join(m.repoRoot, change.Path)
			if _, err := os.Stat(fullPath); err == nil {
				// File exists, will be picked up by scan
				// We don't update here, let the scan handle it
			}
		}
	}

	// Run a scan to update everything
	if len(changedPaths) > 0 {
		m.indexer.Scan(ctx)
	}

	return changedPaths, nil
}

// detectNonGitChanges uses filesystem walk with fast-path for change detection.
func (m *Manager) detectNonGitChanges(ctx context.Context) ([]string, error) {
	// Just run a scan with fast-path optimization
	result, err := m.indexer.Scan(ctx)
	if err != nil {
		return nil, err
	}

	var changedPaths []string
	for _, file := range result.FilesNeedingIndex {
		changedPaths = append(changedPaths, file.Path)
	}

	return changedPaths, nil
}

// handleFileChanges is called by the file watcher when files change.
func (m *Manager) handleFileChanges(paths []string) {
	// For each changed path, mark it as pending in the database
	for _, path := range paths {
		// Check if file still exists
		fullPath := filepath.Join(m.repoRoot, path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File deleted
			m.indexer.db.MarkDeleted(m.ctx, m.repoID, path)
			continue
		}

		// File exists - will be picked up by next scan or worker
		// We don't need to do anything here, the scan will detect the change
	}

	// Trigger a quick scan to update changed files
	go func() {
		m.detectChanges(m.ctx)
	}()
}

// safetyScanLoop runs periodic full scans as a safety net.
func (m *Manager) safetyScanLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.SafetyScanInterval)
	defer ticker.Stop()

	log.Printf("🔄 Safety scan loop started (interval: %v)", m.config.SafetyScanInterval)

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			log.Println("🔍 Running safety scan")
			if _, err := m.detectChanges(m.ctx); err != nil {
				log.Printf("⚠️  Safety scan failed: %v", err)
			}
		}
	}
}

// GetIndexer returns the underlying indexer for direct access.
func (m *Manager) GetIndexer() *Indexer {
	return m.indexer
}

// GetDB returns the database for direct access.
func (m *Manager) GetDB() *DB {
	return m.indexer.db
}

// ReadSpan reads the source code for a specific span from a file.
// Returns the text content from start line to end line (inclusive, 1-indexed).
func (m *Manager) ReadSpan(ctx context.Context, path string, start, end int) (string, error) {
	// Resolve full path
	fullPath := filepath.Join(m.repoRoot, path)

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Split into lines
	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// Clamp both values to valid range (1-indexed)
	if start < 1 {
		start = 1
	}
	if start > totalLines {
		start = totalLines
	}
	if end < 1 {
		end = 1
	}
	if end > totalLines {
		end = totalLines
	}

	// If start > end after clamping, swap them (agent may have reversed them)
	if start > end {
		start, end = end, start
	}

	// Ensure we have at least one line to return
	if totalLines == 0 {
		return "", nil
	}

	// Extract lines (convert to 0-indexed for slice)
	selectedLines := lines[start-1 : end]

	return strings.Join(selectedLines, "\n"), nil
}

// Search finds the top k most relevant code spans for a query using hybrid search.
// Combines BM25 (keyword) and vector (semantic) search using Reciprocal Rank Fusion (RRF).
func (m *Manager) Search(ctx context.Context, query string, globs []string, k int) ([]Span, error) {
	// Default k if not specified
	if k <= 0 {
		k = 10
	}

	// Use hybrid search if BM25 is available, otherwise fall back to embeddings-only
	if m.bm25 == nil {
		return m.searchEmbeddingsOnly(ctx, query, globs, k)
	}

	// Phase 1: BM25 search (keyword matching)
	const Nbm25 = 100 // Top 100 from BM25
	bm25Results, err := m.bm25.Search(query, m.repoID, globs, Nbm25)
	if err != nil {
		log.Printf("⚠️  BM25 search failed: %v", err)
		bm25Results = []BM25Result{} // Continue with embeddings only
	}

	// Phase 2: Embedding search (semantic matching)
	const Nvec = 100 // Top 100 from embeddings
	vecResults, err := m.searchEmbeddings(ctx, query, globs, Nvec)
	if err != nil {
		log.Printf("⚠️  Embedding search failed: %v", err)
		vecResults = []scoredChunk{} // Continue with BM25 only
	}

	// Phase 3: RRF (Reciprocal Rank Fusion) merge
	const kOffset = 60.0
	scores := make(map[string]float64)

	// Add BM25 scores
	for i, r := range bm25Results {
		scores[r.ChunkID] += 1.0 / (kOffset + float64(i+1))
	}

	// Add embedding scores
	for i, r := range vecResults {
		scores[r.chunk.ChunkID] += 1.0 / (kOffset + float64(i+1))
	}

	// Sort by combined RRF score
	type rrfResult struct {
		chunkID string
		score   float64
	}

	rrfResults := make([]rrfResult, 0, len(scores))
	for chunkID, score := range scores {
		rrfResults = append(rrfResults, rrfResult{chunkID, score})
	}

	sort.Slice(rrfResults, func(i, j int) bool {
		return rrfResults[i].score > rrfResults[j].score
	})

	// Take top k
	if len(rrfResults) > k {
		rrfResults = rrfResults[:k]
	}

	// Phase 4: Fetch chunks and build Spans with code file boost
	type scoredSpan struct {
		span  Span
		score float64
	}

	scoredSpans := make([]scoredSpan, 0, len(rrfResults))
	for _, rrf := range rrfResults {
		// Fetch chunk from database
		var chunk Chunk
		var symbolID, symbolName sql.NullString
		query := `SELECT chunk_id, repo_id, file_id, file_path, lang, symbol_id, symbol_name, kind, start_line, end_line, text FROM chunks WHERE chunk_id = ?`
		err := m.indexer.db.db.QueryRowContext(ctx, query, rrf.chunkID).Scan(&chunk.ChunkID, &chunk.RepoID, &chunk.FileID, &chunk.FilePath, &chunk.Lang, &symbolID, &symbolName, &chunk.Kind, &chunk.StartLine, &chunk.EndLine, &chunk.Text)
		if err != nil {
			log.Printf("⚠️  Failed to fetch chunk %s: %v", rrf.chunkID, err)
			continue
		}
		if symbolID.Valid {
			chunk.SymbolID = symbolID.String
		}
		if symbolName.Valid {
			chunk.SymbolName = symbolName.String
		}

		// Extract snippet
		snippet := extractSnippet(chunk.Text, 30)

		// Boost code files over documentation
		score := rrf.score
		if isCodeFile(chunk.FilePath) {
			score *= m.config.CodeFileBoost
		}

		span := Span{
			Path:    chunk.FilePath,
			Start:   chunk.StartLine,
			End:     chunk.EndLine,
			Lang:    chunk.Lang,
			Snippet: snippet,
			Score:   score,
			Reason:  "rrf(bm25+vec)",
		}
		scoredSpans = append(scoredSpans, scoredSpan{span, score})
	}

	// Re-sort by boosted score
	sort.Slice(scoredSpans, func(i, j int) bool {
		return scoredSpans[i].score > scoredSpans[j].score
	})

	// Take top k after boosting
	if len(scoredSpans) > k {
		scoredSpans = scoredSpans[:k]
	}

	// Extract spans
	spans := make([]Span, len(scoredSpans))
	for i, ss := range scoredSpans {
		spans[i] = ss.span
	}

	return spans, nil
}

// searchEmbeddingsOnly performs embeddings-only search (fallback when BM25 unavailable).
func (m *Manager) searchEmbeddingsOnly(ctx context.Context, query string, globs []string, k int) ([]Span, error) {
	vecResults, err := m.searchEmbeddings(ctx, query, globs, k)
	if err != nil {
		return nil, err
	}

	// Boost code files and re-sort
	type scoredSpan struct {
		span  Span
		score float64
	}

	scoredSpans := make([]scoredSpan, len(vecResults))
	for i, sc := range vecResults {
		snippet := extractSnippet(sc.chunk.Text, 30)

		// Boost code files over documentation
		score := sc.score
		if isCodeFile(sc.chunk.FilePath) {
			score *= m.config.CodeFileBoost
		}

		span := Span{
			Path:    sc.chunk.FilePath,
			Start:   sc.chunk.StartLine,
			End:     sc.chunk.EndLine,
			Lang:    sc.chunk.Lang,
			Snippet: snippet,
			Score:   score,
			Reason:  "embedding_only",
		}
		scoredSpans[i] = scoredSpan{span, score}
	}

	// Re-sort by boosted score
	sort.Slice(scoredSpans, func(i, j int) bool {
		return scoredSpans[i].score > scoredSpans[j].score
	})

	// Extract spans
	spans := make([]Span, len(scoredSpans))
	for i, ss := range scoredSpans {
		spans[i] = ss.span
	}

	return spans, nil
}

// scoredChunk is used for embedding search results.
type scoredChunk struct {
	chunk Chunk
	score float64
}

// searchEmbeddings performs embedding-based semantic search.
func (m *Manager) searchEmbeddings(ctx context.Context, query string, globs []string, k int) ([]scoredChunk, error) {
	// Step 1: Embed the query
	queryVec, _, err := m.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Step 2: Get candidate chunks
	candidates, err := m.getCandidateChunks(ctx, globs, 500)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidates: %w", err)
	}

	if len(candidates) == 0 {
		return []scoredChunk{}, nil
	}

	// Step 3: Compute cosine similarity
	queryVector, err := DecodeVector(queryVec)
	if err != nil {
		return nil, fmt.Errorf("failed to decode query vector: %w", err)
	}

	scored := make([]scoredChunk, 0, len(candidates))
	for _, candidate := range candidates {
		// Get embedding
		var embeddingData []byte
		embQuery := `SELECT vector FROM embeddings WHERE chunk_id = ?`
		err := m.indexer.db.db.QueryRowContext(ctx, embQuery, candidate.ChunkID).Scan(&embeddingData)
		if err != nil {
			continue // Skip chunks without embeddings
		}

		chunkVector, err := DecodeVector(embeddingData)
		if err != nil {
			continue
		}

		similarity := cosineSimilarity(queryVector, chunkVector)
		scored = append(scored, scoredChunk{chunk: candidate, score: similarity})
	}

	// Step 4: Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Step 5: Take top k
	if len(scored) > k {
		scored = scored[:k]
	}

	return scored, nil
}

// getCandidateChunks retrieves chunks from the database filtered by globs.
func (m *Manager) getCandidateChunks(ctx context.Context, globs []string, limit int) ([]Chunk, error) {
	query := `
		SELECT chunk_id, repo_id, file_id, file_path, lang, symbol_id, symbol_name, kind, start_line, end_line, text
		FROM chunks
		WHERE repo_id = ?
	`

	args := []interface{}{m.repoID}

	// Add glob filtering
	if len(globs) > 0 {
		query += " AND ("
		for i, glob := range globs {
			if i > 0 {
				query += " OR "
			}
			query += "file_path LIKE ?"
			// Convert glob to SQL LIKE pattern (basic implementation)
			pattern := strings.ReplaceAll(glob, "*", "%")
			args = append(args, pattern)
		}
		query += ")"
	}

	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := m.indexer.db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		var symbolID, symbolName sql.NullString
		err := rows.Scan(&c.ChunkID, &c.RepoID, &c.FileID, &c.FilePath, &c.Lang, &symbolID, &symbolName, &c.Kind, &c.StartLine, &c.EndLine, &c.Text)
		if err != nil {
			return nil, err
		}
		if symbolID.Valid {
			c.SymbolID = symbolID.String
		}
		if symbolName.Valid {
			c.SymbolName = symbolName.String
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// extractSnippet extracts the first n lines from text for preview.
func extractSnippet(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n") + "\n..."
}

// isCodeFile checks if a file is a code file (not documentation).
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go": true, ".py": true, ".ts": true, ".tsx": true,
		".js": true, ".jsx": true, ".java": true, ".cpp": true,
		".c": true, ".rs": true, ".rb": true, ".php": true,
		".swift": true, ".kt": true, ".scala": true, ".clj": true,
		".hs": true, ".ml": true, ".sh": true, ".bash": true,
		".zsh": true, ".fish": true, ".ps1": true,
	}
	return codeExts[ext]
}

// GetWorkspaceContext returns the cached workspace context or generates a new one.
func (m *Manager) GetWorkspaceContext() *WorkspaceContext {
	// Try to read from cache first
	m.contextMu.RLock()
	if m.cachedContext != nil {
		cached := m.cachedContext
		m.contextMu.RUnlock()
		return cached
	}
	m.contextMu.RUnlock()

	// Generate new context
	m.contextMu.Lock()
	defer m.contextMu.Unlock()

	// Double-check after acquiring write lock
	if m.cachedContext != nil {
		return m.cachedContext
	}

	// Generate workspace context
	ctx := context.Background() // Use background context for generation
	wsCtx, err := GenerateWorkspaceContext(ctx, m.repoRoot, m.gitInfo)
	if err != nil {
		log.Printf("⚠️  Failed to generate workspace context: %v", err)
		// Return a minimal context on error
		return &WorkspaceContext{
			UserInfo: UserInfo{
				OS:    "unknown",
				Shell: "unknown",
			},
			ProjectLayout: m.repoRoot + "/\n  [generation failed]",
			GitStatus: GitStatus{
				Branch: "unknown",
				Status: "unknown",
			},
			ProjectType: "unknown",
		}
	}

	m.cachedContext = wsCtx
	return wsCtx
}

// InvalidateWorkspaceContext clears the cached workspace context.
// Should be called when file structure changes (create/delete/move).
func (m *Manager) InvalidateWorkspaceContext() {
	m.contextMu.Lock()
	defer m.contextMu.Unlock()
	m.cachedContext = nil
}
