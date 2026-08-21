// Package domain define las reglas y los puertos (interfaces) que la
// aplicacion necesita de su entorno. No depende de AWS, Wails ni de
// ningun detalle de infraestructura: eso lo implementan los adaptadores
// en internal/infrastructure.
package domain

import (
	"context"
	"io"

	"ssm-portway/models"
)

// RunningSession representa un proceso de sesion SSM en ejecucion.
type RunningSession interface {
	Stdout() io.Reader
	Stderr() io.Reader
	// Wait bloquea hasta que el proceso termina y devuelve su resultado.
	Wait() error
	// Kill termina el proceso (y sus hijos) de forma forzosa.
	Kill() error
}

// SessionRunner sabe iniciar sesiones de port-forwarding via SSM.
type SessionRunner interface {
	Start(req models.TunnelRequest) (RunningSession, error)
}

// EventPublisher notifica cambios de estado y logs de los tuneles
// hacia el exterior (en la practica, el frontend a traves de eventos
// de Wails).
type EventPublisher interface {
	SetContext(ctx context.Context)
	Publish(event string, payload any)
}

// PortAvailabilityChecker verifica si un puerto TCP local esta libre
// para escuchar antes de lanzar el proceso de sesion SSM, de forma
// que un puerto ya ocupado por otro proceso del sistema (ajeno a esta
// app) se reporte con un error claro en vez de dejar que falle el
// CLI de AWS con un mensaje confuso.
type PortAvailabilityChecker interface {
	IsAvailable(port int) bool
}
