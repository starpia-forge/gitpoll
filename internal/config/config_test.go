package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMarshalUnmarshal(t *testing.T) {
	cfg := Config{
		RepoURL:  "http://example.com/repo.git",
		RepoDir:  "/path/to/repo",
		Branch:   "main",
		Command:  "make",
		Interval: 30 * time.Second,
	}

	data, err := Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed Config
	err = Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.RepoURL != cfg.RepoURL {
		t.Errorf("expected %s, got %s", cfg.RepoURL, parsed.RepoURL)
	}
}

func TestMarshalUnmarshalString(t *testing.T) {
	cfg := Config{
		RepoURL: "http://example.com/repo2.git",
		Branch:  "dev",
	}

	dataStr, err := MarshalString(cfg)
	if err != nil {
		t.Fatalf("MarshalString failed: %v", err)
	}

	var parsed Config
	err = UnmarshalString(dataStr, &parsed)
	if err != nil {
		t.Fatalf("UnmarshalString failed: %v", err)
	}

	if parsed.Branch != cfg.Branch {
		t.Errorf("expected %s, got %s", cfg.Branch, parsed.Branch)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	// Creating temp dir to override paths
	tempDir := t.TempDir()

	cfg := &Config{
		RepoURL: "http://example.com/repo.git",
		Branch:  "main",
	}

	path := filepath.Join(tempDir, "config.json")
	err := Save(cfg, path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := loadFromFile(path)
	if err != nil {
		t.Fatalf("loadFromFile failed: %v", err)
	}

	if loaded.RepoURL != "http://example.com/repo.git" {
		t.Errorf("Expected http://example.com/repo.git, got %s", loaded.RepoURL)
	}

	// Test missing file
	missingPath := filepath.Join(tempDir, "missing.json")
	loadedMissing, err := loadFromFile(missingPath)
	if err != nil {
		t.Fatalf("Expected nil error for missing file, got %v", err)
	}
	if loadedMissing != nil {
		t.Errorf("Expected nil config for missing file, got %v", loadedMissing)
	}
}
