package indexer

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches filesystem changes and triggers file processing.
type FileWatcher struct {
	repoRoot         string
	watcher          *fsnotify.Watcher
	onChange         func([]string) // Callback with changed file paths
	onStructureChange func()         // Callback for structural changes (create/delete/rename)
	debounceTime     time.Duration
	mu               sync.Mutex
	pendingEvents    map[string]bool
	structuralChange bool // Tracks if any structural changes occurred
	langDetector     LanguageDetector
	ignoreMatcher    interface{ MatchesPath(string) bool }
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewFileWatcher creates a new file system watcher.
func NewFileWatcher(repoRoot string, langDetector LanguageDetector, ignoreMatcher interface{ MatchesPath(string) bool }) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	fw := &FileWatcher{
		repoRoot:      repoRoot,
		watcher:       watcher,
		debounceTime:  500 * time.Millisecond, // Debounce events by 500ms
		pendingEvents: make(map[string]bool),
		langDetector:  langDetector,
		ignoreMatcher: ignoreMatcher,
		ctx:           ctx,
		cancel:        cancel,
	}
	
	return fw, nil
}

// OnChange sets the callback function for file changes.
// The callback receives a list of changed file paths (relative to repo root).
func (fw *FileWatcher) OnChange(callback func([]string)) {
	fw.onChange = callback
}

// OnStructureChange sets the callback function for structural changes (create/delete/rename).
func (fw *FileWatcher) OnStructureChange(callback func()) {
	fw.onStructureChange = callback
}

// Start begins watching the filesystem.
func (fw *FileWatcher) Start() error {
	// Walk the repo to add all directories to the watcher
	err := filepath.WalkDir(fw.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}
		
		// Get relative path
		relPath, err := filepath.Rel(fw.repoRoot, path)
		if err != nil {
			return nil
		}
		
		// Check if should be ignored
		if fw.ignoreMatcher != nil && fw.ignoreMatcher.MatchesPath(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Watch directories
		if d.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				log.Printf("⚠️  Failed to watch %s: %v", path, err)
			}
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to walk repo: %w", err)
	}
	
	// Start event processing goroutines
	fw.wg.Add(2)
	go fw.eventLoop()
	go fw.debounceLoop()
	
	return nil
}

// Stop stops the file watcher.
func (fw *FileWatcher) Stop() error {
	fw.cancel()
	fw.wg.Wait()
	return fw.watcher.Close()
}

// eventLoop processes filesystem events.
func (fw *FileWatcher) eventLoop() {
	defer fw.wg.Done()
	
	for {
		select {
		case <-fw.ctx.Done():
			return
			
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			
			fw.handleEvent(event)
			
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("⚠️  Watcher error: %v", err)
		}
	}
}

// handleEvent processes a single filesystem event.
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Get relative path
	relPath, err := filepath.Rel(fw.repoRoot, event.Name)
	if err != nil {
		return
	}
	
	// Check if should be ignored
	if fw.ignoreMatcher != nil && fw.ignoreMatcher.MatchesPath(relPath) {
		return
	}
	
	// Check if it's a file we care about
	lang := fw.langDetector.Detect(event.Name)
	if lang == "" && !event.Has(fsnotify.Remove) {
		// Not a file type we index, and not a deletion
		return
	}
	
	// Handle new directories
	if event.Has(fsnotify.Create) {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			// New directory - watch it
			if err := fw.watcher.Add(event.Name); err != nil {
				log.Printf("⚠️  Failed to watch new directory %s: %v", event.Name, err)
			}
			return
		}
	}
	
	// File created, modified, or deleted - add to pending events
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		fw.mu.Lock()
		fw.pendingEvents[relPath] = true
		
		// Track structural changes (create/delete/rename) for workspace context invalidation
		if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
			fw.structuralChange = true
		}
		
		fw.mu.Unlock()
	}
}

// debounceLoop collects pending events and triggers callbacks after debounce period.
func (fw *FileWatcher) debounceLoop() {
	defer fw.wg.Done()
	
	ticker := time.NewTicker(fw.debounceTime)
	defer ticker.Stop()
	
	for {
		select {
		case <-fw.ctx.Done():
			return
			
		case <-ticker.C:
			fw.processPendingEvents()
		}
	}
}

// processPendingEvents processes all pending file change events.
func (fw *FileWatcher) processPendingEvents() {
	fw.mu.Lock()
	if len(fw.pendingEvents) == 0 {
		fw.mu.Unlock()
		return
	}
	
	// Copy pending events and structural change flag, then clear
	paths := make([]string, 0, len(fw.pendingEvents))
	for path := range fw.pendingEvents {
		paths = append(paths, path)
	}
	hadStructuralChange := fw.structuralChange
	fw.pendingEvents = make(map[string]bool)
	fw.structuralChange = false
	fw.mu.Unlock()
	
	// Trigger file change callback
	if fw.onChange != nil {
		log.Printf("📝 File watcher detected %d changed files", len(paths))
		fw.onChange(paths)
	}
	
	// Trigger structural change callback if needed
	if hadStructuralChange && fw.onStructureChange != nil {
		fw.onStructureChange()
	}
}

