package ssm

import (
	"fmt"
	"os/exec"
	"strconv"

	"ssm-portway/internal/domain"
	"ssm-portway/internal/infrastructure/process"
	"ssm-portway/models"
)

// strategy implementa domain.TunnelStrategy para tuneles de
// port-forwarding via AWS Systems Manager: ejecuta "aws ssm
// start-session" con el documento correspondiente.
type strategy struct{}

func NewTunnelStrategy() domain.TunnelStrategy {
	return &strategy{}
}

func (s *strategy) Type() models.FavoriteType { return models.FavoriteTypeSSM }

func (s *strategy) ValidateFavorite(f models.Favorite) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if f.InstanceID == "" {
		return fmt.Errorf("debes seleccionar una instancia")
	}
	return nil
}

func (s *strategy) BuildRequest(f models.Favorite) models.TunnelRequest {
	return models.TunnelRequest{
		FavoriteID:    f.ID,
		Type:          models.FavoriteTypeSSM,
		LocalPort:     f.LocalPort,
		RemotePort:    f.RemotePort,
		RemoteHost:    f.RemoteHost,
		Profile:       f.Profile,
		Region:        f.Region,
		InstanceID:    f.InstanceID,
		InstanceLabel: f.InstanceLabel,
	}
}

func (s *strategy) ValidateRequest(req models.TunnelRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	if req.InstanceID == "" {
		return fmt.Errorf("debes seleccionar una instancia")
	}
	return nil
}

func (s *strategy) Start(req models.TunnelRequest) (domain.RunningSession, error) {
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

	return &session{cmd: cmd, stdout: stdout, stderr: stderr}, nil
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
