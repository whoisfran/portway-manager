// Package ssm agrupa todo lo necesario para tuneles de port-forwarding
// via AWS Systems Manager: el TunnelStrategy (ver strategy.go), el
// listado de instancias administradas y la verificacion de
// prerequisitos del sistema (AWS CLI + session-manager-plugin).
package ssm

import (
	"io"
	"os/exec"

	"ssm-portway/internal/infrastructure/process"
)

// session adapta un *exec.Cmd al puerto domain.RunningSession.
type session struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (s *session) Stdout() io.Reader { return s.stdout }
func (s *session) Stderr() io.Reader { return s.stderr }
func (s *session) Wait() error       { return s.cmd.Wait() }
func (s *session) Kill() error       { return process.KillProcessTree(s.cmd) }
