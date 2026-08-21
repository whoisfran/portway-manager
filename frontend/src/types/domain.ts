// Tipos de dominio del frontend. Coinciden en forma con los tipos que
// wails genera en `@wailsjs/go/models` a partir del backend en Go,
// pero usan nuestro propio vocabulario (ConnectionProfile en vez de
// Favorite, ManagedInstance en vez de Instance) para que el resto de
// la app no dependa de los nombres generados. Al tener los mismos
// campos, TypeScript los acepta de forma intercambiable sin necesidad
// de funciones de mapeo.

/** Mecanismo de tunel de un perfil de conexion: SSM (AWS) o SSH. Se
 *  usa para autocompletar/tipar las opciones que ofrece la UI (ver
 *  ProfileFormModal); el campo `type` de ConnectionProfile en si es
 *  string abierto, por la misma razon que TunnelStatus mas abajo. */
export type ConnectionType = 'ssm' | 'ssh';

/** Como se autentica una conexion SSH; mismo criterio que ConnectionType. */
export type SSHAuthMethod = 'password' | 'privateKey';

/**
 * Perfil de conexion guardado por el usuario. Los campos especificos
 * de cada tipo (SSM: profile/region/instanceId..., SSH:
 * host/user/authMethod...) quedan vacios/undefined en el tipo que no
 * los usa -- igual que en el backend (ver models.Favorite), que es la
 * fuente de verdad de que campos exige cada tipo.
 */
export type ConnectionProfile = {
	id: string;
	label: string;
	/** 'ssm' | 'ssh' en la practica, pero string abierto (ver TunnelStatus) porque asi lo emiten los bindings generados. */
	type: string;
	/** Nombre libre para agrupar perfiles en la lista (p.ej. "Producción"); puramente organizativo. */
	group?: string;
	localPort: number;
	remotePort: number;
	/** Si va vacio, el tunel apunta directo al target (la instancia en SSM, o "localhost" visto desde el servidor SSH). */
	remoteHost?: string;

	// SSM
	/** Nombre del perfil de AWS (~/.aws/config) a usar, puede ir vacio (usa "default"). */
	profile?: string;
	region?: string;
	instanceId?: string;
	instanceLabel?: string;

	// SSH
	host?: string;
	/** Puerto del servidor SSH; vacio/0 se interpreta como 22. */
	port?: number;
	user?: string;
	/** 'password' | 'privateKey' en la practica; string abierto por la misma razon que `type`. */
	authMethod?: string;
	/** Nunca la devuelve el backend (ver ProfileService.List/Get): vacia significa "sin cambios" al editar, no "borrar". */
	password?: string;
	privateKeyPath?: string;
	/** Igual que password: nunca vuelve del backend, vacia = sin cambios al editar. */
	passphrase?: string;

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
	type: string;
	localPort: number;
	remotePort: number;
	remoteHost?: string;

	// SSM
	profile?: string;
	region?: string;
	instanceId?: string;
	instanceLabel?: string;

	// SSH
	host?: string;
	port?: number;
	user?: string;
	authMethod?: string;
	password?: string;
	privateKeyPath?: string;
	passphrase?: string;
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
