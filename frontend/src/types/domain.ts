// Tipos de dominio del frontend. Coinciden en forma con los tipos que
// wails genera en `@wailsjs/go/models` a partir del backend en Go,
// pero usan nuestro propio vocabulario (ConnectionProfile en vez de
// Favorite, ManagedInstance en vez de Instance) para que el resto de
// la app no dependa de los nombres generados. Al tener los mismos
// campos, TypeScript los acepta de forma intercambiable sin necesidad
// de funciones de mapeo.

/** Perfil de conexion guardado por el usuario (perfil de AWS + instancia + puertos). */
export type ConnectionProfile = {
	id: string;
	label: string;
	/** Nombre del perfil de AWS (~/.aws/config) a usar, puede ir vacio (usa "default"). */
	profile: string;
	region: string;
	instanceId: string;
	instanceLabel: string;
	localPort: number;
	remotePort: number;
	/** Si va vacio, el tunel apunta directo a la instancia. */
	remoteHost: string;
	/** ISO 8601; vacio si nunca se ha iniciado un tunel con este perfil. */
	lastConnectedAt?: string;
};

export type ManagedInstance = {
	instanceId: string;
	name: string;
	platformOs: string;
	privateIp: string;
	pingStatus: string;
};

export type Prerequisites = {
	awsCliInstalled: boolean;
	awsCliVersion: string;
	sessionPluginFound: boolean;
	sessionPluginVersion: string;
	allOk: boolean;
	message: string[];
};

export type TunnelRequest = {
	/** ID del ConnectionProfile que origino el tunel; un tunel siempre nace de un perfil ya guardado. */
	favoriteId: string;
	profile: string;
	region: string;
	instanceId: string;
	instanceLabel: string;
	localPort: number;
	remotePort: number;
	remoteHost: string;
};

/** Valores posibles de Tunnel.status; se trata como string abierto (no un union cerrado)
 *  porque asi lo emite el backend, y una UI no deberia romperse ante un valor nuevo. */
export type TunnelStatus = 'starting' | 'running' | 'stopped' | 'error';

export type Tunnel = {
	id: string;
	request: TunnelRequest;
	status: string;
	startedAt: string;
	message: string;
};

export type PortStatus = {
	available: boolean;
	/** true = lo ocupa otro tunel de esta app (aviso); false = lo ocupa el sistema (bloqueante). */
	inUseBySameApp: boolean;
	conflictLabel?: string;
};

export type TunnelLogLine = {
	id: string;
	line: string;
};

/** Perfil/region que la AWS CLI usaria si no se le indica ninguno, para preseleccionar el formulario. */
export type AwsDefaults = {
	profile: string;
	region: string;
};

export type ImportFailure = {
	label: string;
	reason: string;
};

export type ImportResult = {
	importedCount: number;
	failures: ImportFailure[];
};
