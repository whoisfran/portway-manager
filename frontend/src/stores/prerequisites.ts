import { prerequisitesApi } from '@/api/prerequisites';
import type { Prerequisites } from '@/types/domain';
import { defineStore } from 'pinia';
import { ref } from 'vue';

export const usePrerequisitesStore = defineStore('prerequisites', () => {
	const loading = ref(false);
	const result = ref<Prerequisites | null>(null);

	async function check() {
		loading.value = true;
		try {
			result.value = await prerequisitesApi.check();
		} catch (err) {
			result.value = {
				awsCliInstalled: false,
				awsCliVersion: '',
				sessionPluginFound: false,
				sessionPluginVersion: '',
				allOk: false,
				message: [(err as Error).message],
			};
		} finally {
			loading.value = false;
		}
	}

	function openInstallDocs() {
		return prerequisitesApi.openInstallDocs();
	}

	return { loading, result, check, openInstallDocs };
});
