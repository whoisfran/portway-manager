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
 * registro, asi que aqui hacemos lo mismo y dejamos constancia en
 * `notice` para que la UI decida como avisar (p.ej. un toast).
 */
export const useTunnelsStore = defineStore('tunnels', () => {
	const tunnels = ref<Record<string, Tunnel>>({});
	const logsByTunnel = ref<Record<string, string[]>>({});
	const loading = ref(false);
	const error = ref<string | null>(null);
	const notice = ref<{ tunnel: Tunnel; isError: boolean } | null>(null);

	const activeTunnels = computed(() =>
		Object.values(tunnels.value).sort((a, b) => (a.startedAt < b.startedAt ? 1 : -1)),
	);

	function appendLog(id: string, line: string) {
		const lines = logsByTunnel.value[id] ?? [];
		logsByTunnel.value = { ...logsByTunnel.value, [id]: [...lines, line].slice(-MAX_LOG_LINES) };
	}

	function handleStatus(tunnel: Tunnel) {
		if (TERMINAL_STATUSES.has(tunnel.status)) {
			const { [tunnel.id]: _removedTunnel, ...restTunnels } = tunnels.value;
			const { [tunnel.id]: _removedLogs, ...restLogs } = logsByTunnel.value;
			tunnels.value = restTunnels;
			logsByTunnel.value = restLogs;
			notice.value = { tunnel, isError: tunnel.status === 'error' };
			return;
		}
		tunnels.value = { ...tunnels.value, [tunnel.id]: tunnel };
	}

	let stopListening: (() => void) | null = null;

	function subscribe() {
		if (stopListening) return;
		const offStatus = onTunnelStatusChanged(handleStatus);
		const offLog = onTunnelLog(({ id, line }) => appendLog(id, line));
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

	function clearLogs(id: string) {
		const { [id]: _removed, ...rest } = logsByTunnel.value;
		logsByTunnel.value = rest;
	}

	function findFor(profile: ConnectionProfile): Tunnel | undefined {
		return activeTunnels.value.find((t) => t.request.favoriteId === profile.id);
	}

	return {
		activeTunnels,
		logsByTunnel,
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
