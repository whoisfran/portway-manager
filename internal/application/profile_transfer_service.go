package application

import (
	"fmt"

	"ssm-portway/models"
)

// ProfileTransferService convierte entre los perfiles guardados y el
// formato portable que se exporta/importa. Deliberadamente nunca
// incluye el campo Profile (el nombre del perfil de AWS local): es
// especifico del equipo de cada usuario, y quien importa debe elegir
// el suyo.
type ProfileTransferService interface {
	Export() (models.ProfileExport, error)
	Import(export models.ProfileExport) (models.ImportResult, error)
}

type profileTransferService struct {
	profiles ProfileService
}

func NewProfileTransferService(profiles ProfileService) ProfileTransferService {
	return &profileTransferService{profiles: profiles}
}

func (s *profileTransferService) Export() (models.ProfileExport, error) {
	saved, err := s.profiles.List()
	if err != nil {
		return models.ProfileExport{}, err
	}

	items := make([]models.ExportedProfile, 0, len(saved))
	for _, p := range saved {
		items = append(items, models.ExportedProfile{
			Label:         p.Label,
			Region:        p.Region,
			InstanceID:    p.InstanceID,
			InstanceLabel: p.InstanceLabel,
			LocalPort:     p.LocalPort,
			RemotePort:    p.RemotePort,
			RemoteHost:    p.RemoteHost,
		})
	}

	return models.ProfileExport{Version: models.ProfileExportVersion, Profiles: items}, nil
}

// Import agrega los perfiles del archivo a la lista existente (nunca
// la reemplaza). Un perfil invalido no aborta el resto: se reporta en
// Failures para que el usuario decida que hacer con el.
func (s *profileTransferService) Import(export models.ProfileExport) (models.ImportResult, error) {
	if export.Version != models.ProfileExportVersion {
		return models.ImportResult{}, fmt.Errorf("version de exportacion no soportada: %d", export.Version)
	}

	existingLabels, err := s.existingLabels()
	if err != nil {
		return models.ImportResult{}, err
	}

	result := models.ImportResult{Failures: []models.ImportFailure{}}
	for _, item := range export.Profiles {
		label := disambiguate(item.Label, existingLabels)

		_, err := s.profiles.Save(models.Favorite{
			Label:         label,
			Region:        item.Region,
			InstanceID:    item.InstanceID,
			InstanceLabel: item.InstanceLabel,
			LocalPort:     item.LocalPort,
			RemotePort:    item.RemotePort,
			RemoteHost:    item.RemoteHost,
			// Profile queda vacio a proposito: el usuario debe elegir
			// cual de sus perfiles de AWS locales usar para esta conexion.
		})
		if err != nil {
			result.Failures = append(result.Failures, models.ImportFailure{Label: item.Label, Reason: err.Error()})
			continue
		}

		existingLabels[label] = true
		result.ImportedCount++
	}

	return result, nil
}

func (s *profileTransferService) existingLabels() (map[string]bool, error) {
	saved, err := s.profiles.List()
	if err != nil {
		return nil, err
	}

	labels := make(map[string]bool, len(saved))
	for _, p := range saved {
		labels[p.Label] = true
	}
	return labels, nil
}

// disambiguate evita perfiles con el mismo nombre tras importar (p.ej.
// si ya existe uno propio llamado igual, o si el mismo archivo se
// importa dos veces).
func disambiguate(label string, existing map[string]bool) string {
	if !existing[label] {
		return label
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)", label, i)
		if !existing[candidate] {
			return candidate
		}
	}
}
