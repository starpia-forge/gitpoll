//go:build !windows
// +build !windows

package executor

import (
	"os/exec"
	"syscall"
)

// setProcessGroup sets the process group ID so that the entire process tree can be killed.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the process group.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// A negative PID sends the signal to all processes in the process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
