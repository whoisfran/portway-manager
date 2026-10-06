// Package application contiene los casos de uso: coordinan las
// entidades y los puertos de domain para cumplir lo que App necesita,
// sin saber nada de Wails, AWS ni del sistema de archivos.
package application

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
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
// reconexion automatica (ver await/reconnect) en produccion: unos
// pocos intentos con una espera fija alcanzan para curar un timeout de
// inactividad de SSM o un corte de red pasajero, sin insistir
// indefinidamente contra algo que de verdad dejo de responder
// (instancia apagada, credenciales vencidas, etc). Los tests fijan sus
// propios valores (mas cortos) directamente en los campos del struct,
// en vez de una var de paquete compartida entre goroutines de tests
// distintos.
//
// defaultStableAfter es cuanto tiene que sostenerse conectada una
// sesion para que su caida cuente como un problema nuevo (contador de
// intentos desde cero) y no como otro intento fallido del anterior.
const (
	defaultMaxReconnectAttempts = 3
	defaultReconnectBackoff     = 5 * time.Second
	defaultStableAfter          = 30 * time.Second
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
	stableAfter       time.Duration
}

// NewTunnelService crea un TunnelService a partir del registro que
// resuelve la estrategia (SSM o SSH) de cada solicitud, el publisher
// que notificara sus cambios de estado, el checker que valida que el
// puerto local no este ya ocupado antes de intentar abrir el tunel, y
// el repositorio de ajustes del que lee si toca reconectar solo un
// tunel que se cayo inesperadamente (ver reconnect).
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
		stableAfter:       defaultStableAfter,
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
		m.logf(tunnel.ID, req.FavoriteID, "no se pudo iniciar el tunel: %v", err)

		tunnel.Status = "error"
		tunnel.Message = err.Error()
		return tunnel, fmt.Errorf("no se pudo iniciar el tunel: %w", err)
	}

	m.mu.Lock()
	m.sessions[tunnel.ID] = session
	snapshot := *tunnel
	m.mu.Unlock()

	go m.await(tunnel.ID, req, session)

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

// logf publica una linea de log propia de la app (no del proceso) en
// el mismo canal que la salida de la sesion, para que el usuario vea
// en un solo lugar los reintentos, las desconexiones y su motivo. Va
// con favoriteId para que el frontend conserve el historial por perfil
// aun despues de que el tunel se retira del registro.
func (m *tunnelManager) logf(id, favoriteID, format string, args ...any) {
	m.publisher.Publish("tunnel:log", map[string]string{
		"id":         id,
		"favoriteId": favoriteID,
		"line":       "[portway] " + fmt.Sprintf(format, args...),
	})
}

// pump lee la salida de la sesion linea por linea, la reenvia como
// evento de log y devuelve la ultima linea no vacia (util como motivo
// de la caida). Solo stdout marca el tunel como "running": la AWS CLI
// escribe sus errores (credenciales, TargetNotConnected, etc) en
// stderr, y eso no significa que la sesion haya arrancado.
func (m *tunnelManager) pump(id, favoriteID string, pipe io.Reader, marksRunning bool, onRunning func()) string {
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var last string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			last = strings.TrimSpace(line)
		}
		if marksRunning && m.markRunning(id) && onRunning != nil {
			onRunning()
		}
		m.publisher.Publish("tunnel:log", map[string]string{
			"id":         id,
			"favoriteId": favoriteID,
			"line":       line,
		})
	}

	if err := scanner.Err(); err != nil {
		m.logf(id, favoriteID, "error leyendo la salida de la sesion: %v", err)
	}
	return last
}

// markRunning realiza la transicion starting -> running una sola vez,
// protegida por el mutex para evitar condiciones de carrera con Stop
// y con la goroutine que espera la salida del proceso. Devuelve true
// solo para quien efectivamente hizo la transicion.
func (m *tunnelManager) markRunning(id string) bool {
	m.mu.Lock()
	t, ok := m.tunnels[id]
	if !ok || t.Status != "starting" {
		m.mu.Unlock()
		return false
	}
	t.Status = "running"
	t.Message = ""
	snapshot := *t
	m.mu.Unlock()

	m.publisher.Publish("tunnel:status", &snapshot)
	return true
}

// runSession consume stdout/stderr hasta EOF y solo entonces llama a
// Wait: exec.Cmd.Wait cierra los pipes en cuanto el proceso termina,
// asi que llamarlo antes perderia las ultimas lineas (justo las que
// explican por que se cayo) y haria fallar al scanner con "file
// already closed". Devuelve el resultado de Wait y la ultima linea
// que imprimio la sesion, preferentemente de stderr.
func (m *tunnelManager) runSession(id, favoriteID string, session domain.RunningSession, onRunning func()) (error, string) {
	var wg sync.WaitGroup
	var lastOut, lastErr string
	wg.Add(2)
	go func() {
		defer wg.Done()
		lastOut = m.pump(id, favoriteID, session.Stdout(), true, onRunning)
	}()
	go func() {
		defer wg.Done()
		lastErr = m.pump(id, favoriteID, session.Stderr(), false, nil)
	}()
	wg.Wait()

	err := session.Wait()
	if lastErr != "" {
		return err, lastErr
	}
	return err, lastOut
}

// await acompaña a la sesion de un tunel durante toda su vida,
// incluidas las reconexiones. Si al terminar la sesion el tunel ya no
// esta registrado es porque Stop lo finalizo explicitamente, y no hay
// nada mas que hacer.
//
// Cualquier otra salida -- con error o limpia -- es una caida que el
// usuario no pidio: el session-manager-plugin sale con codigo 0 cuando
// AWS cierra la sesion (timeout por inactividad, reinicio de la
// instancia, sesion terminada desde la consola), asi que tratar la
// salida limpia como un Stop dejaba esos tuneles caidos sin
// reconectar.
func (m *tunnelManager) await(id string, req models.TunnelRequest, session domain.RunningSession) {
	attempt := 0
	for {
		startedAt := time.Now()
		var onRunning func()
		if attempt > 0 {
			n := attempt
			onRunning = func() {
				m.logf(id, req.FavoriteID, "conexion restablecida (reintento %d/%d)", n, m.maxReconnectTries)
			}
		}

		exitErr, lastLine := m.runSession(id, req.FavoriteID, session, onRunning)

		m.mu.Lock()
		tunnel, ok := m.tunnels[id]
		reachedRunning := ok && tunnel.Status == "running"
		if ok {
			delete(m.sessions, id) // la sesion ya termino
		}
		m.mu.Unlock()
		if !ok {
			return
		}

		// Una sesion que llego a conectar y se mantuvo un rato reinicia
		// el contador: la siguiente caida es un problema nuevo. Si en
		// cambio murio enseguida (p.ej. el proceso de SSM arranca bien
		// pero AWS rechaza la sesion), sigue contando el mismo intento;
		// si no, un fallo persistente se reintentaria para siempre,
		// porque arrancar el proceso casi nunca falla por si mismo.
		if reachedRunning && time.Since(startedAt) >= m.stableAfter {
			attempt = 0
		}

		cause := describeExit(exitErr, lastLine)
		m.logf(id, req.FavoriteID, "conexion perdida: %s", cause)

		session, attempt = m.reconnect(id, req, attempt, cause)
		if session == nil {
			return
		}
	}
}

// describeExit arma un motivo legible para una sesion que termino
// sola, combinando el resultado del proceso con lo ultimo que imprimio
// (que suele ser el mensaje real de la AWS CLI o del plugin).
func describeExit(err error, lastLine string) string {
	switch {
	case err != nil && lastLine != "":
		return fmt.Sprintf("%s (%v)", lastLine, err)
	case err != nil:
		return err.Error()
	case lastLine != "":
		return fmt.Sprintf("la sesion termino de forma inesperada: %s", lastLine)
	default:
		return "la sesion termino de forma inesperada"
	}
}

// reconnect reintenta abrir el mismo tunel tras una caida: publica
// "reconnecting" (para que la UI lo distinga de un error definitivo),
// espera reconnectBackoff, y prueba de nuevo hasta maxReconnectTries,
// dejando constancia de cada paso en el log. Se rinde -- deja el
// tunel en "error" con el motivo -- si el tipo de tunel tiene la
// reconexion apagada o si se agotan los intentos; y sale en silencio
// si el usuario lo detiene mientras tanto (Stop lo quita de m.tunnels;
// este metodo lo nota en la siguiente vuelta y no sigue insistiendo).
//
// Devuelve la sesion nueva si tuvo exito (nil si no) y el numero de
// intento en el que va, para que await siga contando si esa sesion
// tambien se cae enseguida.
func (m *tunnelManager) reconnect(id string, req models.TunnelRequest, attempt int, cause string) (domain.RunningSession, int) {
	for {
		if !m.autoReconnectEnabled(req.Type) {
			m.fail(id, req.FavoriteID, fmt.Sprintf("reconexion automatica desactivada; motivo: %s", cause))
			return nil, attempt
		}
		if attempt >= m.maxReconnectTries {
			m.fail(id, req.FavoriteID, fmt.Sprintf(
				"se agotaron los %d intentos de reconexion; ultimo error: %s", m.maxReconnectTries, cause))
			return nil, attempt
		}
		attempt++

		m.mu.Lock()
		tunnel, ok := m.tunnels[id]
		if !ok {
			m.mu.Unlock()
			return nil, attempt
		}
		tunnel.Status = "reconnecting"
		tunnel.Message = fmt.Sprintf("reintentando (%d/%d) tras: %s", attempt, m.maxReconnectTries, cause)
		snapshot := *tunnel
		m.mu.Unlock()
		m.publisher.Publish("tunnel:status", &snapshot)
		m.logf(id, req.FavoriteID, "reintento %d/%d en %s...", attempt, m.maxReconnectTries, m.reconnectBackoff)

		time.Sleep(m.reconnectBackoff)

		m.mu.Lock()
		_, stillThere := m.tunnels[id]
		m.mu.Unlock()
		if !stillThere {
			return nil, attempt
		}

		strategy, err := m.strategies.Strategy(req.Type)
		var session domain.RunningSession
		if err == nil {
			session, err = strategy.Start(req)
		}
		if err != nil {
			cause = err.Error()
			m.logf(id, req.FavoriteID, "reintento %d/%d fallo: %s", attempt, m.maxReconnectTries, cause)
			continue
		}

		m.mu.Lock()
		tunnel, ok = m.tunnels[id]
		if !ok {
			m.mu.Unlock()
			_ = session.Kill()
			return nil, attempt
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
		m.logf(id, req.FavoriteID, "reintento %d/%d: sesion iniciada, esperando conexion...", attempt, m.maxReconnectTries)

		return session, attempt
	}
}

// fail deja el tunel en "error" con el motivo, lo retira del registro
// y lo anota en el log. Es el unico camino que termina en un aviso al
// usuario (toast o notificacion de sistema): una desconexion manual
// no lo necesita.
func (m *tunnelManager) fail(id, favoriteID, reason string) {
	m.mu.Lock()
	tunnel, ok := m.tunnels[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	tunnel.Status = "error"
	tunnel.Message = reason
	delete(m.tunnels, id)
	delete(m.sessions, id)
	snapshot := *tunnel
	m.mu.Unlock()

	m.logf(id, favoriteID, "tunel desconectado: %s", reason)
	m.publisher.Publish("tunnel:status", &snapshot)
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
// reconnect). Ahi no hay nada que matar, pero igual hay que quitar
// el tunel del registro para que reconnect lo note en su siguiente
// vuelta y no llegue a reconectar algo que el usuario ya detuvo.
//
// Se sigue publicando "stopped" (la UI y la bandeja necesitan saber
// que el tunel ya no esta), pero ni el frontend ni la bandeja avisan
// por ello: solo "error" amerita un aviso.
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
	m.logf(id, snapshot.Request.FavoriteID, "tunel desconectado")
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
