//go:build windows

package main

import _ "embed"

// trayIconData es el icono de la bandeja del sistema en Windows: debe
// ser un .ico (energye/systray lo escribe a un archivo temporal y lo
// carga con LoadImage), a diferencia de Linux/macOS que esperan PNG.
//
//go:embed build/trayicon.ico
var trayIconData []byte
