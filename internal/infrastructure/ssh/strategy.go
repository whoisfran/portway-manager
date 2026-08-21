package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"ssm-portway/internal/domain"
	"ssm-portway/internal/infrastructure"
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

// buildHostKeyCallback verifica la llave del servidor SSH con la
// misma politica que "ssh -o StrictHostKeyChecking=accept-new": un
// servidor nunca visto se confia y se registra automaticamente (sin
// que el usuario tenga que conectarse antes por terminal), pero un
// servidor YA conocido cuya llave cambio siempre se rechaza -- eso
// indicaria una posible suplantacion, y jamas se acepta en silencio.
//
// Las llaves nuevas se registran en un known_hosts propio de esta app
// (no en el ~/.ssh/known_hosts real del usuario, que no le pertenece
// a esta app modificar); si ese archivo real ya existe, tambien se
// consulta -- una llave ya confiada por el usuario en su terminal se
// respeta igual.
func buildHostKeyCallback() (ssh.HostKeyCallback, error) {
	appKnownHosts, err := appKnownHostsPath()
	if err != nil {
		return nil, err
	}
	if err := ensureFileExists(appKnownHosts); err != nil {
		return nil, fmt.Errorf("no se pudo preparar el archivo known_hosts de la app: %w", err)
	}

	files := []string{appKnownHosts}
	if home, err := os.UserHomeDir(); err == nil {
		userKnownHosts := filepath.Join(home, ".ssh", "known_hosts")
		if _, err := os.Stat(userKnownHosts); err == nil {
			files = append(files, userKnownHosts)
		}
	}

	lookup, err := knownhosts.New(files...)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer los archivos known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := lookup(hostname, remote, key)
		if err == nil {
			return nil
		}

		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return fmt.Errorf("no se pudo verificar la llave del servidor SSH %s: %w", hostname, err)
		}
		if len(keyErr.Want) > 0 {
			// El host ya era conocido con OTRA llave: posible
			// suplantacion. Nunca se acepta en silencio.
			return fmt.Errorf(
				"la llave del servidor %s no coincide con la que tenias registrada; verificala antes de continuar (%w)",
				hostname, err,
			)
		}

		// Host nunca visto: se confia y se registra para detectar
		// cambios en conexiones futuras.
		if err := appendKnownHost(appKnownHosts, hostname, remote, key); err != nil {
			return err
		}
		return nil
	}, nil
}

// appKnownHostsPath es el archivo known_hosts propio de esta app,
// separado del ~/.ssh/known_hosts real del usuario.
func appKnownHostsPath() (string, error) {
	appDir, err := infrastructure.AppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "known_hosts"), nil
}

func ensureFileExists(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

// appendKnownHost agrega una linea nueva al known_hosts propio de la
// app con la llave que acaba de presentar un servidor nunca antes
// visto.
func appendKnownHost(path, hostname string, remote net.Addr, key ssh.PublicKey) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("no se pudo registrar la llave del servidor %s: %w", hostname, err)
	}
	defer f.Close()

	line := knownhosts.Line(knownHostAddresses(hostname, remote), key)
	if _, err := fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("no se pudo registrar la llave del servidor %s: %w", hostname, err)
	}
	return nil
}

// knownHostAddresses devuelve los patrones de host bajo los que se
// registra una llave nueva: el que se uso para conectar (hostname,
// tal como lo escribio el usuario) y, si es distinto, la direccion IP
// real a la que se conecto -- asi la entrada sigue siendo valida sin
// importar cual de los dos se use la siguiente vez.
func knownHostAddresses(hostname string, remote net.Addr) []string {
	addrs := []string{knownhosts.Normalize(hostname)}
	if remote != nil {
		if addr := knownhosts.Normalize(remote.String()); addr != "" && addr != addrs[0] {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}
