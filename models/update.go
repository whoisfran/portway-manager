package models

// UpdateInfo describe si hay una version mas nueva publicada en
// GitHub Releases que la que esta corriendo ahora mismo (ver
// domain.UpdateChecker).
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	// URL es la pagina del release en GitHub; el frontend la abre en
	// el navegador del sistema (ver App.OpenUpdateURL) para que el
	// usuario baje el instalador el mismo -- esta app no se
	// autoactualiza.
	URL string `json:"url"`
}
