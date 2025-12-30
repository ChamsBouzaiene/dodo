package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DodoDir is the directory name for per-project Dodo configuration
	DodoDir = ".dodo"
	// ConfigFile is the name of the project configuration file
	ConfigFile = "config.json"
	// RulesFile is the name of the custom rules file
	RulesFile = "rules"
)

// ProjectConfig holds per-project configuration settings.
type ProjectConfig struct {
	IndexingEnabled bool `json:"indexing_enabled"`
}

// configPath returns the full path to the project config file.
func configPath(repoRoot string) string {
	return filepath.Join(repoRoot, DodoDir, ConfigFile)
}

// rulesPath returns the full path to the project rules file.
func rulesPath(repoRoot string) string {
	return filepath.Join(repoRoot, DodoDir, RulesFile)
}

// ConfigExists checks if a project configuration file exists.
func ConfigExists(repoRoot string) bool {
	_, err := os.Stat(configPath(repoRoot))
	return !os.IsNotExist(err)
}

// LoadConfig reads the project configuration from disk.
// Returns nil and no error if the config file does not exist.
func LoadConfig(repoRoot string) (*ProjectConfig, error) {
	path := configPath(repoRoot)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project config: %w", err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse project config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig writes the project configuration to disk.
// Creates the .dodo directory if it doesn't exist.
func SaveConfig(repoRoot string, cfg *ProjectConfig) error {
	dodoPath := filepath.Join(repoRoot, DodoDir)

	// Ensure .dodo directory exists
	if err := os.MkdirAll(dodoPath, 0755); err != nil {
		return fmt.Errorf("failed to create .dodo directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	path := configPath(repoRoot)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config: %w", err)
	}

	return nil
}

// LoadRules reads custom agent rules from the .dodo/rules file.
// Returns empty string and no error if the file does not exist.
func LoadRules(repoRoot string) (string, error) {
	path := rulesPath(repoRoot)

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read rules file: %w", err)
	}

	return string(data), nil
}
