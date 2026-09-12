package domain

import "portway-manager/models"

// UpdateChecker consulta si hay una version mas nueva publicada que
// currentVersion. No descarga ni instala nada -- esta app no se
// autoactualiza, solo avisa (ver models.UpdateInfo.URL).
type UpdateChecker interface {
	Check(currentVersion string) (models.UpdateInfo, error)
}
