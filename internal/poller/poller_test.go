package poller

import (
	"context"
	"errors"
	"testing"
	"time"

	"repo-gitpoll/internal/events"
)

type mockGitClient struct {
	hashToReturn string
	errToReturn  error
	callCount    int
}

func (m *mockGitClient) LsRemote(ctx context.Context, repoURL, branch string) (string, error) {
	m.callCount++
	return m.hashToReturn, m.errToReturn
}

func TestPoller_BasicPolling(t *testing.T) {
	mockClient := &mockGitClient{
		hashToReturn: "1234567890abcdef",
		errToReturn:  nil,
	}

	outCh := make(chan interface{}, 10)

	p := NewPoller("https://github.com/test/repo", "main", mockClient)
	p.(*defaultPoller).baseInterval = 10 * time.Millisecond
	p.(*defaultPoller).maxJitter = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx, outCh)

	select {
	case msg := <-outCh:
		updateMsg, ok := msg.(events.UpdateDetectedMsg)
		if !ok {
			t.Fatalf("Expected UpdateDetectedMsg, got %T", msg)
		}
		if updateMsg.NewHash != "1234567890abcdef" {
			t.Errorf("Expected hash 1234567890abcdef, got %s", updateMsg.NewHash)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for first polling update")
	}

	mockClient.hashToReturn = "fedcba0987654321"

	select {
	case msg := <-outCh:
		updateMsg, ok := msg.(events.UpdateDetectedMsg)
		if !ok {
			t.Fatalf("Expected UpdateDetectedMsg, got %T", msg)
		}
		if updateMsg.NewHash != "fedcba0987654321" {
			t.Errorf("Expected hash fedcba0987654321, got %s", updateMsg.NewHash)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for second polling update")
	}
}

func TestPoller_ExponentialBackoff(t *testing.T) {
	mockClient := &mockGitClient{
		errToReturn: errors.New("network error"),
	}

	outCh := make(chan interface{}, 10)

	p := NewPoller("https://github.com/test/repo", "main", mockClient)
	p.(*defaultPoller).baseInterval = 5 * time.Millisecond
	p.(*defaultPoller).maxJitter = 1 * time.Millisecond
	p.(*defaultPoller).backoffBase = 10 * time.Millisecond
	p.(*defaultPoller).backoffMax = 100 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Start(ctx, outCh)

	for i := 0; i < 3; i++ {
		select {
		case msg := <-outCh:
			_, ok := msg.(events.ErrorMsg)
			if !ok {
				t.Fatalf("Expected ErrorMsg, got %T", msg)
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout waiting for error message %d", i)
		}
	}
}

func TestPoller_GracefulShutdown(t *testing.T) {
	mockClient := &mockGitClient{
		hashToReturn: "123",
	}

	outCh := make(chan interface{}, 10)

	p := NewPoller("https://github.com/test/repo", "main", mockClient)
	p.(*defaultPoller).baseInterval = 10 * time.Millisecond
	p.(*defaultPoller).maxJitter = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		p.Start(ctx, outCh)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Poller did not shut down gracefully")
	}
}
