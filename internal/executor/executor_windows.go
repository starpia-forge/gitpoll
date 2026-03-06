//go:build windows
// +build windows

package executor

import (
	"os/exec"
)

// setProcessGroup on Windows does nothing in this basic implementation.
// For full process tree termination on Windows, one would typically use
// Taskkill or Job Objects.
func setProcessGroup(cmd *exec.Cmd) {
	// Not supported natively via Setpgid on Windows
}

// killProcessGroup on Windows just kills the process itself.
// This might leave orphaned child processes, but is a fallback for Windows.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
