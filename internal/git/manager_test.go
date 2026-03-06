package git

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"

	repoConfig "repo-gitpoll/internal/config"
)

func TestManager_Pull_Failure_Open(t *testing.T) {
	cfg := &repoConfig.Config{RepoDir: "/tmp/repo", Branch: "main"}
	m := NewManager(cfg)

	m.(*defaultGitManager).opener = func(path string) (*gogit.Repository, error) {
		return nil, errors.New("mock open error")
	}

	err := m.Pull(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestManager_Pull_Success(t *testing.T) {
	remotePath, err := os.MkdirTemp("", "git-remote-*")
	if err != nil {
		t.Fatalf("Failed to create remote temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(remotePath) }()

	localPath, err := os.MkdirTemp("", "git-local-*")
	if err != nil {
		t.Fatalf("Failed to create local temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(localPath) }()

	remoteRepo, err := gogit.PlainInit(remotePath, false)
	if err != nil {
		t.Fatalf("Failed to init remote repo: %v", err)
	}

	wRemote, err := remoteRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get remote worktree: %v", err)
	}

	f, err := os.Create(remotePath + "/test.txt")
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if _, err := f.Write([]byte("initial content")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	if _, err := wRemote.Add("test.txt"); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	_, err = wRemote.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit to remote: %v", err)
	}

	headRef, err := remoteRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get remote head: %v", err)
	}
	branchName := headRef.Name().Short()

	localRepo, err := gogit.PlainInit(localPath, false)
	if err != nil {
		t.Fatalf("Failed to init local repo: %v", err)
	}

	// Create an initial commit in local to establish HEAD, required for Reset
	wLocal, err := localRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get local worktree: %v", err)
	}
	fLocal, err := os.Create(localPath + "/dummy.txt")
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	if _, err := fLocal.Write([]byte("dummy content")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := fLocal.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
	if _, err := wLocal.Add("dummy.txt"); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	_, err = wLocal.Commit("Dummy commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit dummy to local: %v", err)
	}

	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{remotePath},
	})
	if err != nil {
		t.Fatalf("Failed to create remote in local repo: %v", err)
	}

	cfg := &repoConfig.Config{RepoDir: localPath, Branch: branchName}
	m := NewManager(cfg)

	err = m.Pull(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	content, err := os.ReadFile(localPath + "/test.txt")
	if err != nil {
		t.Fatalf("Failed to read pulled file: %v", err)
	}

	if string(content) != "initial content" {
		t.Fatalf("Expected 'initial content', got '%s'", string(content))
	}
}

func TestManager_Pull_Failure_Fetch(t *testing.T) {
	localPath, err := os.MkdirTemp("", "git-local-*")
	if err != nil {
		t.Fatalf("Failed to create local temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(localPath) }()

	localRepo, err := gogit.PlainInit(localPath, false)
	if err != nil {
		t.Fatalf("Failed to init local repo: %v", err)
	}

	// Create an initial commit in local to establish HEAD
	wLocal, err := localRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get local worktree: %v", err)
	}
	fLocal, err := os.Create(localPath + "/dummy.txt")
	if err != nil {
		t.Fatalf("Failed to create dummy file: %v", err)
	}
	if _, err := fLocal.Write([]byte("dummy content")); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if err := fLocal.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
	if _, err := wLocal.Add("dummy.txt"); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	_, err = wLocal.Commit("Dummy commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit dummy to local: %v", err)
	}

	// Create remote but point it to nowhere
	_, err = localRepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"/path/to/nowhere"},
	})
	if err != nil {
		t.Fatalf("Failed to create remote in local repo: %v", err)
	}

	cfg := &repoConfig.Config{RepoDir: localPath, Branch: "master"}
	m := NewManager(cfg)

	err = m.Pull(context.Background())
	if err == nil {
		t.Fatal("Expected fetch error, got nil")
	}
}
