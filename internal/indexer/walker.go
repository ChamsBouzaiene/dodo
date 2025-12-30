package indexer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gitignore "github.com/sabhiram/go-gitignore"
)

// Language represents a programming language.
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "ts"
	LangJavaScript Language = "js"
	LangPython     Language = "python"
	LangRust       Language = "rust"
	LangJava       Language = "java"
	LangC          Language = "c"
	LangCPP        Language = "cpp"
	LangMarkdown   Language = "markdown"
	LangJSON       Language = "json"
	LangYAML       Language = "yaml"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
)

// FileInfo contains metadata about a discovered file.
type FileInfo struct {
	Path       string
	Lang       Language
	Hash       string
	SizeBytes  int64
	MtimeUnix  int64
	NeedsIndex bool
}

// WalkError represents an error that occurred during file walking.
type WalkError struct {
	Path string
	Err  error
}

func (e *WalkError) Error() string {
	return fmt.Sprintf("%s: %v", e.Path, e.Err)
}

// WalkResult contains the results of a repository walk.
type WalkResult struct {
	Files  []FileInfo
	Errors []WalkError
}

// DefaultIgnorePatterns are common directories and files to skip.
var DefaultIgnorePatterns = []string{
	".git",
	"node_modules",
	"dist",
	"build",
	"vendor",
	"__pycache__",
	"coverage",
	".next",
	".cache",
	"target",
	"bin",
	"obj",
	".idea",
	".vscode",
	".DS_Store",
}

// LanguageDetector defines how to detect file languages.
type LanguageDetector interface {
	Detect(path string) Language
}

// DefaultLanguageDetector detects language from file extension.
type DefaultLanguageDetector struct {
	extMap map[string]Language
}

// NewDefaultLanguageDetector creates a new default language detector.
func NewDefaultLanguageDetector() *DefaultLanguageDetector {
	return &DefaultLanguageDetector{
		extMap: map[string]Language{
			".go":   LangGo,
			".ts":   LangTypeScript,
			".tsx":  LangTypeScript,
			".js":   LangJavaScript,
			".jsx":  LangJavaScript,
			".py":   LangPython,
			".rs":   LangRust,
			".java": LangJava,
			".c":    LangC,
			".h":    LangC,
			".cpp":  LangCPP,
			".cc":   LangCPP,
			".cxx":  LangCPP,
			".hpp":  LangCPP,
			".md":   LangMarkdown,
			".json": LangJSON,
			".yaml": LangYAML,
			".yml":  LangYAML,
			".html": LangHTML,
			".htm":  LangHTML,
			".css":  LangCSS,
		},
	}
}

// Detect detects language from file extension.
func (d *DefaultLanguageDetector) Detect(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := d.extMap[ext]; ok {
		return lang
	}
	return ""
}

// WalkerConfig configures the file walker behavior.
type WalkerConfig struct {
	// MaxConcurrency limits parallel file processing. Default: 4
	MaxConcurrency int
	// LanguageDetector for custom language detection. Default: DefaultLanguageDetector
	LanguageDetector LanguageDetector
	// FollowSymlinks enables symlink following with cycle detection. Default: false
	FollowSymlinks bool
	// ExistingFiles is a map of path -> (hash, mtime, size) for fast-path optimization.
	// If provided, walker will skip hashing files that haven't changed.
	ExistingFiles map[string]FileRecord
}

// Walker walks a repository and discovers indexable source files.
type Walker struct {
	repoRoot        string
	config          WalkerConfig
	ignoreMatcher   gitignore.IgnoreParser
	langDetector    LanguageDetector
	visitedSymlinks map[string]bool
	symlinkMutex    sync.Mutex
}

// NewWalker creates a new file walker for the given repository root.
func NewWalker(repoRoot string) (*Walker, error) {
	return NewWalkerWithConfig(repoRoot, WalkerConfig{})
}

// NewWalkerWithConfig creates a new file walker with custom configuration.
func NewWalkerWithConfig(repoRoot string, config WalkerConfig) (*Walker, error) {
	// Set defaults
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = 4
	}
	if config.LanguageDetector == nil {
		config.LanguageDetector = NewDefaultLanguageDetector()
	}

	w := &Walker{
		repoRoot:        repoRoot,
		config:          config,
		langDetector:    config.LanguageDetector,
		visitedSymlinks: make(map[string]bool),
	}

	// Collect all ignore patterns
	allPatterns := make([]string, 0, len(DefaultIgnorePatterns)+10)
	allPatterns = append(allPatterns, DefaultIgnorePatterns...)

	// Load gitignore patterns from the repository
	gitignorePatterns := w.loadGitignorePatterns(repoRoot)
	allPatterns = append(allPatterns, gitignorePatterns...)

	// Compile ignore matcher
	w.ignoreMatcher = gitignore.CompileIgnoreLines(allPatterns...)

	return w, nil
}

// loadGitignorePatterns loads patterns from all .gitignore files in the repo.
func (w *Walker) loadGitignorePatterns(repoRoot string) []string {
	var patterns []string

	// Load root .gitignore
	rootGitignore := filepath.Join(repoRoot, ".gitignore")
	if lines, err := readGitignoreLines(rootGitignore); err == nil {
		patterns = append(patterns, lines...)
	}

	// Walk the repo to find nested .gitignore files
	// Note: This is a simple implementation. For production, you might want
	// to handle .gitignore scoping (patterns only apply to their directory and below)
	filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != ".gitignore" || path == rootGitignore {
			return nil
		}
		if lines, err := readGitignoreLines(path); err == nil {
			patterns = append(patterns, lines...)
		}
		return nil
	})

	return patterns
}

// readGitignoreLines reads patterns from a .gitignore file.
func readGitignoreLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// Walk discovers all indexable files in the repository.
func (w *Walker) Walk() ([]FileInfo, error) {
	result := w.WalkWithErrors()
	return result.Files, nil
}

// WalkWithErrors discovers all indexable files and returns detailed error information.
func (w *Walker) WalkWithErrors() WalkResult {
	// Channel for discovered file paths
	pathChan := make(chan string, 100)
	// Channel for processed file info
	resultChan := make(chan FileInfo, 100)
	// Channel for errors
	errorChan := make(chan WalkError, 100)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < w.config.MaxConcurrency; i++ {
		wg.Add(1)
		go w.fileProcessor(ctx, pathChan, resultChan, errorChan, &wg)
	}

	// Start collector goroutine
	var files []FileInfo
	var errors []WalkError
	collectDone := make(chan struct{})
	go func() {
		defer close(collectDone)
		for {
			select {
			case info, ok := <-resultChan:
				if !ok {
					return
				}
				files = append(files, info)
			case err, ok := <-errorChan:
				if !ok {
					return
				}
				errors = append(errors, err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Walk the filesystem and send paths to workers
	walkErr := filepath.WalkDir(w.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			errorChan <- WalkError{Path: path, Err: err}
			return nil // Continue walking
		}

		// Get relative path
		relPath, err := filepath.Rel(w.repoRoot, path)
		if err != nil {
			errorChan <- WalkError{Path: path, Err: err}
			return nil
		}

		// Check if path should be ignored
		if w.ignoreMatcher.MatchesPath(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle symlinks
		if d.Type()&os.ModeSymlink != 0 {
			if !w.config.FollowSymlinks {
				return nil
			}
			// Check for symlink cycles
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				errorChan <- WalkError{Path: path, Err: fmt.Errorf("failed to resolve symlink: %w", err)}
				return nil
			}

			w.symlinkMutex.Lock()
			if w.visitedSymlinks[realPath] {
				w.symlinkMutex.Unlock()
				return nil // Skip cycle
			}
			w.visitedSymlinks[realPath] = true
			w.symlinkMutex.Unlock()
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Detect language
		lang := w.langDetector.Detect(path)
		if lang == "" {
			// Skip files without recognized language
			return nil
		}

		// Send path to workers for processing
		select {
		case pathChan <- path:
		case <-ctx.Done():
			return ctx.Err()
		}

		return nil
	})

	// Close path channel to signal workers
	close(pathChan)

	// Wait for workers to finish
	wg.Wait()

	// Close result channels
	close(resultChan)
	close(errorChan)

	// Wait for collector to finish
	<-collectDone

	// Add walk error if any
	if walkErr != nil && walkErr != context.Canceled {
		errors = append(errors, WalkError{Path: w.repoRoot, Err: walkErr})
	}

	return WalkResult{
		Files:  files,
		Errors: errors,
	}
}

// fileProcessor processes file paths from the channel.
func (w *Walker) fileProcessor(ctx context.Context, pathChan <-chan string, resultChan chan<- FileInfo, errorChan chan<- WalkError, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range pathChan {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Get relative path
		relPath, err := filepath.Rel(w.repoRoot, path)
		if err != nil {
			errorChan <- WalkError{Path: path, Err: err}
			continue
		}

		// Detect language
		lang := w.langDetector.Detect(path)

		// Get file info
		info, err := w.getFileInfo(path, relPath, lang)
		if err != nil {
			errorChan <- WalkError{Path: path, Err: err}
			continue
		}

		resultChan <- *info
	}
}

// getFileInfo reads file metadata and computes hash with fast-path optimization.
func (w *Walker) getFileInfo(fullPath, relPath string, lang Language) (*FileInfo, error) {
	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	size := stat.Size()
	mtime := stat.ModTime().Unix()

	// Fast path: Check if file hasn't changed using existing metadata
	if existing, ok := w.config.ExistingFiles[relPath]; ok {
		if existing.SizeBytes == size && existing.MtimeUnix == mtime {
			// File unchanged - reuse existing hash
			return &FileInfo{
				Path:       relPath,
				Lang:       lang,
				Hash:       existing.Hash,
				SizeBytes:  size,
				MtimeUnix:  mtime,
				NeedsIndex: false, // Will be determined by DB comparison
			}, nil
		}
	}

	// Slow path: File is new or changed, compute hash
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to hash file: %w", err)
	}

	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	return &FileInfo{
		Path:       relPath,
		Lang:       lang,
		Hash:       hash,
		SizeBytes:  size,
		MtimeUnix:  mtime,
		NeedsIndex: true, // Will be determined by DB comparison
	}, nil
}
