import { profilesApi } from '@/api/profiles';
import type { ConnectionProfile, ImportResult } from '@/types/domain';
import { defineStore } from 'pinia';
import { ref } from 'vue';

/**
 * Estado de los perfiles de conexion guardados. La carga inicial
 * (fetchAll) absorbe sus propios errores en `error`; las acciones que
 * el usuario dispara a proposito (save/remove) no los absorben: quien
 * las llama decide como mostrar el problema (toast, error inline, etc).
 */
export const useProfilesStore = defineStore('profiles', () => {
	const profiles = ref<ConnectionProfile[]>([]);
	const loading = ref(false);
	const error = ref<string | null>(null);

	async function fetchAll() {
		loading.value = true;
		error.value = null;
		try {
			profiles.value = await profilesApi.list();
		} catch (err) {
			error.value = (err as Error).message;
		} finally {
			loading.value = false;
		}
	}

	async function save(profile: ConnectionProfile): Promise<ConnectionProfile> {
		const saved = await profilesApi.save(profile);
		const index = profiles.value.findIndex((p) => p.id === saved.id);
		if (index === -1) {
			profiles.value.push(saved);
		} else {
			profiles.value[index] = saved;
		}
		return saved;
	}

	async function remove(id: string): Promise<void> {
		await profilesApi.remove(id);
		profiles.value = profiles.value.filter((p) => p.id !== id);
	}

	/** Devuelve un borrador (sin persistir) listo para revisarse en el formulario antes de guardarlo como copia. */
	function duplicate(profile: ConnectionProfile): ConnectionProfile {
		return { ...profile, id: '', label: `${profile.label} (copia)` };
	}

	function getById(id: string): ConnectionProfile | undefined {
		return profiles.value.find((p) => p.id === id);
	}

	/** Se llama justo despues de un StartTunnel exitoso, para que "Última conexión" refleje la realidad. */
	async function markConnected(id: string): Promise<void> {
		const profile = profiles.value.find((p) => p.id === id);
		if (!profile) return;
		await save({ ...profile, lastConnectedAt: new Date().toISOString() });
	}

	/** Devuelve la ruta elegida, o null si el usuario cerro el dialogo sin exportar. */
	async function exportProfiles(): Promise<string | null> {
		const path = await profilesApi.export();
		return path || null;
	}

	/** Los perfiles importados se agregan a la lista existente; nunca la reemplazan. */
	async function importProfiles(): Promise<ImportResult> {
		const result = await profilesApi.import();
		if (result.importedCount > 0) {
			await fetchAll();
		}
		return result;
	}

	return { profiles, loading, error, fetchAll, save, remove, duplicate, getById, markConnected, exportProfiles, importProfiles };
});
