//go:build !linux

package main

// En Windows la barra de titulo se controla via windows.Options.Theme
// (ver main.go); en macOS sigue el modo oscuro del sistema. Ninguno de
// los dos necesita el ajuste de GTK_THEME que si hace falta en Linux.
func ensureDarkTitleBar() {}
