package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/config"
	"github.com/ChamsBouzaiene/dodo/internal/engine"
	engineprotocol "github.com/ChamsBouzaiene/dodo/internal/engine/protocol"
	"github.com/ChamsBouzaiene/dodo/internal/factory"
	"github.com/ChamsBouzaiene/dodo/internal/project"
	"github.com/ChamsBouzaiene/dodo/internal/providers"
	"github.com/ChamsBouzaiene/dodo/internal/session"
	"github.com/ChamsBouzaiene/dodo/internal/tools/reasoning"
)

func runStdIOEngine(ctx context.Context, env *runtimeEnv, streaming bool) error {
	log.Println("🔌 Starting engine stdio bridge (--stdio)")
	runner := newStdIORunner(os.Stdin, os.Stdout, env, streaming)
	runner.emitEvent(engineprotocol.NewStatusEvent("", "engine_ready", "stdio protocol ready"))
	return runner.Run(ctx)
}

type stdioRunner struct {
	scanner *bufio.Scanner
	writer  *bufio.Writer
	events  chan engineprotocol.Event
	manager *sessionManager
	config  *config.Manager
}

func newStdIORunner(in io.Reader, out io.Writer, env *runtimeEnv, streaming bool) *stdioRunner {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	writer := bufio.NewWriter(out)
	events := make(chan engineprotocol.Event, 256)

	// Check if config exists
	cfgManager, _ := config.NewManager() // Ignoring error for now, as it just gets the path
	if cfgManager != nil && !cfgManager.Exists() {
		// Emit initial setup event BEFORE creating session manager
		select {
		case events <- engineprotocol.NewSetupRequiredEvent():
		default:
		}
	}

	return &stdioRunner{
		scanner: scanner,
		writer:  writer,
		events:  events,
		manager: newSessionManager(env, streaming, events),
		config:  cfgManager,
	}
}

func (r *stdioRunner) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 1)
	go r.flushEvents(ctx, errCh)

	for {
		select {
		case <-ctx.Done():
			close(r.events)
			return <-errCh
		default:
		}

		if !r.scanner.Scan() {
			break
		}
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		// Run command handler asynchronously to prevent blocking the input loop
		// This is crucial for handling CancelRequest while another command is running
		go func(l string) {
			if err := r.handleLine(ctx, l); err != nil {
				log.Printf("stdio command error: %v", err)
			}
		}(line)
	}

	if err := r.scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		r.emitEvent(engineprotocol.NewErrorEvent("", fmt.Sprintf("stdin error: %v", err), "protocol_error", ""))
		close(r.events)
		return <-errCh
	}

	close(r.events)
	return <-errCh
}

func (r *stdioRunner) flushEvents(ctx context.Context, errCh chan<- error) {
	for {
		select {
		case <-ctx.Done():
			errCh <- nil
			return
		case ev, ok := <-r.events:
			if !ok {
				if err := r.writer.Flush(); err != nil {
					errCh <- err
					return
				}
				errCh <- nil
				return
			}
			if err := r.writeEvent(ev); err != nil {
				errCh <- err
				return
			}
		}
	}
}

func (r *stdioRunner) writeEvent(ev engineprotocol.Event) error {
	payload, err := engineprotocol.MarshalEvent(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if _, err := r.writer.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write event: %w", err)
	}

	return r.writer.Flush()
}

func (r *stdioRunner) emitEvent(ev engineprotocol.Event) {
	select {
	case r.events <- ev:
	default:
		log.Printf("stdio: dropping event %s due to full buffer", ev.GetType())
	}
}

func (r *stdioRunner) handleLine(ctx context.Context, line string) error {
	cmd, err := engineprotocol.DecodeCommand([]byte(line))
	if err != nil {
		r.emitEvent(engineprotocol.NewErrorEvent("", err.Error(), "invalid_command", truncate(line, 256)))
		return err
	}

	switch c := cmd.(type) {
	case engineprotocol.StartSessionCommand:
		session, serr := r.manager.StartSession(ctx, c)
		if serr != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, serr.Error(), "session_error", ""))
			return serr
		}
		r.emitEvent(engineprotocol.NewStatusEvent(session.id, "session_ready", fmt.Sprintf("repo=%s", session.repoRoot)))
		// Emit debug config info to verify what the backend sees
		r.emitEvent(engineprotocol.NewStatusEvent(session.id, "debug_config", fmt.Sprintf("provider=%s openai_model=%s kimi_model=%s", os.Getenv("LLM_PROVIDER"), os.Getenv("OPENAI_MODEL"), os.Getenv("KIMI_MODEL"))))

		// Check for project-level indexing permission
		if !project.ConfigExists(session.repoRoot) {
			r.emitEvent(engineprotocol.NewProjectPermissionRequiredEvent(session.id, session.repoRoot))
		} else if cfg, err := project.LoadConfig(session.repoRoot); err == nil && cfg != nil {
			// Emit rules loaded status if rules file exists
			if rules, _ := project.LoadRules(session.repoRoot); rules != "" {
				r.emitEvent(engineprotocol.NewStatusEvent(session.id, "rules_loaded", "Custom rules active from .dodo/rules"))
			}
		}
		return nil
	case engineprotocol.UserMessageCommand:
		if uerr := r.manager.HandleUserMessage(ctx, c); uerr != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, uerr.Error(), "engine_error", ""))
			return uerr
		}
		return nil
	case engineprotocol.SaveConfigCommand:
		if r.config == nil {
			r.emitEvent(engineprotocol.NewErrorEvent("", "config manager not initialized", "config_error", ""))
			return fmt.Errorf("config manager not initialized")
		}
		// Map map[string]string to config.Config
		cfg := &config.Config{
			LLMProvider:  c.Config["llm_provider"],
			APIKey:       c.Config["api_key"],
			Model:        c.Config["model"],
			BaseURL:      c.Config["base_url"],
			EmbeddingKey: c.Config["embedding_key"],
			AutoIndex:    c.Config["auto_index"] == "true",
		}
		if err := r.config.Save(cfg); err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent("", err.Error(), "config_save_error", ""))
			return err
		}

		// Apply config to environment so next StartSession picks it up
		log.Printf("[DEBUG] Applying config to env. Provider: %s, Model: %s", cfg.LLMProvider, cfg.Model)
		if cfg.LLMProvider != "" {
			os.Setenv("LLM_PROVIDER", cfg.LLMProvider)
		}
		if cfg.APIKey != "" {
			switch cfg.LLMProvider {
			case "openai":
				os.Setenv("OPENAI_API_KEY", cfg.APIKey)
			case "anthropic":
				os.Setenv("ANTHROPIC_API_KEY", cfg.APIKey)
			case "kimi":
				os.Setenv("KIMI_API_KEY", cfg.APIKey)
			case "gemini":
				os.Setenv("GEMINI_API_KEY", cfg.APIKey)
			case "deepseek":
				os.Setenv("DEEPSEEK_API_KEY", cfg.APIKey)
			case "groq":
				os.Setenv("GROQ_API_KEY", cfg.APIKey)
			case "lmstudio":
				os.Setenv("LMSTUDIO_API_KEY", cfg.APIKey)
			case "ollama":
				os.Setenv("OLLAMA_API_KEY", cfg.APIKey)
			case "glm":
				os.Setenv("GLM_API_KEY", cfg.APIKey)
			case "minimax":
				os.Setenv("MINIMAX_API_KEY", cfg.APIKey)
			}
		}
		if cfg.Model != "" {
			switch cfg.LLMProvider {
			case "openai":
				os.Setenv("OPENAI_MODEL", cfg.Model)
			case "anthropic":
				os.Setenv("ANTHROPIC_MODEL", cfg.Model)
			case "kimi":
				os.Setenv("KIMI_MODEL", cfg.Model)
			case "gemini":
				os.Setenv("GEMINI_MODEL", cfg.Model)
			case "deepseek":
				os.Setenv("DEEPSEEK_MODEL", cfg.Model)
			case "groq":
				os.Setenv("GROQ_MODEL", cfg.Model)
			case "lmstudio":
				os.Setenv("LMSTUDIO_MODEL", cfg.Model)
			case "ollama":
				os.Setenv("OLLAMA_MODEL", cfg.Model)
			case "glm":
				os.Setenv("GLM_MODEL", cfg.Model)
			case "minimax":
				os.Setenv("MINIMAX_MODEL", cfg.Model)
			}
		}
		if cfg.BaseURL != "" {
			switch cfg.LLMProvider {
			case "openai":
				os.Setenv("OPENAI_BASE_URL", cfg.BaseURL)
			case "kimi":
				os.Setenv("KIMI_BASE_URL", cfg.BaseURL)
			}
		}
		log.Printf("[DEBUG] Env updated. LLM_PROVIDER=%s, OPENAI_MODEL=%s, KIMI_MODEL=%s", os.Getenv("LLM_PROVIDER"), os.Getenv("OPENAI_MODEL"), os.Getenv("KIMI_MODEL"))

		// Emit success event (maybe just a status?)
		r.emitEvent(engineprotocol.NewStatusEvent("", "setup_complete", "configuration saved"))
		return nil
	case engineprotocol.GetConfigCommand:
		if r.config == nil {
			r.emitEvent(engineprotocol.NewErrorEvent("", "config manager not initialized", "config_error", ""))
			return fmt.Errorf("config manager not initialized")
		}

		cfg, err := r.config.Load()
		if err != nil {
			// If config doesn't exist, return empty config but success
			// OR return error? Better to return empty config event so UI knows it's fresh.
			// But Load() returns error if file missing? Check config manager implementation.
			// Assuming Load returns error on missing file.
			// Let's return empty config map.
			r.emitEvent(engineprotocol.NewConfigLoadedEvent(map[string]string{}))
			return nil
		}

		// Convert Config struct to map
		cfgMap := map[string]string{
			"llm_provider":  cfg.LLMProvider,
			"api_key":       cfg.APIKey,
			"model":         cfg.Model,
			"base_url":      cfg.BaseURL,
			"embedding_key": cfg.EmbeddingKey,
			"auto_index":    fmt.Sprintf("%t", cfg.AutoIndex),
		}

		r.emitEvent(engineprotocol.NewConfigLoadedEvent(cfgMap))
		return nil
	case engineprotocol.ReloadConfigCommand:
		// Reload configuration and swap LLM client in existing session
		if r.config == nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, "config manager not initialized", "config_error", ""))
			return fmt.Errorf("config manager not initialized")
		}

		// Load new config from disk
		cfg, err := r.config.Load()
		if err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("failed to load config: %v", err), "config_error", ""))
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Update environment variables with new config
		if cfg.LLMProvider != "" {
			os.Setenv("LLM_PROVIDER", cfg.LLMProvider)
		}
		if cfg.APIKey != "" {
			switch cfg.LLMProvider {
			case "openai":
				os.Setenv("OPENAI_API_KEY", cfg.APIKey)
				if cfg.Model != "" {
					os.Setenv("OPENAI_MODEL", cfg.Model)
				}
				if cfg.BaseURL != "" {
					os.Setenv("OPENAI_BASE_URL", cfg.BaseURL)
				}
			case "anthropic":
				os.Setenv("ANTHROPIC_API_KEY", cfg.APIKey)
				if cfg.Model != "" {
					os.Setenv("ANTHROPIC_MODEL", cfg.Model)
				}
			case "kimi":
				os.Setenv("KIMI_API_KEY", cfg.APIKey)
				if cfg.Model != "" {
					os.Setenv("KIMI_MODEL", cfg.Model)
				}
			}
		}

		// Create new LLM client with updated config
		ctx := context.Background()
		newLLM, newModelName, err := providers.NewLLMClientFromEnv(ctx)
		if err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("failed to create new LLM client: %v", err), "config_error", ""))
			return fmt.Errorf("failed to create new LLM client: %w", err)
		}

		// Find the session and swap its LLM client
		session, err := r.manager.GetSession(c.SessionID)
		if err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("session not found: %v", err), "session_error", ""))
			return fmt.Errorf("session not found: %w", err)
		}

		// Swap the LLM client in the existing agent
		session.agent.SetLLM(newLLM, newModelName)

		// Emit success event
		r.emitEvent(engineprotocol.NewConfigReloadedEvent(c.SessionID, cfg.LLMProvider, newModelName))
		return nil
	case engineprotocol.CancelRequestCommand:
		// Cancel the currently running task for this session
		session, err := r.manager.GetSession(c.SessionID)
		if err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("session not found: %v", err), "session_error", ""))
			return nil // Don't fail hard on cancel of unknown session
		}

		// Try to cancel
		log.Printf("DEBUG: Received CancelRequest for session %s", c.SessionID)
		cancelled := session.cancel()
		if cancelled {
			log.Printf("DEBUG: Cancel successful for session %s", c.SessionID)
			session.mu.Lock()
			session.lastRunCancelled = true
			session.mu.Unlock()
			session.emit(engineprotocol.NewCancelledEvent(c.SessionID, "Cancelled by user request"))
		} else {
			log.Printf("DEBUG: Cancel failed (no active cancelFunc) for session %s", c.SessionID)
		}
		return nil
	case engineprotocol.ProjectPermissionCommand:
		// Handle project permission response from UI
		session, err := r.manager.GetSession(c.SessionID)
		if err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("session not found: %v", err), "session_error", ""))
			return nil
		}

		// Save the project config
		cfg := &project.ProjectConfig{
			IndexingEnabled: c.IndexingEnabled,
		}
		if err := project.SaveConfig(session.repoRoot, cfg); err != nil {
			r.emitEvent(engineprotocol.NewErrorEvent(c.SessionID, fmt.Sprintf("failed to save project config: %v", err), "config_error", ""))
			return err
		}

		// Emit status to inform UI
		if c.IndexingEnabled {
			r.emitEvent(engineprotocol.NewStatusEvent(c.SessionID, "project_config_saved", "Indexing enabled. Add custom rules in .dodo/rules"))
		} else {
			r.emitEvent(engineprotocol.NewStatusEvent(c.SessionID, "project_config_saved", "Indexing disabled for this project"))
		}
		return nil
	default:
		r.emitEvent(engineprotocol.NewErrorEvent("", "unsupported command", "invalid_command", ""))
		return fmt.Errorf("unsupported command type %T", cmd)
	}
}

type sessionManager struct {
	mu         sync.Mutex
	sessions   map[string]*sessionState
	env        *runtimeEnv
	streaming  bool
	events     chan<- engineprotocol.Event
	store      *session.Store
	summarizer *session.Summarizer
}

func newSessionManager(env *runtimeEnv, streaming bool, sink chan<- engineprotocol.Event) *sessionManager {
	// Check home dir
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("check home dir failed: %v", err)
	}

	var store *session.Store
	if homeDir != "" {
		store = session.NewStore(filepath.Join(homeDir, ".dodo"))
	}

	// Try to load config and apply to env
	if cfgManager, _ := config.NewManager(); cfgManager != nil {
		if cfg, err := cfgManager.Load(); err == nil {
			applyConfigToEnv(cfg)
		}
	}

	// Initialize summarizer if configs are available
	var summarizer *session.Summarizer
	if llm, model, err := providers.NewLLMClientFromEnv(context.Background()); err == nil {
		summarizer = session.NewSummarizer(llm, model)
		log.Printf("Summarizer initialized with model: %s", model)
	} else {
		log.Printf("Summarizer NOT initialized (title generation disabled): %v", err)
	}

	return &sessionManager{
		sessions:   make(map[string]*sessionState),
		env:        env,
		streaming:  streaming,
		events:     sink,
		store:      store,
		summarizer: summarizer,
	}
}

func (m *sessionManager) StartSession(ctx context.Context, cmd engineprotocol.StartSessionCommand) (*sessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repoRoot := cmd.RepoRoot
	if repoRoot == "" {
		repoRoot = m.env.RepoRoot
	} else {
		absRepo, err := filepath.Abs(repoRoot)
		if err != nil {
			return nil, fmt.Errorf("invalid repo_root: %w", err)
		}
		repoRoot = absRepo
	}

	if m.env.RepoRoot != "" && repoRoot != m.env.RepoRoot {
		return nil, fmt.Errorf("repo_root %s does not match running process repo %s", repoRoot, m.env.RepoRoot)
	}

	// Determine Session ID and Load/Create
	var sessionID string
	var loadedSession *session.Session
	var isNew bool

	if cmd.SessionID != "" {
		sessionID = cmd.SessionID
		if _, exists := m.sessions[sessionID]; exists {
			return nil, fmt.Errorf("session already exists: %s", sessionID)
		}
		// Try to load from store
		if m.store != nil {
			if s, err := m.store.Load(sessionID, repoRoot); err == nil {
				loadedSession = s
			}
		}
	} else {
		sessionID = engineprotocol.NewSessionID()
		isNew = true
	}

	// Create Session State
	sessState := &sessionState{
		id:         sessionID,
		repoRoot:   repoRoot,
		eventSink:  m.events,
		streaming:  m.streaming,
		store:      m.store,
		summarizer: m.summarizer,
		createdAt:  time.Now(),
		title:      "Untitled Session",
	}

	if loadedSession != nil {
		sessState.title = loadedSession.Title
		sessState.lastSummary = loadedSession.Summary
		sessState.createdAt = loadedSession.CreatedAt
	}

	hook := newProtocolHook(sessState)
	agent, err := factory.BuildBrainAgent(ctx, repoRoot, m.env.Retrieval, m.env.WorkspaceCtx, m.streaming, true, hook)
	if err != nil {
		return nil, err
	}

	// Populate History or Inject Context
	if loadedSession != nil {
		// Restore history
		for _, msg := range loadedSession.History {
			agent.Append(msg)
		}

		// Emit history to UI so user can see previous conversation summary
		m.events <- engineprotocol.NewSessionHistoryEvent(
			sessionID,
			loadedSession.Title,
			loadedSession.Summary,
			nil, // Don't send full history for display to keep UI clean
		)
	} else if isNew && m.store != nil {
		// Inject previous context for NEW sessions
		if metas, err := m.store.List(repoRoot); err == nil && len(metas) > 0 {
			latest := metas[0] // Sorted by UpdatedAt desc
			if latest.Summary != "" {
				contextMsg := fmt.Sprintf("Previous Session Context: The user was working on '%s'. The last known state was: %s", latest.Title, latest.Summary)
				agent.Append(engine.ChatMessage{
					Role:    engine.RoleSystem,
					Content: contextMsg,
				})
			}
		}
	}

	sessState.agent = agent
	m.sessions[sessionID] = sessState

	// Save session immediately on creation so it persists even if interaction is interrupted
	if isNew && m.store != nil {
		model := &session.Session{
			ID:        sessionID,
			RepoPath:  repoRoot,
			Title:     sessState.title,
			Summary:   "",
			CreatedAt: sessState.createdAt,
			UpdatedAt: time.Now(),
			History:   []engine.ChatMessage{},
		}
		if err := m.store.Save(model); err != nil {
			log.Printf("failed to save initial session %s: %v", sessionID, err)
		} else {
			log.Printf("Initial session saved: %s", sessionID)
		}
	}

	return sessState, nil
}

// GetSession retrieves an existing session by ID.
func (m *sessionManager) GetSession(sessionID string) (*sessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return session, nil
}

func (m *sessionManager) HandleUserMessage(ctx context.Context, cmd engineprotocol.UserMessageCommand) error {
	session, err := m.session(cmd.SessionID)
	if err != nil {
		return err
	}

	if !session.beginRun() {
		return fmt.Errorf("session %s is already processing a request", cmd.SessionID)
	}
	defer session.endRun()

	// Create a cancellable context for this run
	runCtx, cancel := context.WithCancel(ctx)
	session.setCancelFunc(cancel)
	log.Printf("DEBUG: Created cancelFunc for session %s", cmd.SessionID)
	defer func() {
		log.Printf("DEBUG: Clearing cancelFunc for session %s", cmd.SessionID)
		cancel()
		session.setCancelFunc(nil) // Clear the cancel func when done
	}()

	session.emit(engineprotocol.NewStatusEvent(session.id, "message_received", truncate(cmd.Message, 120)))

	message := cmd.Message
	if session.lastRunCancelled {
		log.Printf("DEBUG: Injecting cancellation context for session %s", cmd.SessionID)
		message = "[System Note: The user cancelled the previous task. Please inquire if they want to resume or start something new.] " + message
		session.lastRunCancelled = false
	}

	if err := session.agent.Run(runCtx, message); err != nil {
		// Check if this was a cancellation - that's not a real error
		if runCtx.Err() == context.Canceled {
			// Already handled via the cancel handler, just return nil
			return nil
		}
		session.emit(engineprotocol.NewErrorEvent(session.id, err.Error(), "engine_error", ""))
		return err
	}

	return nil
}

func (m *sessionManager) session(id string) (*sessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("unknown session_id: %s", id)
	}
	return session, nil
}

type sessionState struct {
	id         string
	repoRoot   string
	agent      *engine.Agent
	eventSink  chan<- engineprotocol.Event
	streaming  bool
	store      *session.Store
	summarizer *session.Summarizer

	mu          sync.Mutex
	running     bool
	cancelFunc  context.CancelFunc
	lastSummary string
	files       []string
	title       string
	createdAt   time.Time

	// Track if the last run was cancelled to inject context
	lastRunCancelled bool
}

func (s *sessionState) emit(ev engineprotocol.Event) {
	select {
	case s.eventSink <- ev:
	default:
		log.Printf("stdio session %s: dropping event %s due to full buffer", s.id, ev.GetType())
	}
}

func (s *sessionState) beginRun() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *sessionState) endRun() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}

func (s *sessionState) recordSummary(summary string, files []string) {
	// 1. Get History (safe sync access)
	var history []engine.ChatMessage
	if s.agent != nil && s.agent.LastState() != nil {
		history = s.agent.LastState().History
	}

	// 2. Generate Title if needed
	s.mu.Lock()
	needsTitle := s.title == "Untitled Session"
	summarizer := s.summarizer
	s.mu.Unlock()

	var newTitle string
	if needsTitle && summarizer != nil && len(history) > 2 {
		if t, err := summarizer.GenerateTitle(context.Background(), history); err == nil {
			newTitle = t
		} else {
			log.Printf("failed to generate title: %v", err)
		}
	}

	// Fallback: use first user message as title if summarizer unavailable
	if needsTitle && newTitle == "" && len(history) > 0 {
		for _, msg := range history {
			if msg.Role == engine.RoleUser && msg.Content != "" {
				// Take first 50 chars of first user message
				title := msg.Content
				if len(title) > 50 {
					title = title[:50] + "..."
				}
				newTitle = title
				break
			}
		}
	}

	// 3. Update State and Save
	s.mu.Lock()
	defer s.mu.Unlock()

	if newTitle != "" {
		s.title = newTitle
	}
	s.lastSummary = summary
	s.files = append([]string(nil), files...)

	// Create model snapshot while holding lock
	model := &session.Session{
		ID:        s.id,
		RepoPath:  s.repoRoot,
		Title:     s.title,
		Summary:   s.lastSummary,
		CreatedAt: s.createdAt,
		UpdatedAt: time.Now(),
		History:   history,
	}
	store := s.store

	// Persist (we can do this under lock to ensure sequential saves, or move out if blocking is issue)
	// For now, doing it under lock prevents race conditions on file writes if recordSummary called concurrently (unlikely)
	if store != nil {
		if err := store.Save(model); err != nil {
			log.Printf("failed to save session %s: %v", s.id, err)
		}
	}
}

// cancel attempts to cancel the current running task.
// Returns true if there was a task running and it was cancelled.
func (s *sessionState) cancel() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running && s.cancelFunc != nil {
		s.cancelFunc()
		// Note: endRun() will be called by the goroutine when it exits
		return true
	}
	return false
}

// setCancelFunc stores the cancel function for the current run.
func (s *sessionState) setCancelFunc(cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelFunc = cancel
}

func (s *sessionState) snapshot() (string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files := append([]string(nil), s.files...)
	return s.lastSummary, files
}

func newProtocolHook(session *sessionState) *protocolHook {
	return &protocolHook{
		session:     session,
		invocations: make(map[string]time.Time),
		activityIDs: make(map[string]string),
	}
}

type protocolHook struct {
	engine.NopHook
	session       *sessionState
	invocations   map[string]time.Time // invocation_id -> start time
	activityIDs   map[string]string    // invocation_id -> activity_id (to match started/completed events)
	invocationsMu sync.Mutex

	// Assistant message buffer (simple buffer, no chunking)
	assistantBuffer strings.Builder
}

func (h *protocolHook) getInvocationID(call engine.ToolCall) string {
	// Use tool call ID if available, otherwise generate one
	if call.ID != "" {
		return call.ID
	}
	return fmt.Sprintf("%s-%d", call.Name, time.Now().UnixNano())
}

func (h *protocolHook) OnStepStart(ctx context.Context, st *engine.State) {
	// Reset assistant buffer for new step
	h.assistantBuffer.Reset()

	detail := fmt.Sprintf("step=%d phase=%s", st.Step+1, st.Phase)
	h.session.emit(engineprotocol.NewStatusEvent(h.session.id, "step_start", detail))
}

func (h *protocolHook) OnBeforeLLM(ctx context.Context, st *engine.State, messages []engine.ChatMessage, schemas []engine.ToolSchema) {
	h.session.emit(engineprotocol.NewStatusEvent(h.session.id, "thinking", fmt.Sprintf("%d messages", len(messages))))
}

// OnStreamDelta buffers text deltas silently. The full message will be emitted
// once when the LLM response completes in OnAfterLLM.
func (h *protocolHook) OnStreamDelta(ctx context.Context, st *engine.State, delta string) {
	if strings.TrimSpace(delta) == "" {
		return
	}
	// Simply buffer the delta - no events emitted during streaming
	h.assistantBuffer.WriteString(delta)
}

func (h *protocolHook) OnAfterLLM(ctx context.Context, st *engine.State, resp engine.LLMResponse) {
	// Emit token usage based on the prompt size of this call
	if st.Budget.HardLimit > 0 {
		h.session.emit(engineprotocol.NewTokenUsageEvent(h.session.id, resp.Usage.Prompt, st.Budget.HardLimit, st.Totals.Total))
	}

	// Emit full buffered assistant message as single event when complete
	content := h.assistantBuffer.String()
	if content != "" {
		trimmed := strings.TrimSpace(content)
		if trimmed != "" {
			h.session.emit(engineprotocol.NewAssistantTextEvent(h.session.id, trimmed, "assistant", true))
		}
		// Clear buffer for next message
		h.assistantBuffer.Reset()
	}

	// Also handle non-streaming mode (fallback)
	if h.session.streaming {
		return
	}
	nonStreamingContent := strings.TrimSpace(resp.Assistant.Content)
	if nonStreamingContent != "" {
		h.session.emit(engineprotocol.NewAssistantTextEvent(h.session.id, nonStreamingContent, "assistant", true))
	}

	// Create a reasoning step if there's meaningful content
	// This represents the agent's thinking/reasoning before taking action
	if content != "" && len(content) > 10 {
		reasoningID := fmt.Sprintf("reasoning-step-%d", st.Step)
		startTime := time.Now()

		// Truncate very long reasoning for display
		displayContent := content
		if len(displayContent) > 500 {
			displayContent = displayContent[:500] + "..."
		}

		h.session.emit(engineprotocol.NewActivityEventWithTiming(
			h.session.id,
			reasoningID,
			reasoningID,
			"reasoning",
			"",
			"",
			"completed",
			startTime.Format(time.RFC3339Nano),
			startTime.Format(time.RFC3339Nano),
			0,
			"",
			map[string]any{"content": displayContent, "full_content": content, "step": st.Step, "has_tool_calls": len(resp.ToolCalls) > 0},
			nil,
		))
	}
}

func (h *protocolHook) OnToolCall(ctx context.Context, st *engine.State, call engine.ToolCall) {
	h.session.emit(engineprotocol.NewToolEvent(h.session.id, call.Name, "start", nil, ""))

	// Generate stable invocation ID
	invocationID := h.getInvocationID(call)
	startTime := time.Now()

	// Track start time and generate activity ID
	h.invocationsMu.Lock()
	h.invocations[invocationID] = startTime
	// Generate and store activity ID for this invocation
	activityID := fmt.Sprintf("%s-%d", call.Name, time.Now().UnixNano())
	h.activityIDs[invocationID] = activityID
	h.invocationsMu.Unlock()

	// Extract metadata and send activity event
	metadata, target := extractToolMetadata(call, call.Name)

	activityType := "tool"
	if call.Name == "think" {
		activityType = "reasoning"
	}

	// Extract command for execution tools
	command := ""
	if call.Name == "run_cmd" || call.Name == "run_tests" || call.Name == "run_build" {
		if cmd, ok := metadata["command"].(string); ok {
			command = cmd
		}
	}

	h.session.emit(engineprotocol.NewActivityEventWithTiming(
		h.session.id,
		activityID,
		invocationID,
		activityType,
		call.Name,
		target,
		"started",
		startTime.Format(time.RFC3339Nano),
		"",
		0,
		command,
		metadata,
		nil,
	))
}

func (h *protocolHook) OnToolResult(ctx context.Context, st *engine.State, call engine.ToolCall, result string, err error) {
	success := err == nil
	h.session.emit(engineprotocol.NewToolEvent(h.session.id, call.Name, "end", &success, truncate(result, 500)))

	// Get invocation ID and start time
	invocationID := h.getInvocationID(call)
	h.invocationsMu.Lock()
	startTime, hasStartTime := h.invocations[invocationID]
	// Retrieve the activity ID that was used for the "started" event
	activityID, hasActivityID := h.activityIDs[invocationID]
	delete(h.invocations, invocationID) // Clean up
	delete(h.activityIDs, invocationID) // Clean up
	h.invocationsMu.Unlock()

	// If we don't have a stored activity ID, generate a new one (fallback)
	if !hasActivityID {
		activityID = fmt.Sprintf("%s-%d", call.Name, time.Now().UnixNano())
	}

	finishTime := time.Now()
	var durationMs int64
	var startedAt, finishedAt string
	if hasStartTime {
		duration := finishTime.Sub(startTime)
		durationMs = duration.Milliseconds()
		startedAt = startTime.Format(time.RFC3339Nano)
		finishedAt = finishTime.Format(time.RFC3339Nano)
	}

	// Send activity completion event (activityID already set above)
	metadata, target := extractToolMetadata(call, call.Name)

	// Add result metadata
	if success {
		metadata["result_size"] = len(result)
	}

	status := "completed"
	if !success {
		status = "failed"
		if err != nil {
			metadata["error"] = err.Error()
		}
		// Also include result as error context if it contains useful error info
		if result != "" && len(result) < 1000 {
			// Try to extract error from result if it's not JSON or contains error info
			if !strings.HasPrefix(strings.TrimSpace(result), "{") {
				metadata["error_result"] = result
			}
		}
	}

	// Extract code changes for editing tools
	var codeChange *engineprotocol.CodeChange
	if call.Name == "search_replace" || call.Name == "propose_diff" || call.Name == "write" {
		codeChange = extractCodeChange(call, result)
	}

	activityType := "tool"
	if call.Name == "think" {
		activityType = "reasoning"
	} else if codeChange != nil {
		activityType = "edit"
	}

	// Extract command for execution tools
	command := ""
	if call.Name == "run_cmd" || call.Name == "run_tests" || call.Name == "run_build" {
		if cmd, ok := metadata["command"].(string); ok {
			command = cmd
		}
		// Parse execution tool results and emit log lines
		h.handleExecutionToolResult(invocationID, call.Name, command, result, success, durationMs)

		// Also extract stdout/stderr for display in timeline using standard ExecutionResult
		var execResult engine.ExecutionResult
		if err := json.Unmarshal([]byte(result), &execResult); err == nil {
			if execResult.Stdout != "" {
				metadata["stdout"] = execResult.Stdout
			}
			if execResult.Stderr != "" {
				metadata["stderr"] = execResult.Stderr
			}
			metadata["exit_code"] = execResult.ExitCode
		}
	}

	// Extract file content for read_file tool
	if call.Name == "read_file" && success {
		// Store the file content, but truncate if too large (max 50KB for UI display)
		const maxFileSize = 50 * 1024
		if len(result) > maxFileSize {
			metadata["output"] = result[:maxFileSize] + fmt.Sprintf("\n... (truncated, %d bytes total)", len(result))
		} else {
			metadata["output"] = result
		}
	}

	// Extract plan content for plan tool
	if call.Name == "plan" && success {
		metadata["plan_content"] = result
		// Also extract structured plan data if available
		if taskSummary, ok := call.Args["task_summary"].(string); ok {
			metadata["plan_summary"] = taskSummary
		}
		if stepsRaw, ok := call.Args["steps"].([]interface{}); ok {
			metadata["plan_steps"] = stepsRaw
		}
		if targetAreas, ok := call.Args["target_areas"].([]interface{}); ok {
			metadata["plan_target_areas"] = targetAreas
		}
		if risks, ok := call.Args["risks"].([]interface{}); ok {
			metadata["plan_risks"] = risks
		}
	}

	// Extract project plan content
	if call.Name == "project_plan" && success {
		var content string
		var source string

		mode, _ := call.Args["mode"].(string)
		if mode == "read" {
			content = result
			source = "read"
		} else if mode == "update" {
			if c, ok := call.Args["content"].(string); ok {
				content = c
				source = "update"
			}
		}

		if content != "" {
			h.session.emit(engineprotocol.NewProjectPlanEvent(h.session.id, content, source))
		}
	}

	h.session.emit(engineprotocol.NewActivityEventWithTiming(
		h.session.id,
		activityID,
		invocationID,
		activityType,
		call.Name,
		target,
		status,
		startedAt,
		finishedAt,
		durationMs,
		command,
		metadata,
		codeChange,
	))

	// Handle RESPOND tool result (only on success)
	if call.Name == "respond" && success {
		h.handleRespondResult(result)
	}
	// On failure, error is already captured in metadata above
}

func (h *protocolHook) OnRetryAttempt(ctx context.Context, st *engine.State, attempt, maxAttempts int, delay time.Duration, err error) {
	detail := fmt.Sprintf("attempt=%d/%d delay=%s error=%v", attempt, maxAttempts, delay, err)
	h.session.emit(engineprotocol.NewStatusEvent(h.session.id, "retry", detail))
}

func (h *protocolHook) OnRetryExhausted(ctx context.Context, st *engine.State, err error) {
	h.session.emit(engineprotocol.NewErrorEvent(h.session.id, fmt.Sprintf("retries exhausted: %v", err), "retry_exhausted", ""))
}

func (h *protocolHook) OnBudgetExceeded(ctx context.Context, st *engine.State, tokens, softLimit, hardLimit int) {
	detail := fmt.Sprintf("tokens=%d soft=%d hard=%d", tokens, softLimit, hardLimit)
	h.session.emit(engineprotocol.NewStatusEvent(h.session.id, "budget_exceeded", detail))
}

func (h *protocolHook) OnSoftCapReached(ctx context.Context, st *engine.State, err error) {
	h.session.emit(engineprotocol.NewErrorEvent(h.session.id, err.Error(), "soft_cap", ""))
}

func (h *protocolHook) OnToolOutput(ctx context.Context, st *engine.State, toolName string, output string) {
	// Find the most recent invocation for this tool
	// This is a fallback - ideally we'd have the invocation ID passed in
	h.invocationsMu.Lock()
	var invocationID string
	// Use a simple heuristic: find the most recent invocation
	// In practice, the tool should pass invocation ID through context
	for id := range h.invocations {
		invocationID = id
		break // Just use first one found
	}
	h.invocationsMu.Unlock()

	if invocationID != "" {
		h.session.emit(engineprotocol.NewToolOutputEvent(
			h.session.id,
			invocationID,
			toolName,
			output,
			false,
			"stdout",
		))
	}
}

func (h *protocolHook) OnDone(ctx context.Context, st *engine.State) {
	summary, files := h.session.snapshot()
	h.session.emit(engineprotocol.NewStatusEvent(h.session.id, "done", "session complete"))
	h.session.emit(engineprotocol.NewDoneEvent(h.session.id, summary, files))
}

func (h *protocolHook) handleRespondResult(result string) {
	trimmed := strings.TrimSpace(result)
	if !strings.HasPrefix(trimmed, "{") {
		// If result doesn't start with JSON, it might be an error message
		// This will be handled by the error metadata in OnToolResult
		return
	}

	var resp reasoning.RespondResult
	if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
		// JSON parsing failed - this indicates a RESPOND tool failure
		// The error will be captured in OnToolResult's metadata
		return
	}

	h.session.recordSummary(resp.Summary, resp.FilesChanged)
	if len(resp.FilesChanged) > 0 {
		h.session.emit(engineprotocol.NewFilesChangedEvent(h.session.id, resp.FilesChanged))
	}
	if strings.TrimSpace(resp.Summary) != "" {
		h.session.emit(engineprotocol.NewAssistantTextEvent(h.session.id, resp.Summary, "respond.summary", true))
	}
}

func (h *protocolHook) OnBudgetCompression(ctx context.Context, st *engine.State, beforeTokens, afterTokens int, strategy engine.CompressionStrategy) {
	strategyStr := "unknown"
	switch strategy {
	case engine.CompressionTruncate:
		strategyStr = "truncate"
	case engine.CompressionSummarize:
		strategyStr = "summarize"
	case engine.CompressionAggressiveSummarize:
		strategyStr = "aggressive_summarize"
	case engine.CompressionRemove:
		strategyStr = "remove"
	}
	h.session.emit(engineprotocol.NewContextEvent(h.session.id, "compress", strategyStr, beforeTokens, afterTokens))
	// Also update token usage after compression
	if st.Budget.HardLimit > 0 {
		h.session.emit(engineprotocol.NewTokenUsageEvent(h.session.id, afterTokens, st.Budget.HardLimit, st.Totals.Total))
	}
}

func (h *protocolHook) OnSummarize(ctx context.Context, st *engine.State, before, after []engine.ChatMessage) {
	// Estimate token reduction (rough count or just event)
	h.session.emit(engineprotocol.NewContextEvent(h.session.id, "summarize", "History summarization", len(before), len(after)))
}

// handleExecutionToolResult parses execution tool JSON results and emits log lines.
// It uses the standard ExecutionResult struct to decouple from tool implementation details.
func (h *protocolHook) handleExecutionToolResult(invocationID, toolName, command string, result string, success bool, durationMs int64) {
	// Parse result JSON using standard ExecutionResult struct
	var execResult engine.ExecutionResult
	if err := json.Unmarshal([]byte(result), &execResult); err != nil {
		// If parsing fails, silently return (tool may return non-standard format)
		return
	}

	// Extract command if not already provided
	if command == "" {
		command = execResult.Cmd
	}

	// Emit command start line
	if command != "" {
		h.session.emit(engineprotocol.NewToolOutputEvent(
			h.session.id,
			invocationID,
			toolName,
			command,
			false,
			"command",
		))
	}

	// Emit stdout lines
	if execResult.Stdout != "" {
		lines := strings.Split(execResult.Stdout, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				h.session.emit(engineprotocol.NewToolOutputEvent(
					h.session.id,
					invocationID,
					toolName,
					line,
					false,
					"stdout",
				))
			}
		}
	}

	// Emit stderr lines
	if execResult.Stderr != "" {
		lines := strings.Split(execResult.Stderr, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				h.session.emit(engineprotocol.NewToolOutputEvent(
					h.session.id,
					invocationID,
					toolName,
					line,
					true,
					"stderr",
				))
			}
		}
	}

	// Emit completion line
	durationSec := float64(durationMs) / 1000.0
	statusText := "success"
	if !success || execResult.ExitCode != 0 {
		statusText = fmt.Sprintf("code %d", execResult.ExitCode)
	}
	completionMsg := fmt.Sprintf("[EXEC] Command exited with %s (%.2fs)", statusText, durationSec)
	h.session.emit(engineprotocol.NewToolOutputEvent(
		h.session.id,
		invocationID,
		toolName,
		completionMsg,
		!success,
		"complete",
	))
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}
	return s[:limit-3] + "..."
}

// extractToolMetadata extracts concise metadata from tool arguments
func extractToolMetadata(call engine.ToolCall, toolName string) (map[string]any, string) {
	metadata := make(map[string]any)
	target := ""

	switch toolName {
	case "read_file":
		if path, ok := call.Args["path"].(string); ok {
			target = path
			metadata["path"] = path
		}
	case "read_span":
		if path, ok := call.Args["path"].(string); ok {
			target = path
			metadata["path"] = path
		}
		if start, ok := call.Args["start"].(float64); ok {
			metadata["start_line"] = int(start)
		}
		if end, ok := call.Args["end"].(float64); ok {
			metadata["end_line"] = int(end)
		}
	case "grep":
		if pattern, ok := call.Args["pattern"].(string); ok {
			target = pattern
			metadata["pattern"] = pattern
		}
		if path, ok := call.Args["path"].(string); ok {
			metadata["path"] = path
		}
	case "search_replace", "propose_diff":
		if path, ok := call.Args["file_path"].(string); ok {
			target = path
			metadata["file"] = path
		}
	case "write", "write_file":
		if path, ok := call.Args["path"].(string); ok {
			target = path
			metadata["file"] = path
		} else if path, ok := call.Args["file_path"].(string); ok {
			target = path
			metadata["file"] = path
		}
	case "codebase_search":
		if query, ok := call.Args["query"].(string); ok {
			target = query
			metadata["query"] = truncate(query, 50)
		}
	case "run_cmd":
		// Build command string from cmd and args
		cmd := ""
		if c, ok := call.Args["cmd"].(string); ok {
			cmd = c
		}
		argsStr := ""
		if args, ok := call.Args["args"].(string); ok {
			argsStr = args
		}
		if cmd != "" {
			command := cmd
			if argsStr != "" {
				command = cmd + " " + argsStr
			}
			target = command
			metadata["command"] = command
		}
	case "run_tests", "run_build":
		// These tools don't take arguments, but we'll show them in the timeline
		target = toolName
		metadata["command"] = toolName
	case "run_terminal_cmd":
		if command, ok := call.Args["command"].(string); ok {
			target = command
			metadata["command"] = truncate(command, 50)
		}
	case "think":
		// For think tool, store the full reasoning (will be displayed in timeline)
		if reasoning, ok := call.Args["reasoning"].(string); ok {
			target = "thinking"
			// Store full reasoning for display, but also generate summary
			metadata["reasoning"] = reasoning
			summary := extractThinkSummary(reasoning)
			metadata["summary"] = summary
		} else if reason, ok := call.Args["reason"].(string); ok {
			target = "thinking"
			metadata["reasoning"] = reason
			summary := extractThinkSummary(reason)
			metadata["summary"] = summary
		}
	case "delete_file":
		if path, ok := call.Args["path"].(string); ok {
			target = path
			metadata["path"] = path
		}
	case "list_files":
		if path, ok := call.Args["path"].(string); ok {
			target = path
			metadata["path"] = path
		}
	}

	return metadata, target
}

// extractThinkSummary extracts only a safe, high-level summary from think tool reasoning
func extractThinkSummary(reasoning string) string {
	// Never expose raw reasoning - only high-level summaries
	reasoning = strings.TrimSpace(reasoning)
	if reasoning == "" {
		return "Planning next action"
	}

	// Extract first sentence (up to 100 chars) or generate safe summary
	firstSentence := reasoning
	if len(reasoning) > 100 {
		firstSentence = reasoning[:100]
		// Try to end at sentence boundary
		if idx := strings.LastIndex(firstSentence, "."); idx > 50 {
			firstSentence = firstSentence[:idx+1]
		}
	}

	// Generate safe summary based on keywords
	lower := strings.ToLower(firstSentence)
	if strings.Contains(lower, "search") || strings.Contains(lower, "find") {
		return "Searching codebase"
	}
	if strings.Contains(lower, "edit") || strings.Contains(lower, "modify") || strings.Contains(lower, "change") {
		return "Planning code changes"
	}
	if strings.Contains(lower, "read") || strings.Contains(lower, "examine") {
		return "Reviewing code"
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "run") {
		return "Planning test execution"
	}

	// Default safe summary
	return "Planning next action"
}

// extractCodeChange extracts code diff from editing tool results
func extractCodeChange(call engine.ToolCall, result string) *engineprotocol.CodeChange {
	// Try to parse result as JSON
	var resultData map[string]any
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		return nil
	}

	filePath := ""
	if path, ok := call.Args["file_path"].(string); ok {
		filePath = path
	}

	if filePath == "" {
		return nil
	}

	// For search_replace, extract old_string and new_string
	if call.Name == "search_replace" {
		oldStr, _ := call.Args["old_string"].(string)
		newStr, _ := call.Args["new_string"].(string)

		if oldStr != "" || newStr != "" {
			return &engineprotocol.CodeChange{
				File:   filePath,
				Before: truncate(oldStr, 500),
				After:  truncate(newStr, 500),
			}
		}
	}

	// For write tool, show that file was written
	if call.Name == "write" {
		if contents, ok := call.Args["contents"].(string); ok {
			return &engineprotocol.CodeChange{
				File:   filePath,
				Before: "",
				After:  truncate(contents, 500),
			}
		}
	}

	return nil
}
