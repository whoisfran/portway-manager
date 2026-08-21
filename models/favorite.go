package models

import "fmt"

// Favorite representa un perfil de conexion guardado por el usuario
// (instancia, puertos, perfil/region de AWS) para reutilizarlo desde
// la lista de conexiones sin volver a llenar el formulario.
type Favorite struct {
	ID            string `json:"id"`
	Label         string `json:"label"`
	Profile       string `json:"profile"`
	Region        string `json:"region"`
	InstanceID    string `json:"instanceId"`
	InstanceLabel string `json:"instanceLabel"`
	LocalPort     int    `json:"localPort"`
	RemotePort    int    `json:"remotePort"`
	RemoteHost    string `json:"remoteHost"`
	// LastConnectedAt queda vacio si nunca se ha iniciado un tunel con
	// este perfil. Se actualiza en el frontend (ver stores/profiles.ts)
	// justo despues de que un StartTunnel con estos mismos datos tiene
	// exito; el backend no liga tuneles a un ID de perfil, asi que no
	// puede actualizarlo por su cuenta.
	LastConnectedAt string `json:"lastConnectedAt,omitempty"`
}

// Validate comprueba que el perfil tenga los datos minimos para ser
// util desde la lista de conexiones guardadas.
func (f Favorite) Validate() error {
	if f.Label == "" {
		return fmt.Errorf("el perfil debe tener un nombre")
	}
	if f.InstanceID == "" {
		return fmt.Errorf("debes seleccionar una instancia")
	}
	if f.LocalPort <= 0 || f.RemotePort <= 0 {
		return fmt.Errorf("puertos invalidos")
	}
	return nil
}
