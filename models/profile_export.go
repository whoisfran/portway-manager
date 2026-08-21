package models

// ProfileExportVersion identifica el formato del archivo de
// exportacion, para poder detectar o migrar formatos futuros sin
// romper archivos ya exportados.
const ProfileExportVersion = 1

// ExportedProfile es un perfil de conexion listo para compartir: sin
// el ID interno (se regenera al importar) y sin el nombre del perfil
// de AWS local (el campo "profile" de Favorite), que es especifico
// del equipo de cada usuario y nunca deberia viajar en un archivo que
// se comparte con otra persona.
type ExportedProfile struct {
	Label         string `json:"label"`
	Region        string `json:"region"`
	InstanceID    string `json:"instanceId"`
	InstanceLabel string `json:"instanceLabel"`
	LocalPort     int    `json:"localPort"`
	RemotePort    int    `json:"remotePort"`
	RemoteHost    string `json:"remoteHost"`
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
