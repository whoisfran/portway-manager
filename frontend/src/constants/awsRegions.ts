// Nombres amigables solo para mostrar en la UI (p.ej. "US East (N.
// Virginia)"); coinciden con domain.SupportedRegions en el backend.
// Si el backend agrega una region que no esta aqui, regionLabel cae
// de vuelta al codigo tal cual -- nunca rompe por una region nueva.
const AWS_REGION_NAMES: Record<string, string> = {
	'us-east-1': 'US East (N. Virginia)',
	'us-east-2': 'US East (Ohio)',
	'us-west-1': 'US West (N. California)',
	'us-west-2': 'US West (Oregon)',
	'eu-west-1': 'Europe (Ireland)',
	'eu-west-2': 'Europe (London)',
	'eu-west-3': 'Europe (Paris)',
	'eu-central-1': 'Europe (Frankfurt)',
	'eu-north-1': 'Europe (Stockholm)',
	'sa-east-1': 'South America (São Paulo)',
	'ap-southeast-1': 'Asia Pacific (Singapore)',
	'ap-southeast-2': 'Asia Pacific (Sydney)',
	'ap-northeast-1': 'Asia Pacific (Tokyo)',
	'ap-northeast-2': 'Asia Pacific (Seoul)',
	'ap-south-1': 'Asia Pacific (Mumbai)',
	'ca-central-1': 'Canada (Central)',
};

/** "US East (N. Virginia) - us-east-1", o solo el codigo si no se reconoce. */
export function regionLabel(code: string): string {
	if (!code) return '';
	const name = AWS_REGION_NAMES[code];
	return name ? `${name} - ${code}` : code;
}
