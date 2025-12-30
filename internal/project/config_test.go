package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigExists(t *testing.T) {
	// Create a temp directory
	tempDir := t.TempDir()

	// Initially, config should not exist
	if ConfigExists(tempDir) {
		t.Error("ConfigExists should return false when config doesn't exist")
	}

	// Create .dodo directory and config file
	dodoDir := filepath.Join(tempDir, DodoDir)
	if err := os.MkdirAll(dodoDir, 0755); err != nil {
		t.Fatalf("Failed to create .dodo dir: %v", err)
	}

	configPath := filepath.Join(dodoDir, ConfigFile)
	if err := os.WriteFile(configPath, []byte(`{"indexing_enabled": true}`), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Now config should exist
	if !ConfigExists(tempDir) {
		t.Error("ConfigExists should return true when config exists")
	}
}

func TestLoadConfig_NotExists(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := LoadConfig(tempDir)
	if err != nil {
		t.Errorf("LoadConfig should not error when file doesn't exist: %v", err)
	}
	if cfg != nil {
		t.Error("LoadConfig should return nil when file doesn't exist")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	// Save config
	cfg := &ProjectConfig{IndexingEnabled: true}
	if err := SaveConfig(tempDir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify .dodo directory was created
	dodoDir := filepath.Join(tempDir, DodoDir)
	if _, err := os.Stat(dodoDir); os.IsNotExist(err) {
		t.Error(".dodo directory should be created")
	}

	// Load config
	loaded, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadConfig returned nil")
	}
	if loaded.IndexingEnabled != true {
		t.Errorf("Expected IndexingEnabled=true, got %v", loaded.IndexingEnabled)
	}

	// Test with IndexingEnabled = false
	cfg2 := &ProjectConfig{IndexingEnabled: false}
	if err := SaveConfig(tempDir, cfg2); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	loaded2, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded2.IndexingEnabled != false {
		t.Errorf("Expected IndexingEnabled=false, got %v", loaded2.IndexingEnabled)
	}
}

func TestLoadRules_NotExists(t *testing.T) {
	tempDir := t.TempDir()

	rules, err := LoadRules(tempDir)
	if err != nil {
		t.Errorf("LoadRules should not error when file doesn't exist: %v", err)
	}
	if rules != "" {
		t.Errorf("LoadRules should return empty string when file doesn't exist, got: %s", rules)
	}
}

func TestLoadRules(t *testing.T) {
	tempDir := t.TempDir()

	// Create .dodo directory and rules file
	dodoDir := filepath.Join(tempDir, DodoDir)
	if err := os.MkdirAll(dodoDir, 0755); err != nil {
		t.Fatalf("Failed to create .dodo dir: %v", err)
	}

	expectedRules := "Always respond in French.\nNever use emojis."
	rulesPath := filepath.Join(dodoDir, RulesFile)
	if err := os.WriteFile(rulesPath, []byte(expectedRules), 0644); err != nil {
		t.Fatalf("Failed to write rules file: %v", err)
	}

	rules, err := LoadRules(tempDir)
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}
	if rules != expectedRules {
		t.Errorf("Expected rules:\n%s\nGot:\n%s", expectedRules, rules)
	}
}
