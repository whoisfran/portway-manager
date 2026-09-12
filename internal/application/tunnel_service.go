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

	"portway-manager/internal/domain"
	"portway-manager/models"
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

// defaultMaxReconnectAttempts y defaultReconnectBackoff rigen la
// reconexion automatica (ver await/reconnectLoop) en produccion: unos
// pocos intentos con una espera fija alcanzan para curar un timeout de
// inactividad de SSM o un corte de red pasajero, sin insistir
// indefinidamente contra algo que de verdad dejo de responder
// (instancia apagada, credenciales vencidas, etc). Los tests fijan sus
// propios valores (mas cortos) directamente en los campos del struct,
// en vez de una var de paquete compartida entre goroutines de tests
// distintos.
const (
	defaultMaxReconnectAttempts = 3
	defaultReconnectBackoff     = 5 * time.Second
)

type tunnelManager struct {
	mu                sync.Mutex
	tunnels           map[string]*models.Tunnel
	sessions          map[string]domain.RunningSession
	strategies        domain.TunnelStrategyRegistry
	publisher         domain.EventPublisher
	portChecker       domain.PortAvailabilityChecker
	settings          domain.SettingsRepository
	maxReconnectTries int
	reconnectBackoff  time.Duration
}

// NewTunnelService crea un TunnelService a partir del registro que
// resuelve la estrategia (SSM o SSH) de cada solicitud, el publisher
// que notificara sus cambios de estado, el checker que valida que el
// puerto local no este ya ocupado antes de intentar abrir el tunel, y
// el repositorio de ajustes del que lee si toca reconectar solo un
// tunel que se cayo inesperadamente (ver reconnectLoop).
func NewTunnelService(
	strategies domain.TunnelStrategyRegistry,
	publisher domain.EventPublisher,
	portChecker domain.PortAvailabilityChecker,
	settings domain.SettingsRepository,
) TunnelService {
	return &tunnelManager{
		tunnels:           make(map[string]*models.Tunnel),
		sessions:          make(map[string]domain.RunningSession),
		strategies:        strategies,
		publisher:         publisher,
		portChecker:       portChecker,
		settings:          settings,
		maxReconnectTries: defaultMaxReconnectAttempts,
		reconnectBackoff:  defaultReconnectBackoff,
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

	strategy, err := m.strategies.Strategy(req.Type)
	if err != nil {
		return nil, err
	}
	if err := strategy.ValidateRequest(req); err != nil {
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
			req.LocalPort, conflict.Request.TargetLabel(),
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

	session, err := strategy.Start(req)
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
			ConflictLabel:  conflict.Request.TargetLabel(),
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
//
// Una salida limpia (err == nil) se trata igual que un Stop: la app
// nunca la pidio, pero tampoco hay un error del que reponerse, asi que
// no se reintenta. Una salida con error es la que puede reconectarse
// (ver reconnectLoop) si el usuario lo tiene habilitado para este tipo
// de tunel.
func (m *tunnelManager) await(id string, session domain.RunningSession) {
	for {
		err := session.Wait()

		if err == nil {
			m.mu.Lock()
			tunnel, ok := m.tunnels[id]
			if !ok {
				m.mu.Unlock()
				return
			}
			tunnel.Status = "stopped"
			delete(m.tunnels, id)
			delete(m.sessions, id)
			snapshot := *tunnel
			m.mu.Unlock()

			m.publisher.Publish("tunnel:status", &snapshot)
			return
		}

		newSession, giveUp := m.reconnectLoop(id, err)
		if giveUp {
			return
		}
		session = newSession
		go m.pump(id, session.Stdout())
		go m.pump(id, session.Stderr())
	}
}

// reconnectLoop reintenta abrir el mismo tunel tras una caida
// inesperada: publica "reconnecting" (para que la UI lo distinga de un
// error definitivo), espera reconnectBackoff, y prueba de nuevo hasta
// maxReconnectAttempts. Se rinde -- deja el tunel en "error", como
// antes de que existiera la reconexion -- si el tipo de tunel tiene la
// reconexion apagada, si se agotan los intentos, o si el usuario lo
// detiene mientras tanto (Stop lo quita de m.tunnels; este metodo lo
// nota en la siguiente vuelta y no sigue insistiendo).
//
// Devuelve la sesion nueva si tuvo exito (giveUp=false), para que
// await seiga esperandola igual que a la original.
func (m *tunnelManager) reconnectLoop(id string, lastErr error) (session domain.RunningSession, giveUp bool) {
	m.mu.Lock()
	tunnel, ok := m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return nil, true
	}
	req := tunnel.Request
	m.mu.Unlock()

	for attempt := 1; attempt <= m.maxReconnectTries && m.autoReconnectEnabled(req.Type); attempt++ {
		m.mu.Lock()
		tunnel, ok := m.tunnels[id]
		if !ok {
			m.mu.Unlock()
			return nil, true
		}
		tunnel.Status = "reconnecting"
		tunnel.Message = fmt.Sprintf("reintentando (%d/%d) tras: %s", attempt, m.maxReconnectTries, lastErr.Error())
		delete(m.sessions, id) // la sesion anterior ya termino
		snapshot := *tunnel
		m.mu.Unlock()
		m.publisher.Publish("tunnel:status", &snapshot)

		time.Sleep(m.reconnectBackoff)

		m.mu.Lock()
		if _, stillThere := m.tunnels[id]; !stillThere {
			m.mu.Unlock()
			return nil, true
		}
		m.mu.Unlock()

		strategy, err := m.strategies.Strategy(req.Type)
		if err == nil {
			session, err = strategy.Start(req)
		}
		if err != nil {
			lastErr = err
			continue
		}

		m.mu.Lock()
		tunnel, ok = m.tunnels[id]
		if !ok {
			m.mu.Unlock()
			_ = session.Kill()
			return nil, true
		}
		tunnel.Status = "starting"
		tunnel.Message = ""
		m.sessions[id] = session
		// Variable propia (no reutiliza la de "reconnecting" de mas
		// arriba): esa ya se publico con su propio puntero, y pisarla
		// aqui competiria por la misma memoria con quien la siga leyendo.
		startedSnapshot := *tunnel
		m.mu.Unlock()
		m.publisher.Publish("tunnel:status", &startedSnapshot)

		return session, false
	}

	m.mu.Lock()
	tunnel, ok = m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return nil, true
	}
	tunnel.Status = "error"
	tunnel.Message = lastErr.Error()
	delete(m.tunnels, id)
	delete(m.sessions, id)
	snapshot := *tunnel
	m.mu.Unlock()

	m.publisher.Publish("tunnel:status", &snapshot)
	return nil, true
}

// autoReconnectEnabled lee el ajuste correspondiente al tipo de tunel
// en el momento del intento (no al arrancar el tunel original), para
// que si el usuario lo apaga a medio reintento surta efecto de
// inmediato. Ante un error leyendo el ajuste, se prefiere no
// reconectar en silencio.
func (m *tunnelManager) autoReconnectEnabled(t models.FavoriteType) bool {
	settings, err := m.settings.Get()
	if err != nil {
		return false
	}
	if t == models.FavoriteTypeSSH {
		return settings.AutoReconnectSSH
	}
	return settings.AutoReconnectSSM
}

// Stop marca el tunel como detenido y lo retira del registro antes de
// matar el proceso: asi, si el proceso muere justo en ese instante,
// la goroutine de await (ver arriba) lo encuentra ya ausente del mapa
// y no compite por reportar un estado distinto ("error" en vez de
// "stopped") para el mismo tunel.
//
// La sesion puede no existir aunque el tunel si (sessionOK == false):
// pasa mientras esta "reconnecting", en la ventana entre que la sesion
// anterior murio y la siguiente todavia no arranca (ver
// reconnectLoop). Ahi no hay nada que matar, pero igual hay que
// quitar el tunel del registro para que reconnectLoop lo note en su
// siguiente vuelta y no llegue a reconectar algo que el usuario ya
// detuvo.
func (m *tunnelManager) Stop(id string) error {
	m.mu.Lock()
	tunnel, tunnelOK := m.tunnels[id]
	if !tunnelOK {
		m.mu.Unlock()
		return fmt.Errorf("tunel no encontrado")
	}
	session, sessionOK := m.sessions[id]

	tunnel.Status = "stopped"
	delete(m.tunnels, id)
	delete(m.sessions, id)
	snapshot := *tunnel
	m.mu.Unlock()

	var killErr error
	if sessionOK {
		killErr = session.Kill()
	}
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
