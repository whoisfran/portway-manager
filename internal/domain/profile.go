package domain

import "portway-manager/models"

// ProfileRepository persiste los perfiles de conexion guardados por el
// usuario. La validacion de negocio vive en models.Favorite.Validate;
// el repositorio solo se ocupa de la persistencia.
type ProfileRepository interface {
	List() ([]models.Favorite, error)
	Save(models.Favorite) (models.Favorite, error)
	Delete(id string) error
}

// ProfileExportGateway lee/escribe un archivo de exportacion en una
// ruta elegida por el usuario. Es un puerto distinto de
// ProfileRepository porque no es "la" persistencia de la app: es un
// archivo de intercambio arbitrario, en un formato portable.
type ProfileExportGateway interface {
	Save(path string, export models.ProfileExport) error
	Load(path string) (models.ProfileExport, error)
}
