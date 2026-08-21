package domain

import "ssm-portway/models"

// AWSProfileLister lista los perfiles de AWS configurados localmente
// por el usuario (~/.aws/config y ~/.aws/credentials). Un mismo equipo
// puede tener muchos perfiles (cada uno con sus propias credenciales o
// rol asumido); el usuario elige cual usar por conexion.
type AWSProfileLister interface {
	ListProfiles() ([]string, error)
	// Defaults indica que perfil/region usaria la AWS CLI si no se le
	// especifica ninguno, para preseleccionarlos en el formulario de un
	// perfil nuevo.
	Defaults() models.AWSDefaults
	// AuthMethod describe como se autentica el perfil indicado (SSO,
	// rol asumido, claves de acceso, etc.), solo con fines informativos
	// para mostrarlo en el detalle de la conexion.
	AuthMethod(profile string) string
}

// SupportedRegions son las regiones de AWS ofrecidas en el formulario
// de conexion. AWS tiene mas regiones que estas; la lista cubre las de
// uso mas comun y puede extenderse sin afectar el resto de la app.
var SupportedRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
	"sa-east-1",
	"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
	"ca-central-1",
}
