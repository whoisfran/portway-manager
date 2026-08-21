export namespace models {
	
	export class AWSDefaults {
	    profile: string;
	    region: string;
	
	    static createFrom(source: any = {}) {
	        return new AWSDefaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = source["profile"];
	        this.region = source["region"];
	    }
	}
	export class Favorite {
	    id: string;
	    label: string;
	    type: string;
	    localPort: number;
	    remotePort: number;
	    remoteHost?: string;
	    profile?: string;
	    region?: string;
	    instanceId?: string;
	    instanceLabel?: string;
	    host?: string;
	    port?: number;
	    user?: string;
	    authMethod?: string;
	    password?: string;
	    privateKeyPath?: string;
	    passphrase?: string;
	    lastConnectedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Favorite(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.type = source["type"];
	        this.localPort = source["localPort"];
	        this.remotePort = source["remotePort"];
	        this.remoteHost = source["remoteHost"];
	        this.profile = source["profile"];
	        this.region = source["region"];
	        this.instanceId = source["instanceId"];
	        this.instanceLabel = source["instanceLabel"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authMethod = source["authMethod"];
	        this.password = source["password"];
	        this.privateKeyPath = source["privateKeyPath"];
	        this.passphrase = source["passphrase"];
	        this.lastConnectedAt = source["lastConnectedAt"];
	    }
	}
	export class ImportFailure {
	    label: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportResult {
	    importedCount: number;
	    failures: ImportFailure[];
	
	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.importedCount = source["importedCount"];
	        this.failures = this.convertValues(source["failures"], ImportFailure);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Instance {
	    instanceId: string;
	    name: string;
	    platformOs: string;
	    privateIp: string;
	    pingStatus: string;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.instanceId = source["instanceId"];
	        this.name = source["name"];
	        this.platformOs = source["platformOs"];
	        this.privateIp = source["privateIp"];
	        this.pingStatus = source["pingStatus"];
	    }
	}
	export class PortStatus {
	    available: boolean;
	    inUseBySameApp: boolean;
	    conflictLabel?: string;
	
	    static createFrom(source: any = {}) {
	        return new PortStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.inUseBySameApp = source["inUseBySameApp"];
	        this.conflictLabel = source["conflictLabel"];
	    }
	}
	export class Prerequisites {
	    awsCliInstalled: boolean;
	    awsCliVersion: string;
	    sessionPluginFound: boolean;
	    sessionPluginVersion: string;
	    allOk: boolean;
	    message: string[];
	
	    static createFrom(source: any = {}) {
	        return new Prerequisites(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.awsCliInstalled = source["awsCliInstalled"];
	        this.awsCliVersion = source["awsCliVersion"];
	        this.sessionPluginFound = source["sessionPluginFound"];
	        this.sessionPluginVersion = source["sessionPluginVersion"];
	        this.allOk = source["allOk"];
	        this.message = source["message"];
	    }
	}
	export class TunnelRequest {
	    favoriteId: string;
	    type: string;
	    localPort: number;
	    remotePort: number;
	    remoteHost?: string;
	    profile?: string;
	    region?: string;
	    instanceId?: string;
	    instanceLabel?: string;
	    host?: string;
	    port?: number;
	    user?: string;
	    authMethod?: string;
	    password?: string;
	    privateKeyPath?: string;
	    passphrase?: string;
	
	    static createFrom(source: any = {}) {
	        return new TunnelRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.favoriteId = source["favoriteId"];
	        this.type = source["type"];
	        this.localPort = source["localPort"];
	        this.remotePort = source["remotePort"];
	        this.remoteHost = source["remoteHost"];
	        this.profile = source["profile"];
	        this.region = source["region"];
	        this.instanceId = source["instanceId"];
	        this.instanceLabel = source["instanceLabel"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.authMethod = source["authMethod"];
	        this.password = source["password"];
	        this.privateKeyPath = source["privateKeyPath"];
	        this.passphrase = source["passphrase"];
	    }
	}
	export class Tunnel {
	    id: string;
	    request: TunnelRequest;
	    status: string;
	    // Go type: time
	    startedAt: any;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new Tunnel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.request = this.convertValues(source["request"], TunnelRequest);
	        this.status = source["status"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

