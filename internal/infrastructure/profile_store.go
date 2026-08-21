package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// jsonProfileStore persiste los perfiles de conexion en un archivo
// JSON dentro del directorio de configuracion del usuario (por
// ejemplo, %AppData%/ssm-tunnel-manager en Windows o
// ~/.config/ssm-tunnel-manager en Linux/macOS), fuera del alcance de
// otras apps.
type jsonProfileStore struct {
	mu   sync.Mutex
	path string
}

func NewJSONProfileStore() (domain.ProfileRepository, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}

	appDir := filepath.Join(dir, "ssm-tunnel-manager")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio de configuracion: %w", err)
	}

	return &jsonProfileStore{path: filepath.Join(appDir, "favorites.json")}, nil
}

func (s *jsonProfileStore) load() ([]models.Favorite, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Favorite{}, nil
		}
		return nil, err
	}

	var profiles []models.Favorite
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *jsonProfileStore) persist(profiles []models.Favorite) error {
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *jsonProfileStore) List() ([]models.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *jsonProfileStore) Save(profile models.Favorite) (models.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.load()
	if err != nil {
		return profile, err
	}

	replaced := false
	for i, p := range profiles {
		if p.ID == profile.ID {
			profiles[i] = profile
			replaced = true
			break
		}
	}
	if !replaced {
		profiles = append(profiles, profile)
	}

	if err := s.persist(profiles); err != nil {
		return profile, err
	}
	return profile, nil
}

func (s *jsonProfileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.load()
	if err != nil {
		return err
	}

	kept := make([]models.Favorite, 0, len(profiles))
	for _, p := range profiles {
		if p.ID != id {
			kept = append(kept, p)
		}
	}
	if len(kept) == len(profiles) {
		return fmt.Errorf("perfil no encontrado")
	}
	return s.persist(kept)
}
