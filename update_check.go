package main

import (
	"context"
	"log"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// checkForUpdatesOnStartup consulta una sola vez, al arrancar, si hay
// una version mas nueva publicada (ver App.CheckForUpdates) y si la
// hay, se lo avisa al frontend (ver UpdateNotifier.vue) para que
// muestre un aviso con un boton a la pagina del release -- esta app no
// se autoactualiza. Un fallo aqui (sin internet, GitHub caido, etc) no
// debe molestar al usuario: se registra y ya, como con las
// notificaciones de sistema (ver main.go).
func checkForUpdatesOnStartup(ctx context.Context, app *App) {
	// Deja que la ventana termine de montar antes de la primera
	// consulta de red, para no competir por recursos justo al abrir.
	time.Sleep(3 * time.Second)

	info, err := app.CheckForUpdates()
	if err != nil {
		log.Printf("no se pudo chequear actualizaciones: %v", err)
		return
	}
	if info.Available {
		runtime.EventsEmit(ctx, "update:available", &info)
	}
}
