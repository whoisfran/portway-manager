import { EventsOn } from '@wailsjs/runtime';
import type { Tunnel, TunnelLogLine, UpdateInfo } from '@/types/domain';

/**
 * Adaptador tipado sobre los eventos que el backend emite via el
 * runtime de Wails para reflejar el estado de los tuneles en vivo.
 * Cada suscripcion devuelve su propia funcion de "unsubscribe".
 */
export function onTunnelStatusChanged(handler: (tunnel: Tunnel) => void): () => void {
	return EventsOn('tunnel:status', handler);
}

export function onTunnelLog(handler: (entry: TunnelLogLine) => void): () => void {
	return EventsOn('tunnel:log', handler);
}

// Solo lo emite el backend en Linux, cuando detecta un cambio de tema
// del sistema en caliente (ver systemtheme_linux.go): en Windows/macOS
// el frontend ya lo detecta solo (ver stores/theme.ts).
export function onSystemThemeChanged(handler: (prefersDark: boolean) => void): () => void {
	return EventsOn('system:theme-changed', handler);
}

// Se emite al hacer clic en la notificacion de sistema de un tunel
// desconectado/con error (ver notifyTunnelEnded y OnNotificationResponse,
// en tray.go/main.go), para seleccionar ese perfil al reabrir la ventana.
export function onProfileSelectRequested(handler: (favoriteId: string) => void): () => void {
	return EventsOn('profile:select-requested', handler);
}

// Se emite una sola vez al iniciar (ver checkForUpdatesOnStartup, en
// update_check.go) si hay una version mas nueva publicada en GitHub.
export function onUpdateAvailable(handler: (info: UpdateInfo) => void): () => void {
	return EventsOn('update:available', handler);
}
