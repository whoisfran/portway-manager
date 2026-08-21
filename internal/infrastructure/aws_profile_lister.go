package infrastructure

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

var (
	profileHeaderCredentials = regexp.MustCompile(`^\[([a-zA-Z0-9_.\-]+)\]$`)
	profileHeaderConfig      = regexp.MustCompile(`^\[profile ([a-zA-Z0-9_.\-]+)\]$`)
)

// localAWSProfileLister lee ~/.aws/config y ~/.aws/credentials para
// obtener los perfiles de AWS configurados en el equipo del usuario.
type localAWSProfileLister struct{}

func NewLocalAWSProfileLister() domain.AWSProfileLister {
	return &localAWSProfileLister{}
}

func (l *localAWSProfileLister) ListProfiles() ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	set := map[string]bool{}

	parseFile := func(path string, isConfig bool) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if isConfig {
				if m := profileHeaderConfig.FindStringSubmatch(line); m != nil {
					set[m[1]] = true
					continue
				}
				if line == "[default]" {
					set["default"] = true
				}
			} else if m := profileHeaderCredentials.FindStringSubmatch(line); m != nil {
				set[m[1]] = true
			}
		}

		if err := scanner.Err(); err != nil {
			return
		}
	}

	parseFile(filepath.Join(home, ".aws", "config"), true)
	parseFile(filepath.Join(home, ".aws", "credentials"), false)

	if len(set) == 0 {
		set["default"] = true
	}

	profiles := make([]string, 0, len(set))
	for p := range set {
		profiles = append(profiles, p)
	}
	sort.Strings(profiles)
	return profiles, nil
}

// Defaults replica como la AWS CLI resuelve el perfil/region cuando no
// se le pasa ninguno explicitamente: primero las variables de entorno
// AWS_PROFILE/AWS_REGION (o sus alias AWS_DEFAULT_PROFILE/
// AWS_DEFAULT_REGION), y en su defecto el perfil "default" y la region
// que tenga configurada en ~/.aws/config.
func (l *localAWSProfileLister) Defaults() models.AWSDefaults {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = os.Getenv("AWS_DEFAULT_PROFILE")
	}
	if profile == "" {
		profile = "default"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = l.regionForProfile(profile)
	}

	return models.AWSDefaults{Profile: profile, Region: region}
}

// regionForProfile busca la clave "region" dentro de la seccion de
// ~/.aws/config que corresponde al perfil indicado.
func (l *localAWSProfileLister) regionForProfile(profile string) string {
	return l.configSection(profile)["region"]
}

// AuthMethod describe, en una frase corta para mostrar en la UI, como
// se autentica el perfil de AWS indicado: SSO, un rol asumido, un
// proceso externo de credenciales, o credenciales locales de acceso.
// Es solo informativo -- no cambia como se inicia el tunel, that lo
// sigue resolviendo la AWS CLI por su cuenta.
func (l *localAWSProfileLister) AuthMethod(profile string) string {
	if profile == "" {
		return "Perfil por defecto"
	}

	section := l.configSection(profile)
	switch {
	case section["sso_start_url"] != "" || section["sso_session"] != "":
		return "AWS SSO"
	case section["credential_process"] != "":
		return "Proceso de credenciales"
	case section["role_arn"] != "":
		return "Rol asumido (STS)"
	}

	if l.hasCredentialsEntry(profile) {
		return "Claves de acceso"
	}

	return "Desconocido"
}

// configSection devuelve las claves "clave = valor" dentro de la
// seccion de ~/.aws/config que corresponde al perfil indicado.
func (l *localAWSProfileLister) configSection(profile string) map[string]string {
	values := map[string]string{}

	home, err := os.UserHomeDir()
	if err != nil {
		return values
	}

	f, err := os.Open(filepath.Join(home, ".aws", "config"))
	if err != nil {
		return values
	}
	defer f.Close()

	header := "[default]"
	if profile != "default" {
		header = "[profile " + profile + "]"
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inSection := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inSection = line == header
			continue
		}
		if !inSection {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			values[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}

	if err := scanner.Err(); err != nil {
		return values
	}

	return values
}

// hasCredentialsEntry indica si ~/.aws/credentials tiene una seccion
// para el perfil indicado (senal de que usa claves de acceso locales).
func (l *localAWSProfileLister) hasCredentialsEntry(profile string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	f, err := os.Open(filepath.Join(home, ".aws", "credentials"))
	if err != nil {
		return false
	}
	defer f.Close()

	header := "[" + profile + "]"
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == header {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		return false
	}

	return false
}
