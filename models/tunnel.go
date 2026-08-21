package models

import (
	"fmt"
	"time"
)

type Tunnel struct {
	ID        string        `json:"id"`
	Request   TunnelRequest `json:"request"`
	Status    string        `json:"status"` // starting | running | stopped | error
	StartedAt time.Time     `json:"startedAt"`
	Message   string        `json:"message"`
}

// TunnelRequest son los parametros que llegan desde el frontend para
// abrir un tunel de port-forwarding, ya sea via SSM o via SSH. El
// TunnelStrategy correspondiente a Type (ver internal/domain) valida
// que los campos que le tocan estan completos y sabe arrancar la
// sesion con ellos.
type TunnelRequest struct {
	// FavoriteID enlaza el tunel con el perfil de conexion guardado
	// (ver Favorite) que lo origino. Un tunel siempre nace de un
	// perfil ya guardado -- no existe una conexion rapida sin
	// guardar-- asi que este campo es obligatorio (ver Validate) y es
	// la fuente de verdad para mostrar el nombre del perfil en la
	// bandeja del sistema (ver tray.go) en vez de tener que adivinarlo
	// a partir de los demas campos.
	FavoriteID string       `json:"favoriteId"`
	Type       FavoriteType `json:"type"`

	// Comunes a SSM y SSH.
	LocalPort  int    `json:"localPort"`
	RemotePort int    `json:"remotePort"`
	RemoteHost string `json:"remoteHost,omitempty"` // si viene vacio -> tunel directo al target

	// Especificos de SSM.
	Profile       string `json:"profile,omitempty"`
	Region        string `json:"region,omitempty"`
	InstanceID    string `json:"instanceId,omitempty"`
	InstanceLabel string `json:"instanceLabel,omitempty"`

	// Especificos de SSH.
	Host           string        `json:"host,omitempty"`
	Port           int           `json:"port,omitempty"`
	User           string        `json:"user,omitempty"`
	AuthMethod     SSHAuthMethod `json:"authMethod,omitempty"`
	Password       string        `json:"password,omitempty"`
	PrivateKeyPath string        `json:"privateKeyPath,omitempty"`
	Passphrase     string        `json:"passphrase,omitempty"`
}

// Validate comprueba las reglas comunes a cualquier tipo de tunel.
// Las reglas propias de cada Type viven en el TunnelStrategy
// correspondiente -- ver internal/domain.TunnelStrategy.
func (r TunnelRequest) Validate() error {
	if r.FavoriteID == "" {
		return fmt.Errorf("el tunel debe iniciarse desde un perfil de conexion ya guardado")
	}
	if r.Type != FavoriteTypeSSM && r.Type != FavoriteTypeSSH {
		return fmt.Errorf("tipo de tunel invalido: %q", r.Type)
	}
	if r.LocalPort <= 0 || r.RemotePort <= 0 {
		return fmt.Errorf("puertos invalidos")
	}
	return nil
}

// TargetLabel devuelve una descripcion legible del destino del tunel,
// usada en los mensajes de conflicto de puerto (ver TunnelService).
// Prioriza el alias de instancia (SSM) y cae a los datos del host SSH
// cuando no aplica.
func (r TunnelRequest) TargetLabel() string {
	switch {
	case r.InstanceLabel != "":
		return r.InstanceLabel
	case r.InstanceID != "":
		return r.InstanceID
	case r.Host != "" && r.User != "":
		return fmt.Sprintf("%s@%s", r.User, r.Host)
	case r.Host != "":
		return r.Host
	default:
		return "otro tunel"
	}
}
