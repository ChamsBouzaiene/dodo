package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the user's persistent configuration preferences.
type Config struct {
	LLMProvider  string `json:"llm_provider,omitempty"`  // openai, anthropic, kimi, etc.
	APIKey       string `json:"api_key,omitempty"`       // The API key for the selected provider
	Model        string `json:"model,omitempty"`         // Default model name
	AutoIndex    bool   `json:"auto_index"`              // Whether to auto-index new projects
	BaseURL      string `json:"base_url,omitempty"`      // Optional override for API base URL
	EmbeddingKey string `json:"embedding_key,omitempty"` // Optional separate key for embeddings
}

// Manager handles loading and saving the configuration.
type Manager struct {
	configDir string
}

// NewManager creates a new configuration manager.
func NewManager() (*Manager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user config dir: %w", err)
	}

	dodoConfigDir := filepath.Join(configDir, "dodo")
	return &Manager{
		configDir: dodoConfigDir,
	}, nil
}

// GetConfigPath returns the absolute path to the config.json file.
func (m *Manager) GetConfigPath() string {
	return filepath.Join(m.configDir, "config.json")
}

// Load reads the configuration from disk.
// If the file does not exist, it returns an empty Config and no error.
func (m *Manager) Load() (*Config, error) {
	path := m.GetConfigPath()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config json: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to disk with restricted permissions (0600).
func (m *Manager) Save(cfg *Config) error {
	// Ensure directory exists
	if err := os.MkdirAll(m.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	path := m.GetConfigPath()
	// Write with 0600 permissions (read/write only by owner)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Exists checks if the configuration file has been created.
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.GetConfigPath())
	return !os.IsNotExist(err)
}
