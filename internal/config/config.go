package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config represents the GitPoll configuration
type Config struct {
	RepoURL  string        `json:"repo_url,omitempty"`
	RepoDir  string        `json:"repo_dir,omitempty"`
	Branch   string        `json:"branch,omitempty"`
	Command  string        `json:"command,omitempty"`
	Interval time.Duration `json:"interval,omitempty"`
}

// Marshal stringifies a value to JSON byte array
func Marshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// Unmarshal parses JSON byte array to a value
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// MarshalString stringifies a value to JSON string
func MarshalString(v any) (string, error) {
	b, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UnmarshalString parses JSON string to a value
func UnmarshalString(data string, v any) error {
	return Unmarshal([]byte(data), v)
}

// GetGlobalConfigPath returns the path to the global configuration file
func GetGlobalConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "gitpoll", "config.json"), nil
}

// GetLocalConfigPath returns the path to the local configuration file
func GetLocalConfigPath() string {
	return "./.gitpoll.json"
}

// Save writes the configuration to the specified path
func Save(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadFromFile reads and parses a JSON config file
func loadFromFile(path string) (*Config, error) {
	// #nosec G304 - configuration paths are determined internally and trusted
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return nil if file does not exist
		}
		return nil, err
	}

	var cfg Config
	if err := Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	return &cfg, nil
}

// Merge overrides the fields of base with the non-empty fields of override
func Merge(base, override *Config) *Config {
	if base == nil {
		base = &Config{}
	}

	merged := *base // Copy base

	if override == nil {
		return &merged
	}

	if override.RepoURL != "" {
		merged.RepoURL = override.RepoURL
	}
	if override.RepoDir != "" {
		merged.RepoDir = override.RepoDir
	}
	if override.Branch != "" {
		merged.Branch = override.Branch
	}
	if override.Command != "" {
		merged.Command = override.Command
	}
	if override.Interval != 0 {
		merged.Interval = override.Interval
	}

	return &merged
}

// LoadConfig reads global and local configs, merges them, and checks if the config is fully valid
func LoadConfig() (*Config, bool, error) {
	globalPath, err := GetGlobalConfigPath()
	if err != nil {
		return nil, false, fmt.Errorf("failed to get global config path: %w", err)
	}
	localPath := GetLocalConfigPath()

	globalCfg, err := loadFromFile(globalPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load global config: %w", err)
	}

	localCfg, err := loadFromFile(localPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load local config: %w", err)
	}

	mergedCfg := Merge(globalCfg, localCfg)

	// Determine if config is completely valid
	isValid := mergedCfg.RepoURL != "" &&
		mergedCfg.RepoDir != "" &&
		mergedCfg.Branch != "" &&
		mergedCfg.Command != "" &&
		mergedCfg.Interval > 0

	return mergedCfg, isValid, nil
}
