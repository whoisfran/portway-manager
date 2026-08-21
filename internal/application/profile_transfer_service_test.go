package application

import (
	"testing"

	"portway-manager/models"
)

type fakeProfileService struct {
	saved []models.Favorite
}

func (f *fakeProfileService) List() ([]models.Favorite, error) {
	return f.saved, nil
}

func (f *fakeProfileService) Get(id string) (models.Favorite, error) {
	for _, p := range f.saved {
		if p.ID == id {
			return p, nil
		}
	}
	return models.Favorite{}, nil
}

func (f *fakeProfileService) Save(profile models.Favorite) (models.Favorite, error) {
	f.saved = append(f.saved, profile)
	return profile, nil
}

func (f *fakeProfileService) Delete(string) error { return nil }

func TestImportBlanksAwsProfileAndDisambiguatesLabels(t *testing.T) {
	fake := &fakeProfileService{saved: []models.Favorite{
		{Label: "DB producción", Profile: "mi-perfil-local"},
	}}
	transfer := NewProfileTransferService(fake)

	result, err := transfer.Import(models.ProfileExport{
		Version: models.ProfileExportVersion,
		Profiles: []models.ExportedProfile{
			{Label: "DB producción", InstanceID: "i-123", LocalPort: 5432, RemotePort: 5432},
		},
	})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.ImportedCount != 1 {
		t.Fatalf("ImportedCount = %d, want 1", result.ImportedCount)
	}
	if len(result.Failures) != 0 {
		t.Fatalf("Failures = %v, want none", result.Failures)
	}

	imported := fake.saved[len(fake.saved)-1]
	if imported.Label != "DB producción (2)" {
		t.Errorf("Label = %q, want disambiguated label", imported.Label)
	}
	if imported.Profile != "" {
		t.Errorf("Profile = %q, want empty (el perfil de AWS local no debe importarse)", imported.Profile)
	}
}

func TestImportRejectsUnsupportedVersion(t *testing.T) {
	transfer := NewProfileTransferService(&fakeProfileService{})

	_, err := transfer.Import(models.ProfileExport{Version: 999})
	if err == nil {
		t.Fatal("Import() error = nil, want error for unsupported version")
	}
}

func TestImportResultNeverHasNilFailures(t *testing.T) {
	transfer := NewProfileTransferService(&fakeProfileService{})

	result, err := transfer.Import(models.ProfileExport{Version: models.ProfileExportVersion})
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.Failures == nil {
		t.Error("Failures = nil, want non-nil empty slice (se serializa como null y rompe el tipo TS generado)")
	}
}
