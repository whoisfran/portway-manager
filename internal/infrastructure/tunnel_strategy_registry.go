package infrastructure

import (
	"fmt"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// tunnelStrategyRegistry resuelve el domain.TunnelStrategy adecuado
// segun el FavoriteType de un perfil o solicitud de tunel. No conoce
// los detalles de SSM ni de SSH: solo indexa las estrategias
// concretas que le pasa el composition root (ver main.go).
type tunnelStrategyRegistry struct {
	strategies map[models.FavoriteType]domain.TunnelStrategy
}

// NewTunnelStrategyRegistry indexa las estrategias recibidas por su
// propio Type(). Un tipo repetido sobreescribe al anterior.
func NewTunnelStrategyRegistry(strategies ...domain.TunnelStrategy) domain.TunnelStrategyRegistry {
	byType := make(map[models.FavoriteType]domain.TunnelStrategy, len(strategies))
	for _, s := range strategies {
		byType[s.Type()] = s
	}
	return &tunnelStrategyRegistry{strategies: byType}
}

func (r *tunnelStrategyRegistry) Strategy(t models.FavoriteType) (domain.TunnelStrategy, error) {
	strategy, ok := r.strategies[t]
	if !ok {
		return nil, fmt.Errorf("tipo de conexion no soportado: %q", t)
	}
	return strategy, nil
}
