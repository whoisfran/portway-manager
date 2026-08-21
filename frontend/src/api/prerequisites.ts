import { CheckPrerequisites, OpenAwsInstallDocs } from '@wailsjs/go/main/App';
import type { Prerequisites } from '@/types/domain';

export const prerequisitesApi = {
	check: (): Promise<Prerequisites> => CheckPrerequisites(),
	/** Abre la documentacion de instalacion en el navegador del sistema (no en un enlace <a> dentro del webview). */
	openInstallDocs: (): Promise<void> => OpenAwsInstallDocs(),
};
