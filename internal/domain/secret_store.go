package domain

// SecretStore guarda y recupera secretos (contrasena o passphrase de
// llave privada de una conexion SSH) fuera del almacen de perfiles en
// texto plano, delegando al almacen de credenciales del sistema
// operativo -- Keychain en macOS, Credential Manager en Windows,
// Secret Service (libsecret: GNOME Keyring/KWallet) en Linux.
type SecretStore interface {
	Set(key, secret string) error
	// Get devuelve "" sin error cuando la llave no tiene un secreto
	// guardado -- no todo favorito SSH tiene uno (p.ej. una llave
	// privada sin passphrase), asi que la ausencia no es un error.
	Get(key string) (string, error)
	// Delete no debe fallar si la llave no tenia un secreto guardado.
	Delete(key string) error
}
