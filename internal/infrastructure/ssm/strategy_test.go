package ssm

import (
	"testing"

	"portway-manager/models"
)

func validFavorite() models.Favorite {
	return models.Favorite{
		Label:      "DB produccion",
		Type:       models.FavoriteTypeSSM,
		InstanceID: "i-0123456789abcdef0",
		LocalPort:  5432,
		RemotePort: 5432,
	}
}

func TestValidateFavoriteRequiresInstanceID(t *testing.T) {
	s := NewTunnelStrategy()

	fav := validFavorite()
	fav.InstanceID = ""
	if err := s.ValidateFavorite(fav); err == nil {
		t.Fatal("ValidateFavorite() error = nil, want error por falta de InstanceID")
	}

	if err := s.ValidateFavorite(validFavorite()); err != nil {
		t.Errorf("ValidateFavorite() error = %v, want nil", err)
	}
}

func TestValidateRequestRequiresInstanceID(t *testing.T) {
	s := NewTunnelStrategy()
	req := s.BuildRequest(validFavorite())

	req.InstanceID = ""
	if err := s.ValidateRequest(req); err == nil {
		t.Fatal("ValidateRequest() error = nil, want error por falta de InstanceID")
	}
}

func TestBuildRequestCopiesFields(t *testing.T) {
	s := NewTunnelStrategy()
	fav := validFavorite()
	fav.ID = "fav-1"
	fav.Profile = "mi-perfil"
	fav.Region = "us-east-1"
	fav.RemoteHost = "db.internal"

	req := s.BuildRequest(fav)

	if req.FavoriteID != fav.ID || req.Type != models.FavoriteTypeSSM {
		t.Errorf("BuildRequest() = %+v, want FavoriteID/Type copiados", req)
	}
	if req.InstanceID != fav.InstanceID || req.Profile != fav.Profile || req.Region != fav.Region {
		t.Errorf("BuildRequest() = %+v, want campos SSM copiados", req)
	}
	if req.RemoteHost != fav.RemoteHost || req.LocalPort != fav.LocalPort || req.RemotePort != fav.RemotePort {
		t.Errorf("BuildRequest() = %+v, want puertos/host copiados", req)
	}
}
