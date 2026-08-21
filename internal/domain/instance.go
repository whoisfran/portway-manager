package domain

import (
	"context"

	"portway-manager/models"
)

// InstanceLister consulta las instancias administradas por SSM
// disponibles para un perfil/region de AWS.
type InstanceLister interface {
	List(ctx context.Context, profile, region string) ([]models.Instance, error)
}
