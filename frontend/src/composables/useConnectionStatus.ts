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
 * - rojo:    el ultimo intento de tunel fallo.
 * - naranja: no es un fallo de conexion -- es que el perfil no tiene
 *            un perfil de AWS asignado (p.ej. recien importado), asi
 *            que ni siquiera se puede intentar conectar todavia.
 * - gris:    sin tunel activo, pero configurado correctamente.
 */
export function connectionStatus(profile: ConnectionProfile, activeTunnel: Tunnel | undefined): ConnectionStatus {
	if (activeTunnel?.status === 'error') {
		return { color: 'error', label: 'Falló la conexión' };
	}
	if (activeTunnel?.status === 'running' || activeTunnel?.status === 'starting') {
		return { color: 'success', label: 'Listo' };
	}
	if (!profile.profile) {
		return { color: 'warning', label: 'Falta perfil de AWS' };
	}
	return { color: 'neutral', label: 'Detenido' };
}
