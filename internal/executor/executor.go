package executor

// Executor handles the execution of arbitrary shell commands
type Executor interface {
	Execute() error
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

func (e *defaultExecutor) Execute() error {
	// TODO: Implement actual command execution logic here
	// (e.g., using os/exec to run e.command)
	return nil
}
