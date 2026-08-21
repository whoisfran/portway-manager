package infrastructure

import (
	"testing"

	"github.com/zalando/go-keyring"
)

// TestMain instala el proveedor en memoria de go-keyring antes de
// correr las pruebas de este paquete, para no depender de un keychain
// real del sistema operativo.
func TestMain(m *testing.M) {
	keyring.MockInit()
	m.Run()
}

func TestOSKeyringSecretStoreRoundTrip(t *testing.T) {
	store := NewOSKeyringSecretStore()

	if err := store.Set("fav-1:password", "s3cr3t"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	got, err := store.Get("fav-1:password")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "s3cr3t" {
		t.Errorf("Get() = %q, want %q", got, "s3cr3t")
	}

	if err := store.Delete("fav-1:password"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	got, err = store.Get("fav-1:password")
	if err != nil {
		t.Fatalf("Get() tras Delete() error = %v", err)
	}
	if got != "" {
		t.Errorf("Get() tras Delete() = %q, want vacio", got)
	}
}

func TestOSKeyringSecretStoreGetMissingKeyReturnsEmpty(t *testing.T) {
	store := NewOSKeyringSecretStore()

	got, err := store.Get("no-existe:password")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil para una llave sin secreto guardado", err)
	}
	if got != "" {
		t.Errorf("Get() = %q, want vacio", got)
	}
}

func TestOSKeyringSecretStoreDeleteMissingKeyDoesNotFail(t *testing.T) {
	store := NewOSKeyringSecretStore()

	if err := store.Delete("no-existe:password"); err != nil {
		t.Errorf("Delete() error = %v, want nil al borrar una llave sin secreto guardado", err)
	}
}
