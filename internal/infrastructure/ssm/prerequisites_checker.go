package ssm

import (
	"os/exec"
	"regexp"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

var awsCliVersionRegex = regexp.MustCompile(`aws-cli/([0-9]+\.[0-9]+\.[0-9]+)`)

// prerequisitesChecker verifica que "aws" cli y
// "session-manager-plugin" existan en el PATH del sistema, ya que son
// requeridos para abrir tuneles SSM. Los tuneles SSH no dependen de
// ninguna herramienta externa (ver internal/infrastructure/ssh).
type prerequisitesChecker struct{}

func NewPrerequisitesChecker() domain.PrerequisitesChecker {
	return &prerequisitesChecker{}
}

func (c *prerequisitesChecker) Check() models.Prerequisites {
	p := models.Prerequisites{}

	if path, err := exec.LookPath("aws"); err == nil {
		p.AwsCliInstalled = true
		if out, err := exec.Command(path, "--version").CombinedOutput(); err == nil {
			if match := awsCliVersionRegex.FindStringSubmatch(string(out)); len(match) > 1 {
				p.AwsCliVersion = match[1]
			} else {
				p.AwsCliVersion = "unknown"
			}
		}
	}

	if _, err := exec.LookPath("session-manager-plugin"); err == nil {
		p.SessionPluginFound = true
		if out, err := exec.Command("session-manager-plugin", "--version").CombinedOutput(); err == nil {
			p.SessionPluginVersion = string(out)
		}
	}

	p.AllOk = p.AwsCliInstalled && p.SessionPluginFound
	if !p.AllOk {
		p.Message = append([]string{
			"Falta instalar AWS CLI v2 y/o el Session Manager Plugin.",
			"Documentación de instalación:",
		}, domain.PrerequisitesDocs...)
	}

	return p
}
