package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
)

// appConfigDirName es la carpeta dentro del directorio de
// configuracion del usuario (p.ej. %AppData% en Windows o
// ~/.config en Linux/macOS) donde esta app guarda todo su estado
// local -- favoritos, known_hosts propio, etc -- fuera del alcance de
// otras apps.
const appConfigDirName = "ssm-tunnel-manager"

// AppConfigDir devuelve el directorio de configuracion de esta app,
// creandolo si no existe todavia.
func AppConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}

	appDir := filepath.Join(dir, appConfigDirName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear el directorio de configuracion: %w", err)
	}
	return appDir, nil
}
