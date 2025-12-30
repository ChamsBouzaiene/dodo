package indexer

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Indexer orchestrates file discovery and indexing.
type Indexer struct {
	db       *DB
	walker   *Walker
	repoID   string
	repoRoot string
}

// IndexerConfig configures the indexer behavior.
type IndexerConfig struct {
	// WalkerConfig for customizing file walker
	WalkerConfig WalkerConfig
}

// NewIndexer creates a new indexer for a repository with default configuration.
func NewIndexer(ctx context.Context, dbPath, repoID, repoRoot string) (*Indexer, error) {
	return NewIndexerWithConfig(ctx, dbPath, repoID, repoRoot, IndexerConfig{})
}

// NewIndexerWithConfig creates a new indexer with custom configuration.
func NewIndexerWithConfig(ctx context.Context, dbPath, repoID, repoRoot string, config IndexerConfig) (*Indexer, error) {
	db, err := NewDB(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Get existing files for fast-path optimization
	existingFiles, err := db.GetAllRepoFiles(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing files: %w", err)
	}

	// Build map for walker fast-path
	existingMap := make(map[string]FileRecord)
	for _, f := range existingFiles {
		existingMap[f.Path] = f
	}
	config.WalkerConfig.ExistingFiles = existingMap

	walker, err := NewWalkerWithConfig(repoRoot, config.WalkerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create walker: %w", err)
	}

	return &Indexer{
		db:       db,
		walker:   walker,
		repoID:   repoID,
		repoRoot: repoRoot,
	}, nil
}

// Close closes the database connection.
func (i *Indexer) Close() error {
	return i.db.Close()
}

// ScanResult contains the results of a repository scan.
type ScanResult struct {
	FilesNeedingIndex []FileInfo
	WalkErrors        []WalkError
	TotalDiscovered   int
	FilesDeleted      int
}

// Scan discovers files in the repository and updates the database.
// Returns a ScanResult with files that need indexing and any errors encountered.
func (i *Indexer) Scan(ctx context.Context) (*ScanResult, error) {
	log.Printf("🔍 Scanning repository: %s", i.repoRoot)

	// Walk repository to discover files (with error collection)
	walkResult := i.walker.WalkWithErrors()

	log.Printf("📁 Discovered %d indexable files", len(walkResult.Files))
	if len(walkResult.Errors) > 0 {
		log.Printf("⚠️  Encountered %d errors during walk", len(walkResult.Errors))
		for _, err := range walkResult.Errors {
			log.Printf("  - %s: %v", err.Path, err.Err)
		}
	}

	// Get existing files from database
	existingFiles, err := i.db.GetAllRepoFiles(ctx, i.repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing files: %w", err)
	}

	// Track discovered paths
	discoveredPaths := make(map[string]bool)
	var needsIndexing []FileInfo

	// Process discovered files
	for _, file := range walkResult.Files {
		discoveredPaths[file.Path] = true

		// Upsert file in database (will mark as pending if new/changed)
		needsIndex, err := i.db.UpsertFile(
			ctx,
			i.repoID,
			file.Path,
			string(file.Lang),
			file.Hash,
			file.SizeBytes,
			file.MtimeUnix,
		)
		if err != nil {
			log.Printf("⚠️  Failed to upsert file %s: %v", file.Path, err)
			walkResult.Errors = append(walkResult.Errors, WalkError{
				Path: file.Path,
				Err:  fmt.Errorf("database upsert failed: %w", err),
			})
			continue
		}

		if needsIndex {
			file.NeedsIndex = true
			needsIndexing = append(needsIndexing, file)
			log.Printf("📝 File needs indexing: %s (%s)", file.Path, file.Lang)
		}
	}

	// Mark deleted files
	deletedCount := 0
	for _, existing := range existingFiles {
		if !discoveredPaths[existing.Path] && !existing.Deleted {
			if err := i.db.MarkDeleted(ctx, i.repoID, existing.Path); err != nil {
				log.Printf("⚠️  Failed to mark file as deleted %s: %v", existing.Path, err)
				walkResult.Errors = append(walkResult.Errors, WalkError{
					Path: existing.Path,
					Err:  fmt.Errorf("failed to mark deleted: %w", err),
				})
				continue
			}
			log.Printf("🗑️  File deleted: %s", existing.Path)
			deletedCount++
		}
	}

	log.Printf("✅ Scan complete: %d files need indexing, %d deleted", len(needsIndexing), deletedCount)

	return &ScanResult{
		FilesNeedingIndex: needsIndexing,
		WalkErrors:        walkResult.Errors,
		TotalDiscovered:   len(walkResult.Files),
		FilesDeleted:      deletedCount,
	}, nil
}

// GetFilesNeedingIndex returns all files that need indexing.
func (i *Indexer) GetFilesNeedingIndex(ctx context.Context) ([]FileRecord, error) {
	return i.db.GetFilesNeedingIndex(ctx, i.repoID)
}

// MarkIndexing marks a file as currently being indexed.
func (i *Indexer) MarkIndexing(ctx context.Context, path string) error {
	return i.db.MarkIndexing(ctx, i.repoID, path)
}

// MarkIndexed marks a file as successfully indexed.
func (i *Indexer) MarkIndexed(ctx context.Context, path string) error {
	return i.db.MarkIndexed(ctx, i.repoID, path)
}

// MarkFailed marks a file as failed to index with an error message.
func (i *Indexer) MarkFailed(ctx context.Context, path, errorMsg string) error {
	return i.db.MarkFailed(ctx, i.repoID, path, errorMsg)
}

// ResetStuckIndexing resets files stuck in 'indexing' state back to 'pending'.
// Returns the number of files reset.
func (i *Indexer) ResetStuckIndexing(ctx context.Context, olderThan time.Duration) (int, error) {
	return i.db.ResetStuckIndexing(ctx, i.repoID, olderThan)
}
