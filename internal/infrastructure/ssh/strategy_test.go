package ssh

import (
	"testing"

	"portway-manager/models"
)

func validPasswordFavorite() models.Favorite {
	return models.Favorite{
		Label:      "Bastion produccion",
		Type:       models.FavoriteTypeSSH,
		Host:       "bastion.example.com",
		User:       "deploy",
		AuthMethod: models.SSHAuthMethodPassword,
		Password:   "s3cr3t",
		LocalPort:  5432,
		RemotePort: 5432,
	}
}

func TestValidateFavoriteRequiresHostAndUser(t *testing.T) {
	s := NewTunnelStrategy()

	withoutHost := validPasswordFavorite()
	withoutHost.Host = ""
	if err := s.ValidateFavorite(withoutHost); err == nil {
		t.Error("ValidateFavorite() error = nil, want error por falta de Host")
	}

	withoutUser := validPasswordFavorite()
	withoutUser.User = ""
	if err := s.ValidateFavorite(withoutUser); err == nil {
		t.Error("ValidateFavorite() error = nil, want error por falta de User")
	}
}

func TestValidateFavoritePasswordAuthRequiresPassword(t *testing.T) {
	s := NewTunnelStrategy()

	fav := validPasswordFavorite()
	fav.Password = ""
	if err := s.ValidateFavorite(fav); err == nil {
		t.Error("ValidateFavorite() error = nil, want error por falta de Password")
	}
}

func TestValidateFavoritePrivateKeyAuthRequiresPath(t *testing.T) {
	s := NewTunnelStrategy()

	fav := validPasswordFavorite()
	fav.AuthMethod = models.SSHAuthMethodPrivateKey
	fav.Password = ""
	if err := s.ValidateFavorite(fav); err == nil {
		t.Error("ValidateFavorite() error = nil, want error por falta de PrivateKeyPath")
	}

	fav.PrivateKeyPath = "/home/user/.ssh/id_rsa"
	if err := s.ValidateFavorite(fav); err != nil {
		t.Errorf("ValidateFavorite() error = %v, want nil", err)
	}
}

func TestValidateFavoriteRejectsUnknownAuthMethod(t *testing.T) {
	s := NewTunnelStrategy()

	fav := validPasswordFavorite()
	fav.AuthMethod = "otro"
	if err := s.ValidateFavorite(fav); err == nil {
		t.Error("ValidateFavorite() error = nil, want error por metodo de auth invalido")
	}
}

func TestBuildRequestDefaultsPortTo22(t *testing.T) {
	s := NewTunnelStrategy()
	fav := validPasswordFavorite()
	fav.Port = 0

	req := s.BuildRequest(fav)
	if req.Port != defaultSSHPort {
		t.Errorf("BuildRequest().Port = %d, want %d", req.Port, defaultSSHPort)
	}
}

func TestBuildRequestKeepsExplicitPort(t *testing.T) {
	s := NewTunnelStrategy()
	fav := validPasswordFavorite()
	fav.Port = 2222

	req := s.BuildRequest(fav)
	if req.Port != 2222 {
		t.Errorf("BuildRequest().Port = %d, want 2222", req.Port)
	}
}
