package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

func TestStore(t *testing.T) {
	// Create a temporary directory for the store
	tmpDir, err := os.MkdirTemp("", "dodo-session-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	store := NewStore(tmpDir)
	repoPath := "/path/to/my/project"

	// Create a dummy session
	session := &Session{
		ID:        "test-session-id",
		RepoPath:  repoPath,
		Title:     "Test Session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		History: []engine.ChatMessage{
			{Role: engine.RoleUser, Content: "Hello"},
			{Role: engine.RoleAssistant, Content: "Hi there"},
		},
	}

	// Test Save
	if err := store.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file existence
	repoHash := store.RepoHash(repoPath)
	expectedPath := filepath.Join(tmpDir, "sessions", repoHash, "test-session-id.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected session file to exist at %s", expectedPath)
	}

	// Test Load
	loaded, err := store.Load(session.ID, repoPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, loaded.ID)
	}
	if len(loaded.History) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(loaded.History))
	}

	// Test List
	list, err := store.List(repoPath)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 session in list, got %d", len(list))
	}
	if list[0].Title != session.Title {
		t.Errorf("Expected title %s, got %s", session.Title, list[0].Title)
	}
}
