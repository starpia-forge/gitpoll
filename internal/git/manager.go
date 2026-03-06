package git

import (
	"context"
	"fmt"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"

	repoConfig "repo-gitpoll/internal/config"
)

// RepositoryOpener defines a function to open a git repository.
// This allows for mocking in tests by substituting gogit.PlainOpen with an in-memory equivalent.
type RepositoryOpener func(path string) (*gogit.Repository, error)

// Manager handles local git repository operations
type Manager interface {
	Pull(ctx context.Context) error
}

type defaultGitManager struct {
	repoDir string
	branch  string
	opener  RepositoryOpener
}

// NewManager creates a new instance of a git manager
func NewManager(cfg *repoConfig.Config) Manager {
	return &defaultGitManager{
		repoDir: cfg.RepoDir,
		branch:  cfg.Branch,
		opener:  func(path string) (*gogit.Repository, error) { return gogit.PlainOpen(path) },
	}
}

func (m *defaultGitManager) Pull(ctx context.Context) error {
	r, err := m.opener(m.repoDir)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	// git fetch origin <branch>
	err = r.FetchContext(ctx, &gogit.FetchOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/heads/%[1]s:refs/remotes/origin/%[1]s", m.branch)),
		},
		Force: true,
	})
	if err != nil && err != gogit.NoErrAlreadyUpToDate {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Resolve the reference to origin/<branch>
	refName := plumbing.NewRemoteReferenceName("origin", m.branch)
	ref, err := r.Reference(refName, true)
	if err != nil {
		return fmt.Errorf("failed to get reference to origin/%s: %w", m.branch, err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// git reset --hard origin/<branch>
	err = w.Reset(&gogit.ResetOptions{
		Commit: ref.Hash(),
		Mode:   gogit.HardReset,
	})
	if err != nil {
		return fmt.Errorf("git reset failed: %w", err)
	}

	return nil
}
