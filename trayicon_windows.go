//go:build windows

package main

import (
	_ "embed"
	"time"

	"github.com/energye/systray"
	"golang.org/x/sys/windows/registry"
)

// Iconos de la bandeja en Windows: deben ser .ico (energye/systray los
// escribe a un archivo temporal y los carga con LoadImage), a
// diferencia de Linux/macOS que esperan PNG. "light"/"dark" se refiere
// al fondo de la barra de tareas: glifo negro y blanco respectivamente.
var (
	//go:embed build/tray/windows/tray-light.ico
	trayIconLight []byte
	//go:embed build/tray/windows/tray-dark.ico
	trayIconDark []byte
	//go:embed build/tray/windows/tray-brand.ico
	trayIconBrand []byte
)

// applyTrayIcon usa el glifo azul de la marca mientras haya tuneles
// activos y, si no, el que contraste con la barra de tareas.
func applyTrayIcon(active bool) {
	switch {
	case active:
		setTrayIcon("brand", func() { systray.SetIcon(trayIconBrand) })
	case taskbarUsesLightTheme():
		setTrayIcon("light", func() { systray.SetIcon(trayIconLight) })
	default:
		setTrayIcon("dark", func() { systray.SetIcon(trayIconDark) })
	}
}

// taskbarUsesLightTheme lee el tema de la barra de tareas
// (SystemUsesLightTheme), que es independiente del de las apps
// (AppsUseLightTheme). Sin el valor (Windows antiguos) se asume barra
// oscura, que es la de siempre.
func taskbarUsesLightTheme() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	v, _, err := k.GetIntegerValue("SystemUsesLightTheme")
	return err == nil && v == 1
}

// watchTrayTheme revisa periodicamente el tema de la barra de tareas
// para cambiar el icono en caliente; setTrayIcon evita recargarlo si
// no cambio nada.
func watchTrayTheme() {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-trayDone:
				return
			case <-ticker.C:
				refreshTrayIcon()
			}
		}
	}()
}
