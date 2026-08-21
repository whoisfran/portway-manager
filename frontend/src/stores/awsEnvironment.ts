import { awsEnvironmentApi } from '@/api/awsEnvironment';
import type { AwsDefaults, ManagedInstance } from '@/types/domain';
import { defineStore } from 'pinia';
import { ref } from 'vue';

/**
 * Entorno de AWS disponible localmente: los perfiles configurados en
 * el equipo del usuario (~/.aws/config puede tener varios), las
 * regiones soportadas por el formulario, y las instancias SSM que se
 * cargan a demanda para un perfil/region especifico.
 */
export const useAwsEnvironmentStore = defineStore('awsEnvironment', () => {
	const profiles = ref<string[]>([]);
	const regions = ref<string[]>([]);
	const instances = ref<ManagedInstance[]>([]);
	const defaults = ref<AwsDefaults | null>(null);
	const authMethods = ref<Record<string, string>>({});

	const loadingProfiles = ref(false);
	const loadingInstances = ref(false);
	const instancesError = ref<string | null>(null);

	async function loadProfiles() {
		loadingProfiles.value = true;
		try {
			profiles.value = await awsEnvironmentApi.listProfiles();
		} finally {
			loadingProfiles.value = false;
		}
	}

	async function loadRegions() {
		regions.value = await awsEnvironmentApi.listRegions();
	}

	/** Se resuelve una sola vez por sesion: el perfil/region "por defecto" del entorno no cambia en caliente. */
	async function loadDefaults() {
		defaults.value = await awsEnvironmentApi.getDefaults();
	}

	/** Se cachea por nombre de perfil: no cambia salvo que el usuario edite ~/.aws/config a mano. */
	async function loadAuthMethod(profile: string) {
		if (profile in authMethods.value) return;
		const method = await awsEnvironmentApi.getAuthMethod(profile);
		authMethods.value = { ...authMethods.value, [profile]: method };
	}

	async function loadInstances(profile: string, region: string) {
		loadingInstances.value = true;
		instancesError.value = null;
		instances.value = [];
		try {
			instances.value = await awsEnvironmentApi.listInstances(profile, region);
		} catch (err) {
			instancesError.value = (err as Error).message;
		} finally {
			loadingInstances.value = false;
		}
	}

	return {
		profiles,
		regions,
		instances,
		defaults,
		authMethods,
		loadingProfiles,
		loadingInstances,
		instancesError,
		loadProfiles,
		loadRegions,
		loadInstances,
		loadDefaults,
		loadAuthMethod,
	};
});
