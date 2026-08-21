import { useTunnelsStore } from '@/stores/tunnels';
import type { PortStatus } from '@/types/domain';
import { ref, watch, type Ref } from 'vue';

const DEBOUNCE_MS = 400;

/**
 * Valida en vivo (con debounce) si un puerto local esta disponible,
 * mientras el usuario lo escribe en el formulario. Es estado efimero
 * de esta instancia del formulario, por lo que vive en un composable
 * y no en un store global; internamente usa el mismo chequeo del
 * store de tuneles.
 */
export function usePortAvailability(port: Ref<number | null>) {
	const tunnelsStore = useTunnelsStore();
	const status = ref<PortStatus | null>(null);
	const checking = ref(false);

	let timer: ReturnType<typeof setTimeout> | undefined;
	let requestId = 0;

	watch(
		port,
		(value) => {
			status.value = null;
			checking.value = false;
			if (timer) clearTimeout(timer);

			if (!value || value <= 0) {
				return;
			}

			const thisRequest = ++requestId;
			checking.value = true;
			timer = setTimeout(async () => {
				try {
					const result = await tunnelsStore.checkPort(value);
					if (thisRequest === requestId) {
						status.value = result;
					}
				} catch {
					// El chequeo es solo informativo: si falla, no bloqueamos el formulario.
				} finally {
					if (thisRequest === requestId) {
						checking.value = false;
					}
				}
			}, DEBOUNCE_MS);
		},
		{ immediate: true },
	);

	return { status, checking };
}
