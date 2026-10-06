//go:build linux

package main

import (
	_ "embed"
	"os"
	"strings"

	"github.com/energye/systray"
)

// Iconos de la bandeja en Linux: energye/systray decodifica estos
// bytes como PNG (a diferencia de Windows, que espera un .ico). Se usa
// la variante de 44px para que se vea nitida en pantallas HiDPI; el
// panel la escala al tamano que necesite.
var (
	//go:embed build/tray/linux/tray-black-44.png
	trayIconBlack []byte
	//go:embed build/tray/linux/tray-white-44.png
	trayIconWhite []byte
	//go:embed build/tray/linux/tray-brand-44.png
	trayIconBrand []byte
)

// applyTrayIcon usa el glifo azul de la marca mientras haya tuneles
// activos y, si no, el blanco o el negro segun el fondo del panel.
func applyTrayIcon(active bool) {
	switch {
	case active:
		setTrayIcon("brand", func() { systray.SetIcon(trayIconBrand) })
	case trayPanelIsDark():
		setTrayIcon("white", func() { systray.SetIcon(trayIconWhite) })
	default:
		setTrayIcon("black", func() { systray.SetIcon(trayIconBlack) })
	}
}

// trayPanelIsDark decide el fondo sobre el que se dibuja el icono. La
// barra superior de GNOME es oscura aun con el tema claro, asi que
// ahi siempre va el glifo blanco (el negro quedaria invisible); en el
// resto de escritorios (KDE, etc.) el panel sigue al tema del sistema.
func trayPanelIsDark() bool {
	if strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "GNOME") {
		return true
	}
	return systemPrefersDark()
}

// watchTrayTheme no hace nada en Linux: watchSystemTheme (ver
// systemtheme_linux.go) ya detecta los cambios de tema en caliente y
// refresca el icono desde ahi.
func watchTrayTheme() {}
