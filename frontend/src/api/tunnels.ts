import { CheckLocalPort, ListActiveTunnels, StartTunnel, StopTunnel } from '@wailsjs/go/main/App';
import type { PortStatus, Tunnel } from '@/types/domain';

/** Adaptador sobre los bindings de Wails para el ciclo de vida de los tuneles. */
export const tunnelsApi = {
	list: (): Promise<Tunnel[]> => ListActiveTunnels(),
	/** No existe conexion rapida: un tunel siempre nace de un perfil ya guardado (ver favoriteId). */
	start: (favoriteId: string): Promise<Tunnel> => StartTunnel(favoriteId),
	stop: (id: string): Promise<void> => StopTunnel(id),
	/** Solo lectura: no reserva el puerto, pensado para validar mientras se llena el formulario. */
	checkPort: (localPort: number): Promise<PortStatus> => CheckLocalPort(localPort),
};
