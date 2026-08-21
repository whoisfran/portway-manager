package models

// ProfileExportVersion identifica el formato del archivo de
// exportacion, para poder detectar o migrar formatos futuros sin
// romper archivos ya exportados.
const ProfileExportVersion = 1

// ExportedProfile es un perfil de conexion listo para compartir: sin
// el ID interno (se regenera al importar) y sin ningun dato especifico
// del equipo de quien lo exporta o secreto -- el perfil de AWS local
// (campo "profile" de Favorite), y en SSH la password/passphrase/ruta
// de llave privada. Quien importa debe poner los suyos.
type ExportedProfile struct {
	Label      string       `json:"label"`
	Type       FavoriteType `json:"type"`
	LocalPort  int          `json:"localPort"`
	RemotePort int          `json:"remotePort"`
	RemoteHost string       `json:"remoteHost,omitempty"`

	// SSM
	Region        string `json:"region,omitempty"`
	InstanceID    string `json:"instanceId,omitempty"`
	InstanceLabel string `json:"instanceLabel,omitempty"`

	// SSH
	Host       string        `json:"host,omitempty"`
	Port       int           `json:"port,omitempty"`
	User       string        `json:"user,omitempty"`
	AuthMethod SSHAuthMethod `json:"authMethod,omitempty"`
}

// ProfileExport es el archivo que se exporta/importa.
type ProfileExport struct {
	Version  int               `json:"version"`
	Profiles []ExportedProfile `json:"profiles"`
}

// ImportResult resume una importacion: cuantos perfiles se agregaron
// y cuales fallaron (y por que), sin abortar el resto por un solo
// perfil invalido.
type ImportResult struct {
	ImportedCount int             `json:"importedCount"`
	Failures      []ImportFailure `json:"failures"`
}

type ImportFailure struct {
	Label  string `json:"label"`
	Reason string `json:"reason"`
}
