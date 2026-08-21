package main

import (
	"context"
	"embed"
	"log"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"ssm-portway/internal/application"
	"ssm-portway/internal/infrastructure"
	"ssm-portway/internal/infrastructure/ssh"
	"ssm-portway/internal/infrastructure/ssm"
)

//go:embed all:frontend/dist
var assets embed.FS

// version se sobreescribe en tiempo de build (-ldflags "-X main.version=v1.2.3")
// desde los workflows de release; en desarrollo queda como "dev".
var version = "dev"

func main() {
	log.Printf("SSM Portway %s", version)

	// Debe llamarse antes que cualquier cosa relacionada con Wails: GTK
	// se inicializa dentro de wails.Run, y para entonces ya es tarde
	// para fijar GTK_THEME (ver titlebar_linux.go).
	ensureDarkTitleBar()

	// Composition root: aqui, y solo aqui, se conectan los adaptadores
	// de infraestructura con los casos de uso de application. Nada
	// mas en el proyecto conoce estos tipos concretos.
	profileStore, err := infrastructure.NewJSONProfileStore()
	if err != nil {
		log.Fatalf("no se pudo inicializar el almacen de perfiles: %v", err)
	}

	tunnelStrategies := infrastructure.NewTunnelStrategyRegistry(
		ssm.NewTunnelStrategy(),
		ssh.NewTunnelStrategy(),
	)

	eventPublisher := infrastructure.NewWailsEventPublisher()
	portChecker := infrastructure.NewTCPPortAvailabilityChecker()
	instanceLister := ssm.NewAWSInstanceLister()
	prerequisitesChecker := ssm.NewPrerequisitesChecker()
	awsProfileLister := infrastructure.NewLocalAWSProfileLister()
	profileExportGateway := infrastructure.NewJSONProfileExportGateway()

	profileService := application.NewProfileService(profileStore, tunnelStrategies)

	app := NewApp(
		application.NewTunnelService(tunnelStrategies, eventPublisher, portChecker),
		profileService,
		application.NewProfileTransferService(profileService),
		profileExportGateway,
		prerequisitesChecker,
		awsProfileLister,
		instanceLister,
		tunnelStrategies,
	)

	// El icono de la bandeja del sistema corre en su propio bucle nativo
	// (vease RunWithExternalLoop: pensado justo para convivir con el
	// bucle de otro toolkit grafico, en este caso el webview de Wails),
	// asi que se arranca antes de wails.Run y se cierra despues de que
	// este retorne, cuando la app ya termino de cerrarse por completo.
	trayStart, trayEnd := systray.RunWithExternalLoop(func() { setupTray(app) }, func() {})
	trayStart()

	err = wails.Run(&options.App{
		Title:  "SSM Portway",
		Width:  1000,
		Height: 680,
		// La app esta pensada como un widget de escritorio pequeno, no
		// una ventana de pantalla completa: tamano fijo en vez de
		// libremente redimensionable.
		DisableResize: true,
		// Sin marco nativo: la barra de titulo (icono, titulo, minimizar/
		// maximizar/cerrar) la dibuja el frontend (ver TitleBar.vue). Esto
		// tambien vuelve innecesario el theming nativo de Windows/GTK que
		// se ajustaba antes (ya no hay barra de titulo nativa que colorear).
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Ventana completamente transparente: las esquinas redondeadas las
		// dibuja el CSS (ver #app en main.css), no el sistema operativo.
		// macOS y Windows 11 redondean solos las ventanas normales, pero
		// eso depende de cada OS/version -- Linux no lo hace nunca. Al
		// pintar el redondeado nosotros mismos se ve igual en los tres,
		// sin depender de lo que cada uno decida hacer por su cuenta.
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)

			setAppContext(ctx)
			// Las notificaciones de sistema son un extra, no algo de lo
			// que dependa poder usar la app: si el entorno no tiene un
			// servicio de notificaciones disponible (p.ej. sin D-Bus, o
			// alguna sesion minima), esto no debe tumbar la app entera.
			if err := runtime.InitializeNotifications(ctx); err != nil {
				log.Printf("no se pudo inicializar el servicio de notificaciones: %v", err)
			}
			watchTunnelStatus(ctx, app)
			go watchMinimize(ctx)
		},
		OnShutdown: app.shutdown,
		Bind: []any{
			app,
		},
		// Red de seguridad para cuando el icono de la bandeja no logra
		// registrarse (p.ej. GNOME sin la extension "AppIndicator and
		// KStatusNotifierItem Support": ahi el registro falla en
		// silencio, sin ninguna forma de detectarlo desde el codigo, y
		// la ventana oculta se queda sin nadie que la reabra). Si el
		// usuario intenta abrir la app de nuevo mientras ya hay una
		// instancia corriendo, en vez de una segunda ventana se
		// muestra la que ya estaba oculta.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "ssm-portway-3f2504e0-4f89-11d3-9a0c-0305e82c3301",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				showWindow(getAppContext())
			},
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
		},
		Windows: &windows.Options{
			WindowIsTranslucent: true,
		},
		Mac: &mac.Options{
			WindowIsTranslucent: true,
		},
	})

	trayEnd()

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
