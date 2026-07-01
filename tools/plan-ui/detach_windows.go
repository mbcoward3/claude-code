//go:build windows

package main

import "os/exec"

// detach is a no-op on Windows; the spawned process already runs independently
// once the parent releases it.
func detach(cmd *exec.Cmd) {}
