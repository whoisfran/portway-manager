import { EventsOn } from '@wailsjs/runtime';
import type { Tunnel, TunnelLogLine } from '@/types/domain';

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
