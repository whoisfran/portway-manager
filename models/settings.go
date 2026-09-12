package models

// AppSettings son los ajustes globales de la app, sin relacion con
// ningun perfil de conexion en particular (a diferencia de Favorite).
type AppSettings struct {
	// AutoReconnectSSM/AutoReconnectSSH controlan si, cuando un tunel
	// se cae solo (sin que el usuario lo haya detenido), la app
	// reintenta abrirlo unas pocas veces antes de darse por vencida
	// (ver TunnelService). Van separados porque las causas tipicas son
	// distintas: un SSM suele cortarse por el timeout de inactividad
	// de Session Manager del lado de AWS (seguro reintentar solo),
	// mientras que un SSH puede caerse por algo que conviene que el
	// usuario note, y reintentar credenciales en automatico contra un
	// servidor SSH puede parecer un patron sospechoso para cosas como
	// fail2ban.
	AutoReconnectSSM bool `json:"autoReconnectSsm"`
	AutoReconnectSSH bool `json:"autoReconnectSsh"`
}

// DefaultAppSettings son los valores con los que arranca una
// instalacion nueva, antes de que el usuario haya guardado nada.
func DefaultAppSettings() AppSettings {
	return AppSettings{AutoReconnectSSM: true, AutoReconnectSSH: false}
}
