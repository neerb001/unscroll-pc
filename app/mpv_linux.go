//go:build linux

package main

import (
	"os/exec"
	"syscall"
)

// mpv stopt vanzelf als de app stopt (ook bij een crash), zodat er nooit muziek blijft doorspelen.
func stopMetOuder(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
