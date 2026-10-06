//go:build linux

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	dbusNotificationsIface = "org.freedesktop.Notifications"
	dbusNotificationsPath  = dbus.ObjectPath("/org/freedesktop/Notifications")
	// defaultNotificationAction es la accion que el servidor de
	// notificaciones invoca al hacer clic en el cuerpo de la notificacion.
	defaultNotificationAction = "default"
)

// En Linux las notificaciones se mandan directo por D-Bus en vez de
// usar las de Wails: Wails reporta el cierre manual de una
// notificacion (la X, o "limpiar todo" en la bandeja de
// notificaciones) igual que un clic, con el mismo DEFAULT_ACTION, asi
// que no hay forma de distinguirlos desde OnNotificationResponse y la
// app se reabria solo por limpiar notificaciones. Aqui solo se
// reacciona a ActionInvoked; NotificationClosed solo sirve para olvidar
// la notificacion.
var notifier struct {
	mu      sync.Mutex
	conn    *dbus.Conn
	appName string
	// pending asocia el ID que asigno el servidor de notificaciones al
	// perfil de esa notificacion, para seleccionarlo al hacer clic.
	pending map[uint32]string
}

func initNotifications(ctx context.Context) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Printf("no se pudo inicializar el servicio de notificaciones: %v", err)
		return
	}

	for _, member := range []string{"ActionInvoked", "NotificationClosed"} {
		if err := conn.AddMatchSignal(
			dbus.WithMatchObjectPath(dbusNotificationsPath),
			dbus.WithMatchInterface(dbusNotificationsIface),
			dbus.WithMatchMember(member),
		); err != nil {
			log.Printf("no se pudo escuchar las notificaciones (%s): %v", member, err)
			_ = conn.Close()
			return
		}
	}

	appName := "portway-manager"
	if exe, err := os.Executable(); err == nil {
		appName = filepath.Base(exe)
	}

	notifier.mu.Lock()
	notifier.conn = conn
	notifier.appName = appName
	notifier.pending = make(map[uint32]string)
	notifier.mu.Unlock()

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	// Igual que watchSystemTheme: Wails no cancela ctx al apagar, asi
	// que se cierra la conexion cuando trayDone se cierra.
	go func() {
		<-trayDone
		_ = conn.Close()
	}()

	go func() {
		for signal := range signals {
			handleNotificationSignal(ctx, signal)
		}
	}()
}

func handleNotificationSignal(ctx context.Context, signal *dbus.Signal) {
	if len(signal.Body) < 2 {
		return
	}
	id, ok := signal.Body[0].(uint32)
	if !ok {
		return
	}

	// Ambas senales retiran la notificacion: el servidor la difunde a
	// todos los clientes del bus, asi que las de otras apps simplemente
	// no estan en pending.
	notifier.mu.Lock()
	favoriteID, ours := notifier.pending[id]
	delete(notifier.pending, id)
	notifier.mu.Unlock()
	if !ours {
		return
	}

	if signal.Name != dbusNotificationsIface+".ActionInvoked" {
		return
	}
	if action, _ := signal.Body[1].(string); action != defaultNotificationAction {
		return
	}
	openFromNotification(ctx, favoriteID)
}

// sendNotification muestra una notificacion de sistema que, al hacer
// clic en ella, reabre la ventana y selecciona el perfil favoriteID
// (ver openFromNotification). El id solo lo usan otras plataformas.
func sendNotification(_ context.Context, _ string, title, body, favoriteID string) {
	notifier.mu.Lock()
	conn, appName := notifier.conn, notifier.appName
	notifier.mu.Unlock()
	if conn == nil {
		return
	}

	call := conn.Object(dbusNotificationsIface, dbusNotificationsPath).Call(
		dbusNotificationsIface+".Notify", 0,
		appName,
		uint32(0), // replaces_id
		"",        // icono
		title,
		body,
		[]string{defaultNotificationAction, "Abrir"},
		map[string]dbus.Variant{},
		int32(-1), // timeout por defecto del servidor
	)
	if call.Err != nil {
		log.Printf("no se pudo enviar la notificacion del sistema: %v", call.Err)
		return
	}

	var id uint32
	if err := call.Store(&id); err != nil {
		log.Printf("no se pudo enviar la notificacion del sistema: %v", err)
		return
	}

	notifier.mu.Lock()
	notifier.pending[id] = favoriteID
	notifier.mu.Unlock()
}
