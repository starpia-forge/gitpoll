package git

import (
	"context"
	"fmt"
	"os/exec"
)

// CmdRunner abstracts os/exec for testing
type CmdRunner interface {
	Run(ctx context.Context, dir, command string, args ...string) error
}

type defaultCmdRunner struct{}

func (r *defaultCmdRunner) Run(ctx context.Context, dir, command string, args ...string) error {
	// #nosec G204 - command from configuration is trusted
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	return cmd.Run()
}

// Manager handles local git repository operations
type Manager interface {
	Pull(ctx context.Context) error
}

type defaultGitManager struct {
	repoDir string
	branch  string
	runner  CmdRunner
}

// NewManager creates a new instance of a git manager
func NewManager(dir, branch string) Manager {
	return &defaultGitManager{
		repoDir: dir,
		branch:  branch,
		runner:  &defaultCmdRunner{},
	}
}

func (m *defaultGitManager) Pull(ctx context.Context) error {
	// git fetch origin <branch>
	err := m.runner.Run(ctx, m.repoDir, "git", "fetch", "origin", m.branch)
	if err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// git reset --hard origin/<branch>
	err = m.runner.Run(ctx, m.repoDir, "git", "reset", "--hard", "origin/"+m.branch)
	if err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}

	return nil
}
