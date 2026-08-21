package models

import "fmt"

// FavoriteType distingue el mecanismo de tunel que usa un perfil de
// conexion guardado. Cada tipo requiere un subconjunto distinto de
// campos; esas reglas viven en el domain.TunnelStrategy de cada tipo
// (ver internal/infrastructure/ssm y internal/infrastructure/ssh), no
// en este modelo.
type FavoriteType string

const (
	FavoriteTypeSSM FavoriteType = "ssm"
	FavoriteTypeSSH FavoriteType = "ssh"
)

// SSHAuthMethod distingue como se autentica un perfil de conexion SSH.
type SSHAuthMethod string

const (
	SSHAuthMethodPassword   SSHAuthMethod = "password"
	SSHAuthMethodPrivateKey SSHAuthMethod = "privateKey"
)

// Favorite representa un perfil de conexion guardado por el usuario
// (instancia SSM o host SSH, puertos, credenciales) para reutilizarlo
// desde la lista de conexiones sin volver a llenar el formulario.
//
// Los campos especificos de cada FavoriteType quedan vacios en el
// tipo que no los usa; su presencia se exige en el TunnelStrategy
// correspondiente (ver internal/domain.TunnelStrategy), no aqui.
type Favorite struct {
	ID    string       `json:"id"`
	Label string       `json:"label"`
	Type  FavoriteType `json:"type"`
	// Group es un nombre libre para agrupar perfiles en la lista
	// (p.ej. "Producción", "Cliente ACME"); puramente organizativo,
	// no lo usa ningun TunnelStrategy ni afecta como se abre el tunel.
	Group string `json:"group,omitempty"`

	// Comunes a SSM y SSH: puerto local donde escucha el tunel y, si
	// RemoteHost viene vacio, el destino es el target mismo -- la
	// instancia en SSM, o "localhost" visto desde el servidor SSH.
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	RemoteHost string `json:"remoteHost,omitempty"`

	// Especificos de SSM.
	Profile       string `json:"profile,omitempty"`
	Region        string `json:"region,omitempty"`
	InstanceID    string `json:"instanceId,omitempty"`
	InstanceLabel string `json:"instanceLabel,omitempty"`

	// Especificos de SSH.
	Host           string        `json:"host,omitempty"`
	Port           int           `json:"port,omitempty"` // puerto del servidor SSH; 0 => 22
	User           string        `json:"user,omitempty"`
	AuthMethod     SSHAuthMethod `json:"authMethod,omitempty"`
	Password       string        `json:"password,omitempty"`
	PrivateKeyPath string        `json:"privateKeyPath,omitempty"`
	Passphrase     string        `json:"passphrase,omitempty"`

	// LastConnectedAt queda vacio si nunca se ha iniciado un tunel con
	// este perfil. Se actualiza en el frontend (ver stores/profiles.ts)
	// justo despues de que un StartTunnel con estos mismos datos tiene
	// exito; el backend no liga tuneles a un ID de perfil, asi que no
	// puede actualizarlo por su cuenta.
	LastConnectedAt string `json:"lastConnectedAt,omitempty"`
}

// Validate comprueba las reglas comunes a cualquier tipo de perfil.
// Las reglas propias de cada FavoriteType (que campos son
// obligatorios segun sea SSM o SSH) viven en el TunnelStrategy
// correspondiente -- ver internal/domain.TunnelStrategy.
func (f Favorite) Validate() error {
	if f.Label == "" {
		return fmt.Errorf("el perfil debe tener un nombre")
	}
	if f.Type != FavoriteTypeSSM && f.Type != FavoriteTypeSSH {
		return fmt.Errorf("tipo de perfil invalido: %q", f.Type)
	}
	if f.LocalPort <= 0 || f.RemotePort <= 0 {
		return fmt.Errorf("puertos invalidos")
	}
	return nil
}
