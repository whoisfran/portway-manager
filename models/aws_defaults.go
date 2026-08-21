package models

// AWSDefaults describe el perfil y la region "activos" segun el
// entorno de AWS del usuario: las variables AWS_PROFILE/AWS_REGION si
// estan definidas, o en su defecto el perfil "default" y la region
// configurada para el en ~/.aws/config. Se usan unicamente para
// preseleccionar el formulario de un nuevo perfil; el usuario puede
// cambiarlos libremente.
type AWSDefaults struct {
	Profile string `json:"profile"`
	Region  string `json:"region"`
}
