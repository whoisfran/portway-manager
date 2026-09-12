import { GetSettings, GetVersion, SaveSettings } from '@wailsjs/go/main/App';
import type { AppSettings } from '@/types/domain';

export const appApi = {
	/** Version de la app (ver main.go: "dev" fuera de un build de release). */
	getVersion: (): Promise<string> => GetVersion(),
	getSettings: (): Promise<AppSettings> => GetSettings(),
	saveSettings: (settings: AppSettings): Promise<AppSettings> => SaveSettings(settings),
};
