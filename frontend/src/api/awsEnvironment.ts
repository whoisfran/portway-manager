import { GetAwsAuthMethod, GetAwsDefaults, ListAwsProfiles, ListAwsRegions, ListManagedInstances } from '@wailsjs/go/main/App';
import type { AwsDefaults, ManagedInstance } from '@/types/domain';

/**
 * Adaptador sobre los bindings de Wails para el entorno de AWS local:
 * los perfiles configurados en ~/.aws, las regiones disponibles y las
 * instancias administradas por SSM para un perfil/region dado.
 */
export const awsEnvironmentApi = {
	listProfiles: (): Promise<string[]> => ListAwsProfiles(),
	listRegions: (): Promise<string[]> => ListAwsRegions(),
	listInstances: (profile: string, region: string): Promise<ManagedInstance[]> => ListManagedInstances(profile, region),
	getDefaults: (): Promise<AwsDefaults> => GetAwsDefaults(),
	/** Solo informativo (SSO / rol asumido / claves de acceso); no afecta como se inicia el tunel. */
	getAuthMethod: (profile: string): Promise<string> => GetAwsAuthMethod(profile),
};
