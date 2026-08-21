//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

func SetProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func KillProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	pgid := cmd.Process.Pid
	return syscall.Kill(-pgid, syscall.SIGTERM)
}
