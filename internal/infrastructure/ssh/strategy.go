package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// defaultSSHPort se usa cuando el favorito/solicitud no especifica un
// puerto de servidor SSH explicito.
const defaultSSHPort = 22

// strategy implementa domain.TunnelStrategy para tuneles de
// port-forwarding local (-L) sobre una conexion SSH nativa, con
// autenticacion por password o llave privada.
type strategy struct{}

func NewTunnelStrategy() domain.TunnelStrategy {
	return &strategy{}
}

func (s *strategy) Type() models.FavoriteType { return models.FavoriteTypeSSH }

func (s *strategy) ValidateFavorite(f models.Favorite) error {
	if err := f.Validate(); err != nil {
		return err
	}
	return validateFields(f.Host, f.User, f.AuthMethod, f.Password, f.PrivateKeyPath)
}

func (s *strategy) ValidateRequest(req models.TunnelRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	return validateFields(req.Host, req.User, req.AuthMethod, req.Password, req.PrivateKeyPath)
}

func validateFields(host, user string, authMethod models.SSHAuthMethod, password, privateKeyPath string) error {
	if host == "" {
		return fmt.Errorf("debes indicar el host del servidor SSH")
	}
	if user == "" {
		return fmt.Errorf("debes indicar el usuario de la conexion SSH")
	}
	switch authMethod {
	case models.SSHAuthMethodPassword:
		if password == "" {
			return fmt.Errorf("debes indicar la contrasena de la conexion SSH")
		}
	case models.SSHAuthMethodPrivateKey:
		if privateKeyPath == "" {
			return fmt.Errorf("debes indicar la ruta de la llave privada")
		}
	default:
		return fmt.Errorf("metodo de autenticacion SSH invalido: %q", authMethod)
	}
	return nil
}

func (s *strategy) BuildRequest(f models.Favorite) models.TunnelRequest {
	port := f.Port
	if port <= 0 {
		port = defaultSSHPort
	}

	return models.TunnelRequest{
		FavoriteID:     f.ID,
		Type:           models.FavoriteTypeSSH,
		LocalPort:      f.LocalPort,
		RemotePort:     f.RemotePort,
		RemoteHost:     f.RemoteHost,
		Host:           f.Host,
		Port:           port,
		User:           f.User,
		AuthMethod:     f.AuthMethod,
		Password:       f.Password,
		PrivateKeyPath: f.PrivateKeyPath,
		Passphrase:     f.Passphrase,
	}
}

// Start abre la conexion SSH, arranca a escuchar en el puerto local y
// lanza en segundo plano el ciclo que reenvia cada conexion aceptada
// hacia RemoteHost:RemotePort (o "localhost:RemotePort", visto desde
// el servidor SSH, si RemoteHost viene vacio) a traves de esa
// conexion -- el equivalente de "ssh -L localPort:remoteHost:remotePort
// user@host".
func (s *strategy) Start(req models.TunnelRequest) (domain.RunningSession, error) {
	auth, err := buildAuth(req)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := buildHostKeyCallback()
	if err != nil {
		return nil, err
	}

	port := req.Port
	if port <= 0 {
		port = defaultSSHPort
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(req.Host, strconv.Itoa(port)), &ssh.ClientConfig{
		User:            req.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar a %s@%s: %w", req.User, req.Host, err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", req.LocalPort))
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("no se pudo escuchar en el puerto local %d: %w", req.LocalPort, err)
	}

	remoteHost := req.RemoteHost
	if remoteHost == "" {
		remoteHost = "localhost"
	}
	remoteAddr := net.JoinHostPort(remoteHost, strconv.Itoa(req.RemotePort))

	sess := newSession(listener, client)
	sess.logf("tunel SSH listo: 127.0.0.1:%d -> %s (via %s@%s)", req.LocalPort, remoteAddr, req.User, req.Host)

	go sess.acceptLoop(remoteAddr)

	return sess, nil
}

// buildAuth traduce el metodo de autenticacion del favorito en el
// ssh.AuthMethod correspondiente: password directa, o llave privada
// leida de disco (con passphrase opcional si esta cifrada).
func buildAuth(req models.TunnelRequest) (ssh.AuthMethod, error) {
	switch req.AuthMethod {
	case models.SSHAuthMethodPassword:
		return ssh.Password(req.Password), nil

	case models.SSHAuthMethodPrivateKey:
		keyData, err := os.ReadFile(req.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("no se pudo leer la llave privada %q: %w", req.PrivateKeyPath, err)
		}

		var signer ssh.Signer
		if req.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(req.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("no se pudo leer la llave privada %q: %w", req.PrivateKeyPath, err)
		}
		return ssh.PublicKeys(signer), nil

	default:
		return nil, fmt.Errorf("metodo de autenticacion SSH invalido: %q", req.AuthMethod)
	}
}

// buildHostKeyCallback verifica la llave del servidor SSH contra el
// known_hosts local del usuario (~/.ssh/known_hosts), igual que hace
// el cliente "ssh" del sistema: rechaza servidores desconocidos o con
// una llave distinta a la registrada, en vez de aceptarla a ciegas.
// Si el servidor aun no esta registrado, el usuario debe conectarse
// una vez por terminal para aceptarla (o agregarla a mano).
func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo ubicar el directorio del usuario: %w", err)
	}

	path := filepath.Join(home, ".ssh", "known_hosts")
	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf(
			"no se pudo leer %q para verificar la llave del servidor SSH: conectate una vez por terminal (ssh %s) para registrarla, o agregala manualmente (%w)",
			path, "usuario@host", err,
		)
	}
	return callback, nil
}
