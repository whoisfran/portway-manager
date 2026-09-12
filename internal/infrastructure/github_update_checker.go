package infrastructure

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// githubUpdateChecker consulta el ultimo release publicado en GitHub
// (ver .github/workflows/release-*.yaml, que etiquetan cada release
// como "vX.Y.Z" y son la fuente de verdad de que version es la mas
// reciente) para avisar si hay una version mas nueva que la que esta
// corriendo. No hace falta ningun servicio de actualizaciones propio:
// la API publica de GitHub, sin autenticacion, alcanza para esto.
type githubUpdateChecker struct {
	// repo va como "owner/repo" (ver NewGitHubUpdateChecker).
	repo   string
	client *http.Client
}

// NewGitHubUpdateChecker crea un UpdateChecker contra los releases de
// un repositorio publico de GitHub.
func NewGitHubUpdateChecker(repo string) domain.UpdateChecker {
	return &githubUpdateChecker{
		repo:   repo,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func (c *githubUpdateChecker) Check(currentVersion string) (models.UpdateInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return models.UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.client.Do(req)
	if err != nil {
		return models.UpdateInfo{}, fmt.Errorf("no se pudo consultar GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.UpdateInfo{}, fmt.Errorf("GitHub respondio %d consultando el ultimo release", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return models.UpdateInfo{}, fmt.Errorf("no se pudo leer la respuesta de GitHub: %w", err)
	}

	return models.UpdateInfo{
		Available:      isNewerVersion(currentVersion, release.TagName),
		CurrentVersion: currentVersion,
		LatestVersion:  release.TagName,
		URL:            release.HTMLURL,
	}, nil
}

// isNewerVersion compara dos versiones "vX.Y.Z" (el formato que usan
// los workflows de release; ver -ldflags "-X main.version=..."),
// ignorando cualquier sufijo tipo "-rc1". Si alguna de las dos no se
// puede interpretar (p.ej. version == "dev", un build local fuera de
// un release) se asume que no hay actualizacion que ofrecer, en vez de
// arriesgar un falso positivo.
func isNewerVersion(current, latest string) bool {
	curr, ok := parseVersion(current)
	if !ok {
		return false
	}
	lat, ok := parseVersion(latest)
	if !ok {
		return false
	}
	for i := range curr {
		if lat[i] != curr[i] {
			return lat[i] > curr[i]
		}
	}
	return false
}

func parseVersion(v string) (parts [3]int, ok bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-") // corta sufijos tipo "-rc1" antes de parsear
	segments := strings.Split(v, ".")
	if len(segments) == 0 || segments[0] == "" {
		return parts, false
	}
	for i := 0; i < len(parts) && i < len(segments); i++ {
		n, err := strconv.Atoi(segments[i])
		if err != nil {
			return parts, false
		}
		parts[i] = n
	}
	return parts, true
}
