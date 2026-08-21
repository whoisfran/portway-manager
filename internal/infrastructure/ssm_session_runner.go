// Package infrastructure contiene los adaptadores concretos: AWS SDK,
// ejecucion de procesos del sistema, persistencia en disco y el
// puente hacia el runtime de Wails. Implementan los puertos definidos
// en internal/domain.
package infrastructure

import (
	"fmt"
	"io"
	"os/exec"
	"strconv"

	"ssm-portway/internal/domain"
	"ssm-portway/internal/infrastructure/process"
	"ssm-portway/models"
)

// ssmSession adapta un *exec.Cmd al puerto domain.RunningSession.
type ssmSession struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (s *ssmSession) Stdout() io.Reader { return s.stdout }
func (s *ssmSession) Stderr() io.Reader { return s.stderr }
func (s *ssmSession) Wait() error       { return s.cmd.Wait() }
func (s *ssmSession) Kill() error       { return process.KillProcessTree(s.cmd) }

// ssmSessionRunner inicia tuneles de port-forwarding ejecutando
// "aws ssm start-session" con el documento correspondiente.
type ssmSessionRunner struct{}

func NewSSMSessionRunner() domain.SessionRunner {
	return &ssmSessionRunner{}
}

func (r *ssmSessionRunner) Start(req models.TunnelRequest) (domain.RunningSession, error) {
	cmd := exec.Command("aws", buildSessionArgs(req)...)
	process.SetProcAttr(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &ssmSession{cmd: cmd, stdout: stdout, stderr: stderr}, nil
}

// buildSessionArgs construye los argumentos del CLI: usa el documento
// "ToRemoteHost" cuando hay un host remoto explicito (tunel a traves
// de la instancia hacia otro servicio, p.ej. una base de datos), o el
// documento local cuando el tunel apunta a la instancia misma.
func buildSessionArgs(req models.TunnelRequest) []string {
	args := []string{"ssm", "start-session", "--target", req.InstanceID}

	if req.RemoteHost != "" {
		args = append(args,
			"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
			"--parameters", fmt.Sprintf(
				`{"host":["%s"],"portNumber":["%s"],"localPortNumber":["%s"]}`,
				req.RemoteHost, strconv.Itoa(req.RemotePort), strconv.Itoa(req.LocalPort)),
		)
	} else {
		args = append(args,
			"--document-name", "AWS-StartPortForwardingSession",
			"--parameters", fmt.Sprintf(
				`{"portNumber":["%s"],"localPortNumber":["%s"]}`,
				strconv.Itoa(req.RemotePort), strconv.Itoa(req.LocalPort)),
		)
	}

	if req.Profile != "" {
		args = append(args, "--profile", req.Profile)
	}
	if req.Region != "" {
		args = append(args, "--region", req.Region)
	}

	return args
}
