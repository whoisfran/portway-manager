import { CheckForUpdates, GetSettings, GetVersion, OpenUpdateURL, SaveSettings } from '@wailsjs/go/main/App';
import type { AppSettings, UpdateInfo } from '@/types/domain';

export const appApi = {
	/** Version de la app (ver main.go: "dev" fuera de un build de release). */
	getVersion: (): Promise<string> => GetVersion(),
	getSettings: (): Promise<AppSettings> => GetSettings(),
	saveSettings: (settings: AppSettings): Promise<AppSettings> => SaveSettings(settings),
	/** Consulta el ultimo release publicado en GitHub; no descarga ni instala nada. */
	checkForUpdates: (): Promise<UpdateInfo> => CheckForUpdates(),
	/** Abre la pagina del release en el navegador del sistema. */
	openUpdateUrl: (url: string): Promise<void> => OpenUpdateURL(url),
};
