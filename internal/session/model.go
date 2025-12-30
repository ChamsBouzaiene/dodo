package session

import (
	"time"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// Session represents a persistent user session.
type Session struct {
	ID        string               `json:"id"`
	RepoPath  string               `json:"repo_path"`
	RepoHash  string               `json:"repo_hash"` // Used for directory scoping
	Title     string               `json:"title"`
	CreatedAt time.Time            `json:"created_at"`
	UpdatedAt time.Time            `json:"updated_at"`
	History   []engine.ChatMessage `json:"history"`
	Summary   string               `json:"summary,omitempty"` // Context injection for next session
}

// SessionMeta is a lightweight representation for listing in the UI.
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Summary   string    `json:"summary,omitempty"`
}
