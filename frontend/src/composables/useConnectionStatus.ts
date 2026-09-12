import type { ConnectionProfile, Tunnel } from '@/types/domain';

export type ConnectionStatusColor = 'success' | 'warning' | 'error' | 'neutral';

export type ConnectionStatus = {
	color: ConnectionStatusColor;
	label: string;
};

/**
 * Un solo lugar para decidir el color/etiqueta de una conexion, para
 * que el punto de estado de la lista y el badge del panel de detalle
 * nunca se desincronicen:
 *
 * - verde:   el tunel esta corriendo (o arrancando).
 * - naranja: el tunel se cayo solo y se esta reintentando (ver
 *            reconnectLoop en tunnel_service.go), o -- igual que
 *            antes -- al perfil le falta algo minimo para poder
 *            intentar conectar (p.ej. recien importado sin perfil de
 *            AWS, o una conexion SSH sin host/usuario todavia).
 * - rojo:    el ultimo intento de tunel fallo (se agotaron los
 *            reintentos, o la reconexion automatica esta apagada).
 * - gris:    sin tunel activo, pero configurado correctamente.
 */
export function connectionStatus(profile: ConnectionProfile, activeTunnel: Tunnel | undefined): ConnectionStatus {
	if (activeTunnel?.status === 'error') {
		return { color: 'error', label: 'Falló la conexión' };
	}
	if (activeTunnel?.status === 'reconnecting') {
		return { color: 'warning', label: 'Reconectando…' };
	}
	if (activeTunnel?.status === 'running' || activeTunnel?.status === 'starting') {
		return { color: 'success', label: 'Listo' };
	}
	if (profile.type === 'ssh') {
		if (!profile.host || !profile.user) {
			return { color: 'warning', label: 'Falta configurar el host SSH' };
		}
	} else if (!profile.profile) {
		return { color: 'warning', label: 'Falta perfil de AWS' };
	}
	return { color: 'neutral', label: 'Detenido' };
}
