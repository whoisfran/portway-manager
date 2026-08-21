package infrastructure

import (
	"testing"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

type fakeStrategy struct {
	t models.FavoriteType
}

func (f *fakeStrategy) Type() models.FavoriteType              { return f.t }
func (f *fakeStrategy) ValidateFavorite(models.Favorite) error { return nil }
func (f *fakeStrategy) BuildRequest(models.Favorite) models.TunnelRequest {
	return models.TunnelRequest{}
}
func (f *fakeStrategy) ValidateRequest(models.TunnelRequest) error { return nil }
func (f *fakeStrategy) Start(models.TunnelRequest) (domain.RunningSession, error) {
	return nil, nil
}

func TestRegistryResolvesByType(t *testing.T) {
	ssmStrategy := &fakeStrategy{t: models.FavoriteTypeSSM}
	sshStrategy := &fakeStrategy{t: models.FavoriteTypeSSH}
	registry := NewTunnelStrategyRegistry(ssmStrategy, sshStrategy)

	got, err := registry.Strategy(models.FavoriteTypeSSH)
	if err != nil {
		t.Fatalf("Strategy() error = %v", err)
	}
	if got != sshStrategy {
		t.Errorf("Strategy(ssh) = %v, want sshStrategy", got)
	}
}

func TestRegistryErrorsOnUnknownType(t *testing.T) {
	registry := NewTunnelStrategyRegistry(&fakeStrategy{t: models.FavoriteTypeSSM})

	if _, err := registry.Strategy("desconocido"); err == nil {
		t.Fatal("Strategy() error = nil, want error para tipo no registrado")
	}
}
