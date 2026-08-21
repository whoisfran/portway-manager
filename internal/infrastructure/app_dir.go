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
const appConfigDirName = "portway-manager"

// legacyAppConfigDirName fue el nombre de esta carpeta cuando la app
// todavia se llamaba "ssm-portway" y solo soportaba SSM. Se conserva
// aqui unicamente para migrar instalaciones existentes (ver
// AppConfigDir) -- nunca se escribe en ella.
const legacyAppConfigDirName = "ssm-tunnel-manager"

// AppConfigDir devuelve el directorio de configuracion de esta app,
// migrando el de una instalacion anterior (ver legacyAppConfigDirName)
// si existe, o creandolo desde cero si no hay ninguno todavia.
func AppConfigDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}

	appDir := filepath.Join(dir, appConfigDirName)
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		legacyDir := filepath.Join(dir, legacyAppConfigDirName)
		if _, err := os.Stat(legacyDir); err == nil {
			if err := os.Rename(legacyDir, appDir); err != nil {
				return "", fmt.Errorf("no se pudo migrar el directorio de configuracion de la version anterior: %w", err)
			}
		}
	}

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("no se pudo crear el directorio de configuracion: %w", err)
	}
	return appDir, nil
}
