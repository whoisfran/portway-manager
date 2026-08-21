import { useProfilesStore } from '@/stores/profiles';
import type { ConnectionProfile } from '@/types/domain';
import { defineStore } from 'pinia';
import { computed, ref, watch } from 'vue';

/**
 * Estado de interfaz del feature de perfiles: cual esta seleccionado
 * (vista maestro-detalle) y que modal esta abierto (crear/editar/
 * duplicar). Deliberadamente separado de useProfilesStore -- ese es
 * el estado de dominio (la lista, CRUD contra el backend); esto es
 * solo vista, y lo necesitan componentes que ya no comparten un padre
 * comun (el toolbar vive en Layout; la lista, el detalle y el modal
 * viven en otro lado), asi que ya no hay a quien pasarle esto por
 * props/emit.
 */
export const useProfileUiStore = defineStore('profileUi', () => {
	const profilesStore = useProfilesStore();

	const selectedProfileId = ref<string | null>(null);
	const formOpen = ref(false);
	const editingProfile = ref<ConnectionProfile | null>(null);

	const selectedProfile = computed(
		() => profilesStore.profiles.find((p) => p.id === selectedProfileId.value) ?? null,
	);

	// Si la seleccion actual deja de existir (se borro, o todavia no se
	// ha cargado nada), cae al primer perfil disponible.
	watch(
		() => profilesStore.profiles,
		(list) => {
			if (selectedProfileId.value && list.some((p) => p.id === selectedProfileId.value)) return;
			selectedProfileId.value = list[0]?.id ?? null;
		},
		{ immediate: true },
	);

	function selectProfile(id: string) {
		selectedProfileId.value = id;
	}

	function openCreate() {
		editingProfile.value = null;
		formOpen.value = true;
	}

	function openEdit(profile: ConnectionProfile) {
		editingProfile.value = profile;
		formOpen.value = true;
	}

	function openDuplicate(profile: ConnectionProfile) {
		editingProfile.value = profilesStore.duplicate(profile);
		formOpen.value = true;
	}

	return {
		selectedProfileId,
		selectedProfile,
		formOpen,
		editingProfile,
		selectProfile,
		openCreate,
		openEdit,
		openDuplicate,
	};
});
