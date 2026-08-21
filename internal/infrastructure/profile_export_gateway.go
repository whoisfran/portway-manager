package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// jsonProfileExportGateway lee/escribe el archivo de exportacion de
// perfiles en la ruta que el usuario elija (via el dialogo nativo que
// orquesta App), en el mismo formato JSON que ya usa el resto de la
// app.
type jsonProfileExportGateway struct{}

func NewJSONProfileExportGateway() domain.ProfileExportGateway {
	return &jsonProfileExportGateway{}
}

func (g *jsonProfileExportGateway) Save(path string, export models.ProfileExport) error {
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (g *jsonProfileExportGateway) Load(path string) (models.ProfileExport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.ProfileExport{}, err
	}

	var export models.ProfileExport
	if err := json.Unmarshal(data, &export); err != nil {
		return models.ProfileExport{}, fmt.Errorf("el archivo no tiene un formato valido: %w", err)
	}
	return export, nil
}
