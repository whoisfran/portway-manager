// Package application contiene los casos de uso: coordinan las
// entidades y los puertos de domain para cumplir lo que App necesita,
// sin saber nada de Wails, AWS ni del sistema de archivos.
package application

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"ssm-portway/internal/domain"
	"ssm-portway/models"
)

// TunnelService orquesta el ciclo de vida de los tuneles de
// port-forwarding: valida las solicitudes, delega el arranque del
// proceso a un domain.SessionRunner, mantiene el registro de tuneles
// activos y notifica su estado mediante un domain.EventPublisher.
type TunnelService interface {
	SetContext(ctx context.Context)
	Start(req models.TunnelRequest) (*models.Tunnel, error)
	Stop(id string) error
	StopAll()
	List() []*models.Tunnel
	// CheckPort indica si un puerto local esta disponible, sin
	// reservarlo. Pensado para que el frontend valide el puerto
	// mientras el usuario llena el formulario; Start sigue siendo quien
	// hace el chequeo autoritativo y atomico al arrancar de verdad.
	CheckPort(localPort int) models.PortStatus
}

type tunnelManager struct {
	mu          sync.Mutex
	tunnels     map[string]*models.Tunnel
	sessions    map[string]domain.RunningSession
	runner      domain.SessionRunner
	publisher   domain.EventPublisher
	portChecker domain.PortAvailabilityChecker
}

// NewTunnelService crea un TunnelService a partir del runner que
// abrira los procesos de sesion SSM, el publisher que notificara sus
// cambios de estado, y el checker que valida que el puerto local no
// este ya ocupado antes de intentar abrir el tunel.
func NewTunnelService(
	runner domain.SessionRunner,
	publisher domain.EventPublisher,
	portChecker domain.PortAvailabilityChecker,
) TunnelService {
	return &tunnelManager{
		tunnels:     make(map[string]*models.Tunnel),
		sessions:    make(map[string]domain.RunningSession),
		runner:      runner,
		publisher:   publisher,
		portChecker: portChecker,
	}
}

func (m *tunnelManager) SetContext(ctx context.Context) {
	m.publisher.SetContext(ctx)
}

// Start valida la solicitud, comprueba que el puerto local no choque
// con otro tunel ya activo ni con algun proceso del sistema, lanza la
// sesion SSM correspondiente y registra el tunel para poder
// consultarlo/detenerlo mas adelante.
func (m *tunnelManager) Start(req models.TunnelRequest) (*models.Tunnel, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	tunnel := &models.Tunnel{
		ID:        uuid.NewString(),
		Request:   req,
		Status:    "starting",
		StartedAt: time.Now(),
	}

	m.mu.Lock()
	if conflict := m.portInUseLocked(req.LocalPort); conflict != nil {
		m.mu.Unlock()
		return nil, fmt.Errorf(
			"el puerto local %d ya esta en uso por el tunel hacia %q",
			req.LocalPort, conflict.Request.InstanceLabel,
		)
	}
	// Se reserva el puerto de inmediato (antes de soltar el mutex) para
	// que dos Start() concurrentes con el mismo puerto no pasen ambos
	// el chequeo anterior.
	m.tunnels[tunnel.ID] = tunnel
	m.mu.Unlock()

	if !m.portChecker.IsAvailable(req.LocalPort) {
		m.mu.Lock()
		delete(m.tunnels, tunnel.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("el puerto local %d ya esta en uso por otro proceso del sistema", req.LocalPort)
	}

	session, err := m.runner.Start(req)
	if err != nil {
		m.mu.Lock()
		delete(m.tunnels, tunnel.ID)
		m.mu.Unlock()

		tunnel.Status = "error"
		tunnel.Message = err.Error()
		return tunnel, fmt.Errorf("no se pudo iniciar el tunel: %w", err)
	}

	m.mu.Lock()
	m.sessions[tunnel.ID] = session
	snapshot := *tunnel
	m.mu.Unlock()

	go m.pump(tunnel.ID, session.Stdout())
	go m.pump(tunnel.ID, session.Stderr())
	go m.await(tunnel.ID, session)

	return &snapshot, nil
}

// portInUseLocked busca un tunel activo que ya use el puerto local
// indicado. Debe llamarse con m.mu ya tomado.
func (m *tunnelManager) portInUseLocked(localPort int) *models.Tunnel {
	for _, t := range m.tunnels {
		if t.Request.LocalPort == localPort {
			snapshot := *t
			return &snapshot
		}
	}
	return nil
}

// CheckPort es la version de solo lectura de las mismas dos
// validaciones que hace Start: primero contra los tuneles activos de
// esta app (conflicto "blando", el usuario podria simplemente detener
// ese tunel) y despues contra el sistema operativo (conflicto
// "duro", hay que elegir otro puerto). No reserva nada, asi que puede
// llamarse tantas veces como el usuario cambie el puerto en el
// formulario sin efectos secundarios.
func (m *tunnelManager) CheckPort(localPort int) models.PortStatus {
	m.mu.Lock()
	conflict := m.portInUseLocked(localPort)
	m.mu.Unlock()

	if conflict != nil {
		return models.PortStatus{
			Available:      false,
			InUseBySameApp: true,
			ConflictLabel:  conflict.Request.InstanceLabel,
		}
	}

	if !m.portChecker.IsAvailable(localPort) {
		return models.PortStatus{Available: false, InUseBySameApp: false}
	}

	return models.PortStatus{Available: true}
}

// pump lee la salida del proceso linea por linea, la reenvia como
// evento de log y marca el tunel como "running" en cuanto llega la
// primera linea.
func (m *tunnelManager) pump(id string, pipe io.Reader) {
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		m.markRunning(id)
		m.publisher.Publish("tunnel:log", map[string]string{
			"id":   id,
			"line": scanner.Text(),
		})
	}

	if err := scanner.Err(); err != nil {
		m.markError(id, err)
	}
}

// markRunning realiza la transicion starting -> running una sola vez,
// protegida por el mutex para evitar condiciones de carrera con Stop
// y con la goroutine que espera la salida del proceso.
func (m *tunnelManager) markRunning(id string) {
	m.mu.Lock()
	t, ok := m.tunnels[id]
	if !ok || t.Status != "starting" {
		m.mu.Unlock()
		return
	}
	t.Status = "running"
	snapshot := *t
	m.mu.Unlock()

	m.publisher.Publish("tunnel:status", &snapshot)
}

func (m *tunnelManager) markError(id string, err error) {
	m.mu.Lock()
	t, ok := m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	t.Status = "error"
	t.Message = err.Error()
	snapshot := *t
	m.mu.Unlock()

	m.publisher.Publish("tunnel:status", &snapshot)
	m.publisher.Publish("tunnel:log", map[string]string{
		"id":   id,
		"line": fmt.Sprintf("error leyendo salida del proceso: %v", err),
	})
}

// await espera a que el proceso termine por su cuenta (crash, cierre
// remoto, etc). Si el tunel ya no esta registrado es porque Stop ya lo
// finalizo explicitamente, y no hay nada mas que hacer.
func (m *tunnelManager) await(id string, session domain.RunningSession) {
	err := session.Wait()

	m.mu.Lock()
	tunnel, ok := m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	if err != nil {
		tunnel.Status = "error"
		tunnel.Message = err.Error()
	} else {
		tunnel.Status = "stopped"
	}
	delete(m.tunnels, id)
	delete(m.sessions, id)
	snapshot := *tunnel
	m.mu.Unlock()

	m.publisher.Publish("tunnel:status", &snapshot)
}

// Stop marca el tunel como detenido y lo retira del registro antes de
// matar el proceso: asi, si el proceso muere justo en ese instante,
// la goroutine de await (ver arriba) lo encuentra ya ausente del mapa
// y no compite por reportar un estado distinto ("error" en vez de
// "stopped") para el mismo tunel.
func (m *tunnelManager) Stop(id string) error {
	m.mu.Lock()
	session, sessionOK := m.sessions[id]
	tunnel, tunnelOK := m.tunnels[id]
	if !sessionOK || !tunnelOK {
		m.mu.Unlock()
		return fmt.Errorf("tunel no encontrado")
	}

	tunnel.Status = "stopped"
	delete(m.tunnels, id)
	delete(m.sessions, id)
	snapshot := *tunnel
	m.mu.Unlock()

	killErr := session.Kill()
	m.publisher.Publish("tunnel:status", &snapshot)
	return killErr
}

// StopAll termina todos los tuneles activos; se usa al cerrar la app.
func (m *tunnelManager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.tunnels))
	for id := range m.tunnels {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.Stop(id)
	}
}

// List devuelve una copia del estado actual de cada tunel activo.
func (m *tunnelManager) List() []*models.Tunnel {
	m.mu.Lock()
	defer m.mu.Unlock()

	list := make([]*models.Tunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		snapshot := *t
		list = append(list, &snapshot)
	}
	return list
}
