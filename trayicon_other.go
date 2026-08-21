//go:build !windows

package main

import _ "embed"

// trayIconData es el icono de la bandeja del sistema en Linux/macOS:
// energye/systray decodifica estos bytes como PNG, a diferencia de
// Windows que espera un .ico.
//
//go:embed build/trayicon.png
var trayIconData []byte
