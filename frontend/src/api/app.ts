import { GetVersion } from '@wailsjs/go/main/App';

export const appApi = {
	/** Version de la app (ver main.go: "dev" fuera de un build de release). */
	getVersion: (): Promise<string> => GetVersion(),
};
