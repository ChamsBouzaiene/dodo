package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/config"
	"github.com/ChamsBouzaiene/dodo/internal/indexer"
)

type runtimeEnv struct {
	RepoRoot     string
	Retrieval    indexer.Retrieval
	WorkspaceCtx *indexer.WorkspaceContext
	manager      *indexer.Manager
}

func (r *runtimeEnv) Close() {
	if r.manager != nil {
		r.manager.Stop()
	}
}

func prepareRuntimeEnv(ctx context.Context, repoFlag string) (*runtimeEnv, error) {
	// Determine repository root
	repoRoot := repoFlag
	if repoRoot == "" {
		// Default to current directory
		var err error
		repoRoot, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Resolve absolute path
	absRepoRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repository path: %w", err)
	}

	// Verify directory exists
	if info, err := os.Stat(absRepoRoot); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("repository path is not a valid directory: %s", absRepoRoot)
	}

	log.Printf("Repository root: %s", absRepoRoot)

	// Detect git info
	gitInfo := indexer.DetectGit(ctx, absRepoRoot)
	if gitInfo.IsGit {
		log.Printf("Git repository detected at: %s", gitInfo.GitRoot)
	}

	// Generate workspace context
	workspaceCtx, err := indexer.GenerateWorkspaceContext(ctx, absRepoRoot, gitInfo)
	if err != nil {
		log.Printf("⚠️  Failed to generate workspace context: %v (continuing without it)", err)
		workspaceCtx = nil
	}

	// Set up indexing manager for semantic search
	var retrieval indexer.Retrieval
	manager, err := setupIndexingManager(ctx, absRepoRoot)
	if err != nil {
		log.Printf("⚠️  Failed to setup indexing manager: %v (semantic search will be disabled)", err)
		retrieval = nil
	} else {
		// Run quick freshness check to ensure index is reasonably up-to-date
		log.Println("🔄 Running quick freshness check...")
		if err := manager.QuickFreshness(ctx, 10); err != nil {
			log.Printf("⚠️  Quick freshness check failed: %v (continuing anyway)", err)
		}
		retrieval = manager // Manager implements Retrieval interface
		log.Println("✅ Semantic search enabled")
	}

	return &runtimeEnv{
		RepoRoot:     absRepoRoot,
		Retrieval:    retrieval,
		WorkspaceCtx: workspaceCtx,
		manager:      manager,
	}, nil
}

// setupIndexingManager creates and configures the indexing manager for semantic search.
func setupIndexingManager(ctx context.Context, repoRoot string) (*indexer.Manager, error) {
	// Generate repo ID from path
	repoID := generateRepoID(repoRoot)

	// Determine DB path (store in repo/.dodo/index.db)
	dbPath := filepath.Join(repoRoot, ".dodo", "index.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create .dodo directory: %w", err)
	}

	// Load User Configuration
	cfgManager, err := config.NewManager()
	var userConfig *config.Config
	if err == nil {
		userConfig, err = cfgManager.Load()
		if err != nil {
			log.Printf("⚠️  Failed to load user config: %v", err)
			userConfig = &config.Config{} // Fallback to empty
		} else {
			log.Printf("User config loaded from: %s", cfgManager.GetConfigPath())
		}
	} else {
		log.Printf("⚠️  Failed to initialize config manager: %v", err)
		userConfig = &config.Config{}
	}

	// Get embedder from environment (OpenAI if available, NoOp otherwise)
	var embedder indexer.Embedder

	// Populate environment variables from config
	// We allow config to override environment if explicitly set.
	// This ensures that if the user runs the Setup Wizard, their choices (saved in config.json)
	// take precedence over potentially stale environment variables in the shell or .env files.
	if userConfig.LLMProvider != "" {
		os.Setenv("LLM_PROVIDER", userConfig.LLMProvider)
	}

	if userConfig.APIKey != "" {
		switch userConfig.LLMProvider {
		case "openai":
			// For OpenAI, we set OPENAI_API_KEY
			os.Setenv("OPENAI_API_KEY", userConfig.APIKey)
			if userConfig.Model != "" {
				os.Setenv("OPENAI_MODEL", userConfig.Model)
			}
			if userConfig.BaseURL != "" {
				os.Setenv("OPENAI_BASE_URL", userConfig.BaseURL)
			}
		case "anthropic":
			os.Setenv("ANTHROPIC_API_KEY", userConfig.APIKey)
			if userConfig.Model != "" {
				os.Setenv("ANTHROPIC_MODEL", userConfig.Model)
			}
		case "kimi":
			os.Setenv("KIMI_API_KEY", userConfig.APIKey)
			if userConfig.Model != "" {
				os.Setenv("KIMI_MODEL", userConfig.Model)
			}
		}
	}

	// For embeddings, we default to OpenAI if we have a key, unless specified otherwise
	embeddingKey := os.Getenv("OPENAI_API_KEY")
	if embeddingKey == "" {
		embeddingKey = userConfig.EmbeddingKey
		if embeddingKey == "" && userConfig.LLMProvider == "openai" {
			// Fallback to main API key if using OpenAI
			embeddingKey = userConfig.APIKey
		}
	}

	if embeddingKey != "" {
		log.Println("📊 Using OpenAI embeddings for semantic search")
		embedder = indexer.NewOpenAIEmbedder(embeddingKey, "text-embedding-3-small", 1536)
	} else {
		log.Println("📊 Using no-op embeddings (set OPENAI_API_KEY for semantic search)")
		embedder = indexer.NewNoOpEmbedder(384)
	}

	// Parse code file boost from environment variable (optional)
	codeFileBoost := 0.0 // 0 means use default
	if boostStr := os.Getenv("DODO_CODE_FILE_BOOST"); boostStr != "" {
		if boost, err := strconv.ParseFloat(boostStr, 64); err == nil {
			// Validate range: 1.0 to 2.0 (reasonable bounds)
			if boost >= 1.0 && boost <= 2.0 {
				codeFileBoost = boost
			} else {
				log.Printf("WARNING: DODO_CODE_FILE_BOOST value %f is outside valid range [1.0, 2.0], using default", boost)
			}
		} else {
			log.Printf("WARNING: Invalid DODO_CODE_FILE_BOOST value '%s', using default: %v", boostStr, err)
		}
	}

	// Create manager config (disable file watching for REPL to reduce overhead)
	config := indexer.ManagerConfig{
		DBPath:             dbPath,
		RepoID:             repoID,
		RepoRoot:           repoRoot,
		Chunker:            indexer.NewDefaultChunker(),
		Embedder:           embedder,
		EnableFileWatcher:  false, // Disable for REPL to reduce overhead
		SafetyScanInterval: 10 * time.Minute,
		WorkerBatchSize:    20,
		WorkerTickInterval: 5 * time.Second,
		CodeFileBoost:      codeFileBoost, // 0 will use default in NewManager
	}

	return indexer.NewManager(ctx, config)
}

// generateRepoID generates a unique ID for a repository based on its path.
func generateRepoID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes for shorter ID
}
