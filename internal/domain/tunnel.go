// Package domain define las reglas y los puertos (interfaces) que la
// aplicacion necesita de su entorno. No depende de AWS, Wails ni de
// ningun detalle de infraestructura: eso lo implementan los adaptadores
// en internal/infrastructure.
package domain

import (
	"context"
	"io"

	"portway-manager/models"
)

// RunningSession representa un tunel de port-forwarding en ejecucion,
// sin importar si por debajo es un proceso "aws ssm start-session" o
// una conexion SSH nativa: ambos exponen el mismo contrato para que
// TunnelService pueda tratarlos por igual.
type RunningSession interface {
	Stdout() io.Reader
	Stderr() io.Reader
	// Wait bloquea hasta que la sesion termina y devuelve su resultado.
	Wait() error
	// Kill termina la sesion de forma forzosa.
	Kill() error
}

// SessionRunner sabe validar y arrancar sesiones de port-forwarding
// para un tipo de tunel especifico.
type SessionRunner interface {
	ValidateRequest(req models.TunnelRequest) error
	Start(req models.TunnelRequest) (RunningSession, error)
}

// TunnelStrategy agrupa todo lo que un tipo de tunel (SSM o SSH)
// necesita: como validar el Favorite guardado con ese tipo, como
// derivar de el un TunnelRequest, y como validar/arrancar la sesion
// correspondiente. Cada tipo vive en su propio paquete de
// infraestructura (ver internal/infrastructure/ssm y
// internal/infrastructure/ssh) implementando esta misma interfaz.
type TunnelStrategy interface {
	Type() models.FavoriteType
	// ValidateFavorite comprueba que el perfil tenga los campos
	// obligatorios para este tipo (p.ej. InstanceID en SSM, o
	// Host/User/credenciales en SSH).
	ValidateFavorite(models.Favorite) error
	// BuildRequest deriva el TunnelRequest a partir de un Favorite ya
	// validado con este mismo tipo.
	BuildRequest(models.Favorite) models.TunnelRequest

	SessionRunner
}

// TunnelStrategyRegistry resuelve el TunnelStrategy adecuado segun el
// FavoriteType de un perfil o solicitud de tunel.
type TunnelStrategyRegistry interface {
	Strategy(t models.FavoriteType) (TunnelStrategy, error)
}

// EventPublisher notifica cambios de estado y logs de los tuneles
// hacia el exterior (en la practica, el frontend a traves de eventos
// de Wails).
type EventPublisher interface {
	SetContext(ctx context.Context)
	Publish(event string, payload any)
}

// PortAvailabilityChecker verifica si un puerto TCP local esta libre
// para escuchar antes de lanzar la sesion de tunel, de forma que un
// puerto ya ocupado por otro proceso del sistema (ajeno a esta app) se
// reporte con un error claro en vez de dejar que falle con un mensaje
// confuso.
type PortAvailabilityChecker interface {
	IsAvailable(port int) bool
}
