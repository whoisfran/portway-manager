package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"portway-manager/internal/application"
	"portway-manager/internal/domain"
	"portway-manager/models"
)

// App es el adaptador que Wails expone al frontend. No contiene
// logica de negocio: cada metodo traduce una llamada del frontend en
// una llamada al puerto correspondiente de application/domain.
type App struct {
	ctx context.Context

	tunnelService          application.TunnelService
	profileService         application.ProfileService
	profileTransferService application.ProfileTransferService
	profileExportGateway   domain.ProfileExportGateway
	prerequisitesChecker   domain.PrerequisitesChecker
	awsProfileLister       domain.AWSProfileLister
	instanceLister         domain.InstanceLister
	tunnelStrategies       domain.TunnelStrategyRegistry
}

func NewApp(
	tunnelService application.TunnelService,
	profileService application.ProfileService,
	profileTransferService application.ProfileTransferService,
	profileExportGateway domain.ProfileExportGateway,
	prerequisitesChecker domain.PrerequisitesChecker,
	awsProfileLister domain.AWSProfileLister,
	instanceLister domain.InstanceLister,
	tunnelStrategies domain.TunnelStrategyRegistry,
) *App {
	return &App{
		tunnelService:          tunnelService,
		profileService:         profileService,
		profileTransferService: profileTransferService,
		profileExportGateway:   profileExportGateway,
		prerequisitesChecker:   prerequisitesChecker,
		awsProfileLister:       awsProfileLister,
		instanceLister:         instanceLister,
		tunnelStrategies:       tunnelStrategies,
	}
}

// startup se ejecuta cuando Wails termina de inicializar la ventana.
// Guardamos el contexto para poder emitir eventos y abrir el
// navegador mas adelante.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.tunnelService.SetContext(ctx)
}

// shutdown se ejecuta al cerrar la app: detiene los tuneles activos
// para no dejar procesos "aws ssm start-session" huerfanos.
func (a *App) shutdown(_ context.Context) {
	a.tunnelService.StopAll()
	close(trayDone)
}

// HideToTray oculta la ventana principal en la bandeja del sistema en
// vez de minimizarla de verdad: el proceso sigue corriendo (los
// tuneles activos no se interrumpen) y la ventana se recupera dando
// clic en el icono de la bandeja o en "Abrir" de su menu (ver tray.go).
// La usa el boton de minimizar del titlebar propio (ver TitleBar.vue),
// ya que al no haber marco nativo (Frameless) minimizar de verdad no
// tendria forma de deshacerse sin un titlebar visible.
func (a *App) HideToTray() {
	hideWindow(a.ctx)
}

func (a *App) CheckPrerequisites() models.Prerequisites {
	return a.prerequisitesChecker.Check()
}

// ---------- Perfiles y regiones de AWS ----------

func (a *App) ListAwsProfiles() ([]string, error) {
	return a.awsProfileLister.ListProfiles()
}

func (a *App) ListAwsRegions() []string {
	return domain.SupportedRegions
}

// GetAwsDefaults permite al frontend preseleccionar el perfil/region
// de un formulario nuevo con lo que la AWS CLI usaria por defecto.
func (a *App) GetAwsDefaults() models.AWSDefaults {
	return a.awsProfileLister.Defaults()
}

// GetAwsAuthMethod describe como se autentica un perfil de AWS local
// (SSO, rol asumido, claves de acceso, etc.), solo informativo para
// el detalle de una conexion.
func (a *App) GetAwsAuthMethod(profile string) string {
	return a.awsProfileLister.AuthMethod(profile)
}

// ---------- Instancias EC2/SSM ----------

func (a *App) ListManagedInstances(profile, region string) ([]models.Instance, error) {
	return a.instanceLister.List(a.ctx, profile, region)
}

// ---------- Tuneles ----------

// StartTunnel arranca un tunel a partir de un perfil de conexion ya
// guardado: no existe una conexion "rapida" sin guardar antes, asi
// que el frontend solo manda el ID del perfil (ver ConnectionProfile
// en el frontend) y aqui se construye el TunnelRequest a partir de el
// -- el perfil guardado es la unica fuente de verdad para esos datos,
// en vez de que el frontend tenga que reenviarlos por su cuenta.
func (a *App) StartTunnel(favoriteID string) (*models.Tunnel, error) {
	favorite, err := a.profileService.Get(favoriteID)
	if err != nil {
		return nil, err
	}

	strategy, err := a.tunnelStrategies.Strategy(favorite.Type)
	if err != nil {
		return nil, err
	}

	return a.tunnelService.Start(strategy.BuildRequest(favorite))
}

func (a *App) StopTunnel(id string) error {
	return a.tunnelService.Stop(id)
}

func (a *App) ListActiveTunnels() []*models.Tunnel {
	return a.tunnelService.List()
}

// CheckLocalPort permite al frontend validar un puerto mientras el
// usuario llena el formulario, antes de intentar iniciar el tunel.
// PortStatus.InUseBySameApp distingue si conviene solo un aviso (otro
// tunel de esta app ya usa el puerto, y el usuario podria detenerlo) o
// si hay que pedirle que elija otro puerto (lo ocupa el sistema).
func (a *App) CheckLocalPort(port int) models.PortStatus {
	return a.tunnelService.CheckPort(port)
}

// ---------- Perfiles de conexion guardados ----------

func (a *App) ListFavorites() ([]models.Favorite, error) {
	return a.profileService.List()
}

func (a *App) SaveFavorite(fav models.Favorite) (models.Favorite, error) {
	return a.profileService.Save(fav)
}

func (a *App) DeleteFavorite(id string) error {
	return a.profileService.Delete(id)
}

// ExportFavorites deja que el usuario elija donde guardar sus
// perfiles de conexion, para compartirlos con alguien mas. El archivo
// nunca incluye el perfil de AWS local de cada uno (ver
// ProfileTransferService): quien lo importe debera elegir el suyo.
// Devuelve una ruta vacia (sin error) si el usuario cierra el dialogo
// sin elegir nada.
func (a *App) ExportFavorites() (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Exportar perfiles de conexión",
		DefaultFilename: "portway-manager-perfiles.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}

	export, err := a.profileTransferService.Export()
	if err != nil {
		return "", err
	}
	if err := a.profileExportGateway.Save(path, export); err != nil {
		return "", err
	}
	return path, nil
}

// ImportFavorites deja que el usuario elija un archivo exportado
// previamente y agrega esos perfiles a su lista (nunca la reemplaza).
// Devuelve un ImportResult vacio (sin error) si el usuario cierra el
// dialogo sin elegir nada.
func (a *App) ImportFavorites() (models.ImportResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Importar perfiles de conexión",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || path == "" {
		return models.ImportResult{}, err
	}

	export, err := a.profileExportGateway.Load(path)
	if err != nil {
		return models.ImportResult{}, err
	}
	return a.profileTransferService.Import(export)
}

// ---------- Documentacion ----------

// OpenAwsInstallDocs abre en el navegador del usuario la
// documentacion de instalacion de AWS CLI y del Session Manager
// Plugin, usadas cuando CheckPrerequisites reporta que falta alguno.
func (a *App) OpenAwsInstallDocs() {
	for _, url := range domain.PrerequisitesDocs {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}
