//go:build windows

package process

import (
	"os/exec"
	"strconv"
	"syscall"
)

func SetProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
}

func KillProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	killCmd := exec.Command(
		"taskkill",
		"/T",
		"/F",
		"/PID",
		strconv.Itoa(cmd.Process.Pid),
	)
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return killCmd.Run()
}
