package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	RepoURL  string
	RepoDir  string
	Branch   string
	Command  string
	Interval time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		RepoURL: os.Getenv("GITPOLL_REPO_URL"),
		RepoDir: os.Getenv("GITPOLL_REPO_DIR"),
		Branch:  os.Getenv("GITPOLL_BRANCH"),
		Command: os.Getenv("GITPOLL_COMMAND"),
	}

	intervalStr := os.Getenv("GITPOLL_INTERVAL")
	if intervalStr != "" {
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid interval format: %w", err)
		}
		cfg.Interval = interval
	} else {
		cfg.Interval = 1 * time.Minute // Default interval
	}

	// Basic validation (can be expanded)
	if cfg.RepoURL == "" {
		return nil, fmt.Errorf("GITPOLL_REPO_URL environment variable is required")
	}

	return cfg, nil
}
