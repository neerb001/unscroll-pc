//go:build !linux

package main

import "os/exec"

func stopMetOuder(cmd *exec.Cmd) {}
