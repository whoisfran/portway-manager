package infrastructure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// jsonSettingsStore persiste los ajustes globales de la app en un
// archivo JSON dentro del directorio de configuracion de esta app (ver
// AppConfigDir), separado de favorites.json porque no describe ningun
// perfil de conexion.
type jsonSettingsStore struct {
	mu   sync.Mutex
	path string
}

func NewJSONSettingsStore() (domain.SettingsRepository, error) {
	appDir, err := AppConfigDir()
	if err != nil {
		return nil, err
	}
	return &jsonSettingsStore{path: filepath.Join(appDir, "settings.json")}, nil
}

// Get devuelve los ajustes guardados, o los valores por defecto si
// todavia no se ha guardado nada (instalacion nueva).
func (s *jsonSettingsStore) Get() (models.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return models.DefaultAppSettings(), nil
		}
		return models.AppSettings{}, err
	}

	// Arranca de los defaults y los sobreescribe con lo que haya en
	// disco, para que un ajuste agregado despues de que el archivo ya
	// existia (p.ej. una version futura con un campo nuevo) no quede
	// en su zero-value de Go en vez de su default real.
	settings := models.DefaultAppSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return models.AppSettings{}, err
	}
	return settings, nil
}

func (s *jsonSettingsStore) Save(settings models.AppSettings) (models.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return settings, err
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return settings, err
	}
	return settings, nil
}
