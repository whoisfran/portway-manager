package domain

import "portway-manager/models"

// SettingsRepository persiste los ajustes globales de la app (ver
// models.AppSettings), fuera de cualquier perfil de conexion en
// particular.
type SettingsRepository interface {
	Get() (models.AppSettings, error)
	Save(models.AppSettings) (models.AppSettings, error)
}
