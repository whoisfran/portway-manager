package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"portway-manager/models"
)

// maxTrayProfileSlots limita cuantas lineas de "perfiles activos" se
// muestran en el menu de la bandeja. energye/systray no permite
// quitar items ya creados (solo Show()/Hide()), asi que se reserva un
// numero fijo de renglones por adelantado y se muestran/ocultan segun
// haga falta; si hay mas tuneles activos que renglones, el sobrante se
// resume en un ultimo item ("y N mas").
const maxTrayProfileSlots = 8

// windowHidden indica si la ventana principal esta oculta en la
// bandeja (por el boton de minimizar o por el watcher de minimizado
// nativo). Se consulta antes de emitir una notificacion de sistema:
// solo tiene sentido avisar por ese canal cuando el usuario no esta ya
// viendo la ventana (y, por lo tanto, el aviso en pantalla del propio
// frontend).
var windowHidden atomic.Bool

// appCtx guarda el contexto de Wails para que los callbacks del menu
// de bandeja (que corren en goroutines propias de energye/systray, no
// en las de Wails) puedan invocar el runtime aunque se disparen antes
// de que App.startup lo haya fijado.
var (
	appCtxMu sync.RWMutex
	appCtx   context.Context
)

func setAppContext(ctx context.Context) {
	appCtxMu.Lock()
	appCtx = ctx
	appCtxMu.Unlock()
}

func getAppContext() context.Context {
	appCtxMu.RLock()
	defer appCtxMu.RUnlock()
	return appCtx
}

type trayMenu struct {
	header       *systray.MenuItem
	profileSlots [maxTrayProfileSlots]*systray.MenuItem
	overflow     *systray.MenuItem
}

var tray trayMenu

// trayDone se cierra al apagar la app para detener el watcher de
// minimizado nativo (ver watchMinimize). No hay problema si el
// proceso termina sin llegar a cerrarlo: es una sola goroutine que
// muere con el proceso de todas formas.
var trayDone = make(chan struct{})

// setupTray arma el icono y el menu de la bandeja del sistema. Se
// invoca como el callback "onReady" de systray.RunWithExternalLoop
// (ver main.go), antes de que Wails termine de inicializar la
// ventana: por eso los manejadores de clic resuelven el contexto en
// el momento (getAppContext) en vez de recibirlo por parametro.
func setupTray(app *App) {
	systray.SetIcon(trayIconData)
	systray.SetTooltip("Portway Manager")

	mOpen := systray.AddMenuItem("Abrir", "Mostrar la ventana de Portway Manager")
	systray.AddSeparator()

	tray.header = systray.AddMenuItem("Sin conexiones activas", "")
	tray.header.Disable()
	for i := range tray.profileSlots {
		slot := systray.AddMenuItem("", "")
		slot.Disable()
		slot.Hide()
		tray.profileSlots[i] = slot
	}
	tray.overflow = systray.AddMenuItem("", "")
	tray.overflow.Disable()
	tray.overflow.Hide()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Salir", "Cerrar Portway Manager por completo (detiene los tuneles activos)")

	mOpen.Click(func() { showWindow(getAppContext()) })
	// En Windows y macOS el clic izquierdo no muestra el menu por
	// defecto (a diferencia del clic derecho), asi que se usa para
	// reabrir la ventana. En Linux depende de que el entorno de
	// escritorio soporte el metodo "Activate" del protocolo
	// StatusNotifierItem; donde no lo soporte, "Abrir" en el menu
	// sigue funcionando.
	systray.SetOnClick(func(_ systray.IMenu) { showWindow(getAppContext()) })
	mQuit.Click(func() {
		if ctx := getAppContext(); ctx != nil {
			runtime.Quit(ctx)
		}
	})

	refreshTrayProfiles(app)
}

// showWindow restaura la ventana principal desde la bandeja: la marca
// como visible y limpia cualquier estado de minimizado nativo que
// pudiera tener.
func showWindow(ctx context.Context) {
	if ctx == nil {
		return
	}
	windowHidden.Store(false)
	runtime.WindowShow(ctx)
	runtime.WindowUnminimise(ctx)
}

// hideWindow oculta la ventana principal sin cerrar la app: los
// tuneles activos siguen corriendo y la app se recupera desde el
// icono de la bandeja.
func hideWindow(ctx context.Context) {
	if ctx == nil {
		return
	}
	windowHidden.Store(true)
	runtime.WindowHide(ctx)
}

// watchMinimize complementa al boton de minimizar del titlebar propio
// (ver App.HideToTray): la ventana no tiene marco nativo, pero el
// sistema operativo/gestor de ventanas igual puede minimizarla por
// otras vias (atajo de teclado, boton derecho sobre la barra de
// tareas, etc). Para que "minimizar" ocurra siempre por la bandeja sin
// importar como se disparo, se sondea el estado cada cierto tiempo y,
// si ya quedo minimizada nativamente, se reemplaza por un ocultamiento
// a la bandeja.
func watchMinimize(ctx context.Context) {
	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-trayDone:
			return
		case <-ticker.C:
			if runtime.WindowIsMinimised(ctx) {
				hideWindow(ctx)
			}
		}
	}
}

// watchTunnelStatus mantiene el resumen de perfiles activos de la
// bandeja al dia y, cuando la ventana esta oculta, refleja alli mismo
// los avisos de desconexion/fallo que el frontend ya muestra como
// toast (ver frontend/src/stores/tunnels.ts) con una notificacion de
// sistema.
func watchTunnelStatus(ctx context.Context, app *App) {
	runtime.EventsOn(ctx, "tunnel:status", func(optionalData ...interface{}) {
		refreshTrayProfiles(app)

		if len(optionalData) == 0 {
			return
		}
		tunnel, ok := optionalData[0].(*models.Tunnel)
		if !ok || (tunnel.Status != "stopped" && tunnel.Status != "error") {
			return
		}
		if !windowHidden.Load() {
			return
		}
		notifyTunnelEnded(ctx, app, tunnel)
	})
}

func notifyTunnelEnded(ctx context.Context, app *App, tunnel *models.Tunnel) {
	label := tunnelDisplayLabel(favoritesByID(app), tunnel)

	title := "Tunel detenido"
	body := label
	if tunnel.Status == "error" {
		title = "Conexion interrumpida"
		if tunnel.Message != "" {
			body = fmt.Sprintf("%s: %s", label, tunnel.Message)
		}
	}

	if err := runtime.SendNotification(ctx, runtime.NotificationOptions{
		ID:    tunnel.ID,
		Title: title,
		Body:  body,
	}); err != nil {
		log.Printf("no se pudo enviar la notificacion del sistema: %v", err)
	}
}

// refreshTrayProfiles reconstruye el resumen de "perfiles activos" del
// menu de la bandeja a partir de los tuneles que sigue el
// TunnelService. Se ordenan por inicio para que la lista no cambie de
// orden en cada refresco (el mapa interno del servicio no garantiza
// ningun orden).
func refreshTrayProfiles(app *App) {
	tunnels := app.tunnelService.List()
	sort.Slice(tunnels, func(i, j int) bool {
		return tunnels[i].StartedAt.Before(tunnels[j].StartedAt)
	})

	if len(tunnels) == 0 {
		tray.header.SetTitle("Sin conexiones activas")
	} else {
		tray.header.SetTitle(fmt.Sprintf("Perfiles activos (%d)", len(tunnels)))
	}

	favorites := favoritesByID(app)
	for i, slot := range tray.profileSlots {
		if i >= len(tunnels) {
			slot.Hide()
			continue
		}
		slot.SetTitle(fmt.Sprintf("%s -> %d", tunnelDisplayLabel(favorites, tunnels[i]), tunnels[i].Request.LocalPort))
		slot.Show()
	}

	if extra := len(tunnels) - len(tray.profileSlots); extra > 0 {
		tray.overflow.SetTitle(fmt.Sprintf("y %d mas...", extra))
		tray.overflow.Show()
	} else {
		tray.overflow.Hide()
	}
}

// favoritesByID trae los perfiles guardados indexados por ID, para
// resolver el nombre real (Label) del perfil que origino cada tunel
// activo (ver TunnelRequest.FavoriteID). Un error al leer el almacen
// de perfiles no debe romper la bandeja: se degrada al nombre por
// instancia/perfil de AWS (ver tunnelLabel).
func favoritesByID(app *App) map[string]models.Favorite {
	favorites, err := app.profileService.List()
	if err != nil {
		return nil
	}
	byID := make(map[string]models.Favorite, len(favorites))
	for _, f := range favorites {
		byID[f.ID] = f
	}
	return byID
}

// tunnelLabel resuelve el nombre a mostrar para un tunel: el Label del
// perfil guardado que lo origino (lo que el usuario realmente le puso
// al perfil, p.ej. "Develop DB"). Si el perfil ya no existe (se borro
// mientras el tunel seguia activo) cae al nombre de la instancia, al
// host SSH o al perfil de AWS, para no dejar la bandeja con un
// renglon en blanco.
func tunnelLabel(favorites map[string]models.Favorite, t *models.Tunnel) string {
	if fav, ok := favorites[t.Request.FavoriteID]; ok && fav.Label != "" {
		return fav.Label
	}
	return t.Request.TargetLabel()
}

// tunnelTypeLabel es el prefijo corto que distingue de un vistazo, en
// el menu de la bandeja y en las notificaciones, si un tunel es una
// sesion SSM o una conexion SSH.
func tunnelTypeLabel(t models.FavoriteType) string {
	if t == models.FavoriteTypeSSH {
		return "SSH"
	}
	return "SSM"
}

// tunnelDisplayLabel combina el tipo de tunel con su nombre, p.ej.
// "SSM Database" o "SSH Bastion".
func tunnelDisplayLabel(favorites map[string]models.Favorite, t *models.Tunnel) string {
	return fmt.Sprintf("%s %s", tunnelTypeLabel(t.Request.Type), tunnelLabel(favorites, t))
}
