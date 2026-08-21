package application

import (
	"fmt"

	"github.com/google/uuid"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// ProfileService valida y gestiona los perfiles de conexion guardados
// por el usuario para reutilizarlos desde la lista de conexiones.
//
// Las credenciales SSH (password/passphrase) nunca viven en el
// Favorite que devuelven List/Save: se guardan aparte en un
// domain.SecretStore (el almacen de claves del sistema operativo) y
// solo se recuperan internamente en Get, que es lo que usa
// TunnelService (via App.StartTunnel) para poder abrir la conexion
// real.
type ProfileService interface {
	List() ([]models.Favorite, error)
	// Get busca un perfil guardado por su ID y completa sus
	// credenciales SSH desde el SecretStore. Lo usa TunnelService (via
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
	secrets    domain.SecretStore
}

func NewProfileService(
	repo domain.ProfileRepository,
	strategies domain.TunnelStrategyRegistry,
	secrets domain.SecretStore,
) ProfileService {
	return &profileService{repo: repo, strategies: strategies, secrets: secrets}
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
			return s.hydrateSecrets(f)
		}
	}
	return models.Favorite{}, fmt.Errorf("no se encontro el perfil de conexion")
}

// Save valida el perfil, mueve sus credenciales SSH (si trae alguna)
// al SecretStore, y persiste el resto sin secretos en el repositorio.
//
// Si el perfil ya existia y esta edicion llega con Password/
// Passphrase vacios, se asume que el usuario no los toco -- List/Get
// nunca le devuelven la credencial real al frontend (ver arriba), asi
// que un formulario de edicion siempre los reenvia vacios salvo que el
// usuario los haya escrito de nuevo a proposito. Por eso se hidratan
// antes de validar: de lo contrario, editar solo el nombre de un
// favorito SSH por password fallaria la validacion por "falta la
// contrasena", cuando en realidad ya esta guardada.
func (s *profileService) Save(profile models.Favorite) (models.Favorite, error) {
	if profile.ID == "" {
		profile.ID = uuid.NewString()
	}

	hydrated, err := s.hydrateSecrets(profile)
	if err != nil {
		return profile, err
	}

	strategy, err := s.strategies.Strategy(hydrated.Type)
	if err != nil {
		return profile, err
	}
	if err := strategy.ValidateFavorite(hydrated); err != nil {
		return profile, err
	}

	if err := s.storeSecrets(hydrated); err != nil {
		return profile, err
	}

	toPersist := hydrated
	toPersist.Password = ""
	toPersist.Passphrase = ""

	return s.repo.Save(toPersist)
}

func (s *profileService) Delete(id string) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Un favorito sin credenciales guardadas (SSM, o SSH por llave sin
	// passphrase) simplemente no tiene nada que borrar aqui -- ver
	// domain.SecretStore.Delete.
	if err := s.secrets.Delete(passwordKey(id)); err != nil {
		return err
	}
	return s.secrets.Delete(passphraseKey(id))
}

// hydrateSecrets completa Password/Passphrase desde el SecretStore
// cuando el favorito los trae vacios. No hace nada para perfiles SSM
// (no tienen credenciales propias) ni para uno sin ID todavia (un
// favorito nuevo no puede tener nada guardado bajo un ID que no
// existe aun).
func (s *profileService) hydrateSecrets(f models.Favorite) (models.Favorite, error) {
	if f.Type != models.FavoriteTypeSSH || f.ID == "" {
		return f, nil
	}

	if f.Password == "" {
		stored, err := s.secrets.Get(passwordKey(f.ID))
		if err != nil {
			return f, err
		}
		f.Password = stored
	}
	if f.Passphrase == "" {
		stored, err := s.secrets.Get(passphraseKey(f.ID))
		if err != nil {
			return f, err
		}
		f.Passphrase = stored
	}
	return f, nil
}

func (s *profileService) storeSecrets(f models.Favorite) error {
	if f.Type != models.FavoriteTypeSSH {
		return nil
	}
	if err := s.setOrClearSecret(passwordKey(f.ID), f.Password); err != nil {
		return err
	}
	return s.setOrClearSecret(passphraseKey(f.ID), f.Passphrase)
}

func (s *profileService) setOrClearSecret(key, value string) error {
	if value == "" {
		return s.secrets.Delete(key)
	}
	return s.secrets.Set(key, value)
}

func passwordKey(favoriteID string) string   { return favoriteID + ":password" }
func passphraseKey(favoriteID string) string { return favoriteID + ":passphrase" }
