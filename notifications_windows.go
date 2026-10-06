//go:build windows

package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// initNotifications usa las notificaciones de Wails tal cual: en
// Windows/macOS solo hay respuesta cuando el usuario hace clic, no
// cuando descarta la notificacion. Ver notifications_linux.go para el
// caso que si los confunde.
func initNotifications(ctx context.Context) {
	// Las notificaciones de sistema son un extra, no algo de lo que
	// dependa poder usar la app: si no hay un servicio disponible, esto
	// no debe tumbar la app entera.
	if err := runtime.InitializeNotifications(ctx); err != nil {
		log.Printf("no se pudo inicializar el servicio de notificaciones: %v", err)
		return
	}

	runtime.OnNotificationResponse(ctx, func(result runtime.NotificationResult) {
		favoriteID, _ := result.Response.UserInfo["favoriteId"].(string)
		openFromNotification(ctx, favoriteID)
	})
}

// sendNotification muestra una notificacion de sistema que, al hacer
// clic en ella, reabre la ventana y selecciona el perfil favoriteID
// (ver openFromNotification).
func sendNotification(ctx context.Context, id, title, body, favoriteID string) {
	if err := runtime.SendNotification(ctx, runtime.NotificationOptions{
		ID:    id,
		Title: title,
		Body:  body,
		// Recuperado en OnNotificationResponse (ver initNotifications).
		Data: map[string]interface{}{"favoriteId": favoriteID},
	}); err != nil {
		log.Printf("no se pudo enviar la notificacion del sistema: %v", err)
	}
}
