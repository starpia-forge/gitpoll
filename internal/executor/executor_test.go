package executor

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecutor_Execute(t *testing.T) {
	e := NewExecutor("echo 'hello world'")

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
	e := NewExecutor("sleep 5")

	logCh := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := e.Execute(ctx, logCh)
	if err == nil {
		t.Fatal("Expected error due to cancellation, got nil")
	}
}
