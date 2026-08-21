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
// abrir un tunel port-forwarding a traves de SSM.
type TunnelRequest struct {
	// FavoriteID enlaza el tunel con el perfil de conexion guardado
	// (ver Favorite) que lo origino. Un tunel siempre nace de un
	// perfil ya guardado -- no existe una conexion rapida sin
	// guardar-- asi que este campo es obligatorio (ver Validate) y es
	// la fuente de verdad para mostrar el nombre del perfil en la
	// bandeja del sistema (ver tray.go) en vez de tener que adivinarlo
	// a partir de los demas campos.
	FavoriteID    string `json:"favoriteId"`
	Profile       string `json:"profile"`
	Region        string `json:"region"`
	InstanceID    string `json:"instanceId"`
	InstanceLabel string `json:"instanceLabel"`
	LocalPort     int    `json:"localPort"`
	RemotePort    int    `json:"remotePort"`
	RemoteHost    string `json:"remoteHost"` // si viene vacio -> tunel directo a la instancia
}

// Validate comprueba que la solicitud tenga los datos minimos para
// abrir un tunel de port-forwarding via SSM.
func (r TunnelRequest) Validate() error {
	if r.FavoriteID == "" {
		return fmt.Errorf("el tunel debe iniciarse desde un perfil de conexion ya guardado")
	}
	if r.InstanceID == "" {
		return fmt.Errorf("debes seleccionar una instancia")
	}
	if r.LocalPort <= 0 || r.RemotePort <= 0 {
		return fmt.Errorf("puertos invalidos")
	}
	return nil
}
