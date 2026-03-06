package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"repo-gitpoll/internal/config"
)

func TestExecutor_Execute(t *testing.T) {
	cfg := &config.Config{Command: "echo 'hello world'"}
	e := NewExecutor(cfg)

	logCh := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Capture logs in background
	var output string
	done := make(chan struct{})
	go func() {
		for line := range logCh {
			output += line
		}
		close(done)
	}()

	err := e.Execute(ctx, logCh)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	close(logCh) // Signal listener to finish
	<-done

	if !strings.Contains(output, "hello world") {
		t.Fatalf("Expected log to contain 'hello world', got '%s'", output)
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	cfg := &config.Config{Command: "sleep 5"}
	e := NewExecutor(cfg)

	logCh := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	start := time.Now()
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	err := e.Execute(ctx, logCh)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected error due to cancellation, got nil")
	}

	if duration > 2*time.Second {
		t.Fatalf("Cancellation took too long, expected < 2s, got %v", duration)
	}
}

func TestExecutor_ProcessGroupKill(t *testing.T) {
	// A script that creates a long running child process
	// We use 'sh -c' inherently in Execute.
	// So we can just launch a shell loop that ignores simple signals
	// or sleeps in a loop.
	script := `
while true; do
  sleep 1
done
`
	cfg := &config.Config{Command: script}
	e := NewExecutor(cfg)

	logCh := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	start := time.Now()
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()

	err := e.Execute(ctx, logCh)
	duration := time.Since(start)

	if err == nil {
		t.Fatal("Expected error due to cancellation, got nil")
	}

	if duration > 2*time.Second {
		t.Fatalf("Process group kill failed or took too long, got %v", duration)
	}
}
