//go:build linux

package main

import (
	"bufio"
	"context"
	"log"
	"os/exec"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// watchSystemTheme reenvia al frontend, en caliente, los cambios de la
// preferencia clara/oscura del sistema (ver systemPrefersDark, en
// titlebar_linux.go).
//
// Hace falta porque en Linux el webview (webkit2gtk) no reevalua
// @media (prefers-color-scheme) cuando GNOME cambia el tema con la
// app ya abierta -- a diferencia de WebView2/WKWebView en
// Windows/macOS, donde el frontend ya lo detecta solo con matchMedia
// (ver frontend/src/stores/theme.ts). Aqui empujamos el evento
// nosotros, seleccionando el mismo gsettings que ya usa
// systemPrefersDark: cuando el escritorio no es GNOME/GTK (no hay
// gsettings), el comando simplemente no arranca y la app se queda con
// el tema detectado al iniciar, igual que antes de este cambio.
func watchSystemTheme(ctx context.Context) {
	cmd := exec.Command("gsettings", "monitor", "org.gnome.desktop.interface", "color-scheme")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
		return
	}

	// Wails no cancela ctx al apagar la app (por eso watchMinimize, en
	// tray.go, tampoco depende de el): hay que matar el proceso a mano
	// cuando trayDone se cierra para no dejarlo huerfano corriendo de
	// fondo.
	go func() {
		<-trayDone
		_ = cmd.Process.Kill()
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		runtime.EventsEmit(ctx, "system:theme-changed", systemPrefersDark())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("gsettings monitor: %v", err)
	}
	_ = cmd.Wait()
}
