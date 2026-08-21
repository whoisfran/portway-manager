package infrastructure

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"

	"ssm-portway/internal/domain"
)

// keyringService agrupa todos los secretos de esta app bajo un mismo
// "service" en el almacen de credenciales del sistema; cada secreto
// se distingue por su "user" (ver profileService.passwordKey/
// passphraseKey en internal/application).
const keyringService = "ssm-portway"

// osKeyringSecretStore delega en el almacen de credenciales nativo
// del sistema operativo (via github.com/zalando/go-keyring): Keychain
// en macOS, Credential Manager en Windows, Secret Service en Linux
// (requiere un proveedor corriendo, p.ej. gnome-keyring o kwallet).
type osKeyringSecretStore struct{}

func NewOSKeyringSecretStore() domain.SecretStore {
	return &osKeyringSecretStore{}
}

func (s *osKeyringSecretStore) Set(key, secret string) error {
	if err := keyring.Set(keyringService, key, secret); err != nil {
		return fmt.Errorf("no se pudo guardar la credencial en el almacen de claves del sistema: %w", err)
	}
	return nil
}

func (s *osKeyringSecretStore) Get(key string) (string, error) {
	secret, err := keyring.Get(keyringService, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la credencial del almacen de claves del sistema: %w", err)
	}
	return secret, nil
}

func (s *osKeyringSecretStore) Delete(key string) error {
	if err := keyring.Delete(keyringService, key); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("no se pudo borrar la credencial del almacen de claves del sistema: %w", err)
	}
	return nil
}
