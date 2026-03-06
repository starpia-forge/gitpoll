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

// GetLocalConfigPath returns the path to the local configuration file
func GetLocalConfigPath() string {
	return "./gitpoll.config.json"
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

// LoadConfig reads local config and checks if the config is fully valid
func LoadConfig() (*Config, bool, error) {
	localPath := GetLocalConfigPath()

	cfg, err := loadFromFile(localPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil {
		return nil, false, nil
	}

	// Determine if config is completely valid
	isValid := cfg.RepoURL != "" &&
		cfg.RepoDir != "" &&
		cfg.Branch != "" &&
		cfg.Command != "" &&
		cfg.Interval > 0

	return cfg, isValid, nil
}
