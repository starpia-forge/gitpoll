package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Executor handles the execution of arbitrary shell commands
type Executor interface {
	Execute(ctx context.Context, logCh chan<- string) error
}

type defaultExecutor struct {
	command string
}

// NewExecutor creates a new command executor
func NewExecutor(cmd string) Executor {
	return &defaultExecutor{
		command: cmd,
	}
}

func (e *defaultExecutor) Execute(ctx context.Context, logCh chan<- string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", e.command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("could not create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	streamOutput := func(r io.Reader, isErr bool) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if isErr {
				line = "[ERROR] " + line
			}
			select {
			case logCh <- line:
			case <-ctx.Done():
				return
			}
		}
	}

	go streamOutput(stdout, false)
	go streamOutput(stderr, true)

	// Wait for streams to finish parsing before cmd.Wait
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}
