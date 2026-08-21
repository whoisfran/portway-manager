package application

import (
	"fmt"
	"testing"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// fakeProfileRepo y fakeSecretStore son dobles en memoria: alcanza con
// que se comporten como el store JSON real y el keyring real para
// probar que profileService mueve las credenciales SSH del uno al
// otro, sin necesitar un archivo en disco ni un keychain real.
type fakeProfileRepo struct {
	byID map[string]models.Favorite
}

func newFakeProfileRepo() *fakeProfileRepo {
	return &fakeProfileRepo{byID: map[string]models.Favorite{}}
}

func (r *fakeProfileRepo) List() ([]models.Favorite, error) {
	list := make([]models.Favorite, 0, len(r.byID))
	for _, f := range r.byID {
		list = append(list, f)
	}
	return list, nil
}

func (r *fakeProfileRepo) Save(f models.Favorite) (models.Favorite, error) {
	r.byID[f.ID] = f
	return f, nil
}

func (r *fakeProfileRepo) Delete(id string) error {
	if _, ok := r.byID[id]; !ok {
		return fmt.Errorf("perfil no encontrado")
	}
	delete(r.byID, id)
	return nil
}

type fakeSecretStore struct {
	values map[string]string
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{values: map[string]string{}}
}

func (s *fakeSecretStore) Set(key, secret string) error {
	s.values[key] = secret
	return nil
}

func (s *fakeSecretStore) Get(key string) (string, error) {
	return s.values[key], nil
}

func (s *fakeSecretStore) Delete(key string) error {
	delete(s.values, key)
	return nil
}

// fakeSSHStrategy replica solo las reglas de validacion de
// internal/infrastructure/ssh.strategy que le importan a estas
// pruebas, para no depender de ese paquete de infraestructura desde
// application (mantiene la prueba aislada, igual que el resto de
// fakes de este archivo).
type fakeSSHStrategy struct{}

func (fakeSSHStrategy) Type() models.FavoriteType { return models.FavoriteTypeSSH }

func (fakeSSHStrategy) ValidateFavorite(f models.Favorite) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if f.Host == "" {
		return fmt.Errorf("falta el host")
	}
	if f.AuthMethod == models.SSHAuthMethodPassword && f.Password == "" {
		return fmt.Errorf("falta la contrasena")
	}
	if f.AuthMethod == models.SSHAuthMethodPrivateKey && f.PrivateKeyPath == "" {
		return fmt.Errorf("falta la llave privada")
	}
	return nil
}

func (fakeSSHStrategy) BuildRequest(f models.Favorite) models.TunnelRequest {
	return models.TunnelRequest{FavoriteID: f.ID, Type: f.Type, Host: f.Host, Password: f.Password}
}

func (fakeSSHStrategy) ValidateRequest(models.TunnelRequest) error { return nil }

func (fakeSSHStrategy) Start(models.TunnelRequest) (domain.RunningSession, error) { return nil, nil }

type fakeRegistry struct{ strategy domain.TunnelStrategy }

func (r fakeRegistry) Strategy(t models.FavoriteType) (domain.TunnelStrategy, error) {
	if r.strategy.Type() != t {
		return nil, fmt.Errorf("tipo no soportado: %q", t)
	}
	return r.strategy, nil
}

func newTestProfileService() (*profileService, *fakeSecretStore) {
	secrets := newFakeSecretStore()
	svc := &profileService{
		repo:       newFakeProfileRepo(),
		strategies: fakeRegistry{strategy: fakeSSHStrategy{}},
		secrets:    secrets,
	}
	return svc, secrets
}

func sshFavorite(password string) models.Favorite {
	return models.Favorite{
		Label:      "Bastion",
		Type:       models.FavoriteTypeSSH,
		Host:       "bastion.example.com",
		User:       "deploy",
		AuthMethod: models.SSHAuthMethodPassword,
		Password:   password,
		LocalPort:  5432,
		RemotePort: 5432,
	}
}

func TestSaveMovesPasswordToSecretStoreAndStripsIt(t *testing.T) {
	svc, secrets := newTestProfileService()

	saved, err := svc.Save(sshFavorite("s3cr3t"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if saved.Password != "" {
		t.Errorf("Save() devolvio Password = %q, want vacio (no debe salir del SecretStore)", saved.Password)
	}
	if got := secrets.values[passwordKey(saved.ID)]; got != "s3cr3t" {
		t.Errorf("secrets[%s] = %q, want %q", passwordKey(saved.ID), got, "s3cr3t")
	}

	persisted := svc.repo.(*fakeProfileRepo).byID[saved.ID]
	if persisted.Password != "" {
		t.Errorf("persisted.Password = %q, want vacio (nunca en el store JSON)", persisted.Password)
	}
}

func TestSaveKeepsExistingPasswordWhenEditingWithoutResendingIt(t *testing.T) {
	svc, secrets := newTestProfileService()

	saved, err := svc.Save(sshFavorite("s3cr3t"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	edit := saved
	edit.Label = "Bastion produccion"
	edit.Password = "" // el formulario de edicion nunca recibe la real de vuelta

	if _, err := svc.Save(edit); err != nil {
		t.Fatalf("Save() (edicion) error = %v", err)
	}

	if got := secrets.values[passwordKey(saved.ID)]; got != "s3cr3t" {
		t.Errorf("password tras editar sin reenviarla = %q, want %q (no debio borrarse)", got, "s3cr3t")
	}
}

func TestGetHydratesPasswordForStartingATunnel(t *testing.T) {
	svc, _ := newTestProfileService()

	saved, err := svc.Save(sshFavorite("s3cr3t"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := svc.Get(saved.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Password != "s3cr3t" {
		t.Errorf("Get().Password = %q, want %q (debe hidratarse para poder conectar)", got.Password, "s3cr3t")
	}
}

func TestDeleteRemovesStoredSecrets(t *testing.T) {
	svc, secrets := newTestProfileService()

	saved, err := svc.Save(sshFavorite("s3cr3t"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := svc.Delete(saved.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, ok := secrets.values[passwordKey(saved.ID)]; ok {
		t.Error("la contrasena sigue en el SecretStore despues de borrar el favorito")
	}
}
