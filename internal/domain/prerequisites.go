package domain

import "ssm-portway/models"

// PrerequisitesChecker verifica que las herramientas de AWS necesarias
// para abrir sesiones SSM (CLI y Session Manager Plugin) esten
// disponibles en el sistema.
type PrerequisitesChecker interface {
	Check() models.Prerequisites
}

// PrerequisitesDocs son los enlaces de instalacion que se muestran (y
// pueden abrirse en el navegador) cuando falta algun prerequisito.
var PrerequisitesDocs = []string{
	"https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html",
	"https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
}
