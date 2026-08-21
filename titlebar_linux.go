//go:build linux

package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ensureDarkTitleBar hace que la barra de titulo respete el tema del
// sistema en Linux, sin pisarlo.
//
// A diferencia de Windows, Wails no tiene ninguna opcion (windows.Theme
// no aplica aqui) para controlar la barra de titulo en Linux: en
// entornos con decoracion del lado del cliente (GNOME/GTK3) la barra
// de titulo la dibuja GTK dentro del propio proceso de la app, segun
// GTK_THEME. Hay que fijarlo antes de que GTK se inicialice -- Wails
// lo hace mas adelante, dentro de wails.Run -- por eso esto se llama
// al inicio de main().
//
// No forzamos un tema fijo (eso pisaria el tema real del usuario, o
// dejaria de servir si el dia de mañana usa modo claro): en vez de
// eso, leemos el tema GTK y la preferencia clara/oscura que GNOME ya
// tiene configurados, y solo intervenimos para pedir la variante
// oscura de ESE tema cuando el sistema prefiere oscuro y el tema no
// ya lo indica en su nombre (p.ej. "Yaru-dark"). Si el usuario ya fijo
// su propio GTK_THEME, o el sistema no esta en modo oscuro, no
// tocamos nada.
//
// En entornos con decoracion del lado del servidor (p.ej. KDE Plasma
// con KWin) la barra de titulo la dibuja el compositor, no la app: ahi
// esto no tiene efecto y el tema se cambia desde la configuracion de
// la sesion.
func ensureDarkTitleBar() {
	if os.Getenv("GTK_THEME") != "" {
		return
	}

	if !systemPrefersDark() {
		return
	}

	theme := systemGtkTheme()
	if theme == "" || strings.Contains(strings.ToLower(theme), "dark") {
		return
	}

	os.Setenv("GTK_THEME", theme+":dark")
}

func systemPrefersDark() bool {
	out, err := gsettingsGet("org.gnome.desktop.interface", "color-scheme")
	return err == nil && strings.Contains(out, "prefer-dark")
}

func systemGtkTheme() string {
	out, err := gsettingsGet("org.gnome.desktop.interface", "gtk-theme")
	if err != nil {
		return ""
	}
	return strings.Trim(out, "'")
}

func gsettingsGet(schema, key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "gsettings", "get", schema, key).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
