//go:build windows

package process

import (
	"os/exec"
	"strconv"
)

func SetProcAttr(cmd *exec.Cmd) {
}

func KillProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	return exec.Command(
		"taskkill",
		"/T",
		"/F",
		"/PID",
		strconv.Itoa(cmd.Process.Pid),
	).Run()
}
