package application

import (
	"fmt"

	"github.com/google/uuid"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// ProfileService valida y gestiona los perfiles de conexion guardados
// por el usuario para reutilizarlos desde la lista de conexiones.
type ProfileService interface {
	List() ([]models.Favorite, error)
	// Get busca un perfil guardado por su ID. Lo usa TunnelService (via
	// App.StartTunnel) para construir el TunnelRequest a partir del
	// perfil, en vez de que el frontend tenga que reenviar sus campos
	// por su cuenta -- el perfil guardado es la unica fuente de verdad.
	Get(id string) (models.Favorite, error)
	Save(profile models.Favorite) (models.Favorite, error)
	Delete(id string) error
}

type profileService struct {
	repo       domain.ProfileRepository
	strategies domain.TunnelStrategyRegistry
}

func NewProfileService(repo domain.ProfileRepository, strategies domain.TunnelStrategyRegistry) ProfileService {
	return &profileService{repo: repo, strategies: strategies}
}

func (s *profileService) List() ([]models.Favorite, error) {
	return s.repo.List()
}

func (s *profileService) Get(id string) (models.Favorite, error) {
	favorites, err := s.repo.List()
	if err != nil {
		return models.Favorite{}, err
	}
	for _, f := range favorites {
		if f.ID == id {
			return f, nil
		}
	}
	return models.Favorite{}, fmt.Errorf("no se encontro el perfil de conexion")
}

// Save valida el perfil y le asigna un ID nuevo si es la primera vez
// que se guarda; si ya tiene ID, actualiza el existente.
func (s *profileService) Save(profile models.Favorite) (models.Favorite, error) {
	strategy, err := s.strategies.Strategy(profile.Type)
	if err != nil {
		return profile, err
	}
	if err := strategy.ValidateFavorite(profile); err != nil {
		return profile, err
	}

	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}

	return s.repo.Save(profile)
}

func (s *profileService) Delete(id string) error {
	return s.repo.Delete(id)
}
