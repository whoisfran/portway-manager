package models

// PortStatus es el resultado de comprobar si un puerto local esta
// disponible para usarse en un tunel, pensado para que el frontend lo
// valide mientras el usuario llena el formulario (antes de intentar
// arrancar el tunel de verdad).
type PortStatus struct {
	Available bool `json:"available"`
	// InUseBySameApp distingue el origen del conflicto: true si el
	// puerto ya lo usa otro tunel activo de esta misma app (alcanza
	// con un aviso, el usuario puede simplemente detener ese tunel), o
	// false si esta ocupado por el sistema operativo/otro proceso
	// ajeno a la app (hay que pedirle al usuario que elija otro puerto).
	InUseBySameApp bool `json:"inUseBySameApp"`
	// ConflictLabel describe el tunel que ya usa el puerto. Solo se
	// llena cuando InUseBySameApp es true.
	ConflictLabel string `json:"conflictLabel,omitempty"`
}
