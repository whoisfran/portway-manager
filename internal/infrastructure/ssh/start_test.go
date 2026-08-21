package ssh

import (
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"portway-manager/models"
)

// startTestSSHServer arranca un servidor SSH minimo en 127.0.0.1 que
// solo completa el handshake (nunca sirve canales de reenvio de
// verdad): alcanza para probar que Start() regresa sin colgarse --
// ver el comentario sobre sess.logf en strategy.go.
func startTestSSHServer(t *testing.T) (addr string) {
	t.Helper()

	config := &ssh.ServerConfig{
		PasswordCallback: func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
			return nil, nil // acepta cualquier password: es solo para la prueba
		},
	}
	config.AddHostKey(newTestSigner(t))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
		if err != nil {
			return
		}
		defer sshConn.Close()
		go ssh.DiscardRequests(reqs)
		for newChan := range chans {
			_ = newChan.Reject(ssh.Prohibited, "no soportado en esta prueba")
		}
	}()

	return listener.Addr().String()
}

// freeLocalPort reserva un puerto local libre y lo suelta de
// inmediato, para que Start() pueda tomarlo el mismo.
func freeLocalPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestStartReturnsWithoutHanging es una prueba de regresion: Start()
// escribia su primer log de forma sincrona, ANTES de devolver la
// sesion, a traves de un io.Pipe cuyo Write bloquea hasta que algo lo
// lea -- y ese lector (tunnel_service.go: pump) solo se conecta
// DESPUES de que Start() regresa. El resultado era un deadlock
// permanente: Start() nunca regresaba y el frontend se quedaba
// esperando para siempre al intentar abrir cualquier tunel SSH.
func TestStartReturnsWithoutHanging(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	addr := startTestSSHServer(t)
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("net.SplitHostPort() error = %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("strconv.Atoi() error = %v", err)
	}

	req := models.TunnelRequest{
		Type:       models.FavoriteTypeSSH,
		Host:       host,
		Port:       port,
		User:       "tester",
		AuthMethod: models.SSHAuthMethodPassword,
		Password:   "cualquier-cosa",
		LocalPort:  freeLocalPort(t),
		RemotePort: 80,
	}

	strategy := NewTunnelStrategy()

	type result struct {
		session interface {
			Stdout() io.Reader
			Stderr() io.Reader
			Kill() error
		}
		err error
	}
	done := make(chan result, 1)
	go func() {
		session, err := strategy.Start(req)
		done <- result{session: session, err: err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("Start() error = %v", r.err)
		}
		// Drena los logs, igual que hace tunnel_service.go en
		// produccion, y limpia la sesion al terminar.
		go io.Copy(io.Discard, r.session.Stdout())
		go io.Copy(io.Discard, r.session.Stderr())
		t.Cleanup(func() { _ = r.session.Kill() })
	case <-time.After(5 * time.Second):
		t.Fatal("Start() no regreso en 5s: se colgo (regresion del deadlock en sess.logf)")
	}
}
