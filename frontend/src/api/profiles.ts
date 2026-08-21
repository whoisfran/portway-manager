import { DeleteFavorite, ExportFavorites, ImportFavorites, ListFavorites, SaveFavorite } from '@wailsjs/go/main/App';
import type { ConnectionProfile, ImportResult } from '@/types/domain';

/** Adaptador sobre los bindings de Wails para los perfiles de conexion guardados. */
export const profilesApi = {
	list: (): Promise<ConnectionProfile[]> => ListFavorites(),
	save: (profile: ConnectionProfile): Promise<ConnectionProfile> => SaveFavorite(profile),
	remove: (id: string): Promise<void> => DeleteFavorite(id),
	/** Abre el dialogo nativo de guardado; devuelve la ruta elegida, o cadena vacia si el usuario cancelo. */
	export: (): Promise<string> => ExportFavorites(),
	/** Abre el dialogo nativo de apertura; importedCount es 0 si el usuario cancelo. */
	import: (): Promise<ImportResult> => ImportFavorites(),
};
