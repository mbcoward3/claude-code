//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// detach puts the spawned server in its own session so it survives the parent
// CLI process exiting.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
