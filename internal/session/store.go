package session

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store handles persistence of sessions.
type Store struct {
	basePath string
}

// NewStore creates a new session store.
// configPath is typically ~/.dodo
func NewStore(configPath string) *Store {
	return &Store{
		basePath: filepath.Join(configPath, "sessions"),
	}
}

// RepoHash generates a consistent hash for a repository path.
// This is used to scope sessions to a specific project.
func (s *Store) RepoHash(repoPath string) string {
	hash := sha256.Sum256([]byte(filepath.Clean(repoPath)))
	return hex.EncodeToString(hash[:])[:12] // Short hash is sufficient
}

// Save persists a session to disk.
func (s *Store) Save(session *Session) error {
	if session.RepoHash == "" {
		session.RepoHash = s.RepoHash(session.RepoPath)
	}

	dir := filepath.Join(s.basePath, session.RepoHash)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	filename := filepath.Join(dir, fmt.Sprintf("%s.json", session.ID))
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load retrieves a specific session.
func (s *Store) Load(id string, repoPath string) (*Session, error) {
	repoHash := s.RepoHash(repoPath)
	filename := filepath.Join(s.basePath, repoHash, fmt.Sprintf("%s.json", id))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// List returns all sessions for a given repository.
// Sessions are sorted by UpdatedAt (newest first).
func (s *Store) List(repoPath string) ([]SessionMeta, error) {
	repoHash := s.RepoHash(repoPath)
	dir := filepath.Join(s.basePath, repoHash)

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []SessionMeta{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list session directory: %w", err)
	}

	var sessions []SessionMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Optimization: checking only basic metadata without loading full history?
		// For now, we load the file to get accurate title/times.
		// If performance becomes an issue, we can peek or store separate index.
		filepath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filepath)
		if err != nil {
			continue // Skip unreadable files
		}

		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue // Skip invalid files
		}

		sessions = append(sessions, SessionMeta{
			ID:        sess.ID,
			Title:     sess.Title,
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
			Summary:   sess.Summary,
		})
	}

	// Sort by UpdatedAt descending
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}
