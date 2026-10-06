import { onTunnelLog, onTunnelStatusChanged } from '@/api/events';
import { tunnelsApi } from '@/api/tunnels';
import type { ConnectionProfile, PortStatus, Tunnel } from '@/types/domain';
import { defineStore } from 'pinia';
import { computed, ref } from 'vue';

const MAX_LOG_LINES = 200;
const TERMINAL_STATUSES = new Set(['stopped', 'error']);

/**
 * Estado de los tuneles activos. Se mantiene en vivo suscribiendose a
 * los eventos "tunnel:status"/"tunnel:log" que emite el backend
 * (subscribe() es idempotente: puede llamarse desde varios componentes
 * sin duplicar la suscripcion). Cuando un tunel llega a un estado
 * terminal (detenido o con error) el backend ya lo retiro de su propio
 * registro, asi que aqui hacemos lo mismo; solo si termino con error
 * se deja constancia en `notice` para que la UI avise (p.ej. un toast).
 *
 * Los logs se guardan por perfil (favoriteId), no por tunel: asi
 * sobreviven a la desconexion -- que es justo cuando el usuario quiere
 * leer por que se cayo -- y se limpian recien al volver a iniciar ese
 * perfil (ver start) o al cerrar la app.
 */
export const useTunnelsStore = defineStore('tunnels', () => {
	const tunnels = ref<Record<string, Tunnel>>({});
	const logsByProfile = ref<Record<string, string[]>>({});
	const loading = ref(false);
	const error = ref<string | null>(null);
	const notice = ref<{ tunnel: Tunnel } | null>(null);

	const activeTunnels = computed(() =>
		Object.values(tunnels.value).sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1)),
	);

	function appendLog(favoriteId: string, line: string) {
		const lines = logsByProfile.value[favoriteId] ?? [];
		logsByProfile.value = { ...logsByProfile.value, [favoriteId]: [...lines, line].slice(-MAX_LOG_LINES) };
	}

	function handleStatus(tunnel: Tunnel) {
		if (TERMINAL_STATUSES.has(tunnel.status)) {
			const { [tunnel.id]: _removedTunnel, ...restTunnels } = tunnels.value;
			tunnels.value = restTunnels;
			if (tunnel.status === 'error') notice.value = { tunnel };
			return;
		}
		tunnels.value = { ...tunnels.value, [tunnel.id]: tunnel };
	}

	let stopListening: (() => void) | null = null;

	function subscribe() {
		if (stopListening) return;
		const offStatus = onTunnelStatusChanged(handleStatus);
		const offLog = onTunnelLog(({ favoriteId, line }) => appendLog(favoriteId, line));
		stopListening = () => {
			offStatus();
			offLog();
		};
	}

	async function fetchAll() {
		loading.value = true;
		error.value = null;
		try {
			const list = await tunnelsApi.list();
			tunnels.value = Object.fromEntries(list.map((t) => [t.id, t]));
		} catch (err) {
			error.value = (err as Error).message;
		} finally {
			loading.value = false;
		}
	}

	async function start(favoriteId: string): Promise<Tunnel> {
		// Antes de llamar al backend: las lineas que emita la sesion nueva
		// (incluido un posible error de arranque) ya caen en el log limpio.
		clearLogs(favoriteId);
		const tunnel = await tunnelsApi.start(favoriteId);
		tunnels.value = { ...tunnels.value, [tunnel.id]: tunnel };
		return tunnel;
	}

	async function stop(id: string): Promise<void> {
		await tunnelsApi.stop(id);
	}

	function checkPort(localPort: number): Promise<PortStatus> {
		return tunnelsApi.checkPort(localPort);
	}

	function clearLogs(favoriteId: string) {
		const { [favoriteId]: _removed, ...rest } = logsByProfile.value;
		logsByProfile.value = rest;
	}

	function findFor(profile: ConnectionProfile): Tunnel | undefined {
		return activeTunnels.value.find((t) => t.request.favoriteId === profile.id);
	}

	return {
		activeTunnels,
		logsByProfile,
		loading,
		error,
		notice,
		subscribe,
		fetchAll,
		start,
		stop,
		checkPort,
		clearLogs,
		findFor,
	};
});
