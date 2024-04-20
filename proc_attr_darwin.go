//go:build darwin

package main

import (
	"os/exec"
	"syscall"
)

func sysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
