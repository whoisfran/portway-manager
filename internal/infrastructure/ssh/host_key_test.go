package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"testing"

	"golang.org/x/crypto/ssh"
)

// newTestSigner genera una llave ed25519 efimera solo para pruebas.
func newTestSigner(t *testing.T) ssh.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey() error = %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("ssh.NewSignerFromKey() error = %v", err)
	}
	return signer
}

// isolateAppConfigDir aisla el known_hosts propio de la app en un
// directorio temporal: esta prueba nunca debe leer ni escribir el
// ~/.config real de quien la corre. El directorio persiste durante
// toda la prueba, para poder simular varias conexiones separadas
// (cada una con su propio callback, igual que hace Start) contra el
// mismo estado en disco.
func isolateAppConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// newHostKeyCallbackForTest construye un callback nuevo, igual que
// hace Start() en cada conexion real: knownhosts.New lee el archivo
// known_hosts de un jalon al crearse y no se actualiza solo, asi que
// cada conexion necesita el suyo para ver lo que quedo registrado por
// una conexion anterior.
func newHostKeyCallbackForTest(t *testing.T) ssh.HostKeyCallback {
	t.Helper()
	callback, err := buildHostKeyCallback()
	if err != nil {
		t.Fatalf("buildHostKeyCallback() error = %v", err)
	}
	return callback
}

func TestHostKeyCallbackTrustsAndRemembersANewHost(t *testing.T) {
	isolateAppConfigDir(t)
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}
	key := newTestSigner(t).PublicKey()

	if err := newHostKeyCallbackForTest(t)("bastion.example.test:22", remote, key); err != nil {
		t.Fatalf("callback() en primera conexion a un host nuevo = %v, want nil (debe confiar sin exigir known_hosts previo)", err)
	}

	// Una conexion posterior (con su propio callback, como en la
	// realidad) y la MISMA llave debe seguir aceptandose -- ya quedo
	// registrada por la conexion anterior.
	if err := newHostKeyCallbackForTest(t)("bastion.example.test:22", remote, key); err != nil {
		t.Errorf("callback() en segunda conexion con la MISMA llave = %v, want nil", err)
	}
}

func TestHostKeyCallbackRejectsAChangedKey(t *testing.T) {
	isolateAppConfigDir(t)
	remote := &net.TCPAddr{IP: net.ParseIP("203.0.113.10"), Port: 22}

	firstKey := newTestSigner(t).PublicKey()
	if err := newHostKeyCallbackForTest(t)("bastion.example.test:22", remote, firstKey); err != nil {
		t.Fatalf("callback() primera conexion error = %v, want nil", err)
	}

	otherKey := newTestSigner(t).PublicKey()
	if err := newHostKeyCallbackForTest(t)("bastion.example.test:22", remote, otherKey); err == nil {
		t.Fatal("callback() en una conexion posterior con una llave DISTINTA a la registrada = nil, want error (posible suplantacion)")
	}
}
