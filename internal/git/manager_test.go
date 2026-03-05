package git

import (
	"context"
	"errors"
	"testing"
)

type mockCmdRunner struct {
	errToReturn error
	callLog     []string
}

func (m *mockCmdRunner) Run(ctx context.Context, dir, command string, args ...string) error {
	m.callLog = append(m.callLog, command)
	return m.errToReturn
}

func TestManager_Pull_Success(t *testing.T) {
	runner := &mockCmdRunner{}
	m := NewManager("/tmp/repo", "main")
	m.(*defaultGitManager).runner = runner

	err := m.Pull(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(runner.callLog) != 2 {
		t.Fatalf("Expected 2 calls (fetch, reset), got %d", len(runner.callLog))
	}
}

func TestManager_Pull_Failure(t *testing.T) {
	runner := &mockCmdRunner{errToReturn: errors.New("git error")}
	m := NewManager("/tmp/repo", "main")
	m.(*defaultGitManager).runner = runner

	err := m.Pull(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
