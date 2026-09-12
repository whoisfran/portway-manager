package application

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"portway-manager/internal/domain"
	"portway-manager/models"
)

// fakeSession es un domain.RunningSession controlable desde el test:
// Wait() bloquea hasta que el test llama a finish(err), simulando que
// el proceso real termino (crash, cierre remoto, Kill, etc).
type fakeSession struct {
	done   chan struct{}
	err    error
	killed chan struct{}
}

func newFakeSession() *fakeSession {
	return &fakeSession{done: make(chan struct{}), killed: make(chan struct{})}
}

func (s *fakeSession) Stdout() io.Reader { return strings.NewReader("") }
func (s *fakeSession) Stderr() io.Reader { return strings.NewReader("") }

func (s *fakeSession) Wait() error {
	<-s.done
	return s.err
}

func (s *fakeSession) Kill() error {
	select {
	case <-s.killed:
	default:
		close(s.killed)
	}
	s.finish(nil)
	return nil
}

// finish simula que el proceso termino por su cuenta con este error
// (nil = salida limpia). Solo debe llamarse una vez.
func (s *fakeSession) finish(err error) {
	s.err = err
	close(s.done)
}

// fakeStrategy arranca fakeSession's segun startFn, que el test define
// para simular exito/fracaso en cada intento de (re)conexion.
type fakeStrategy struct {
	favoriteType models.FavoriteType
	mu           sync.Mutex
	startCalls   int
	startFn      func(call int) (domain.RunningSession, error)
}

func (s *fakeStrategy) Type() models.FavoriteType                             { return s.favoriteType }
func (s *fakeStrategy) ValidateFavorite(models.Favorite) error                { return nil }
func (s *fakeStrategy) BuildRequest(models.Favorite) models.TunnelRequest     { return models.TunnelRequest{} }
func (s *fakeStrategy) ValidateRequest(models.TunnelRequest) error            { return nil }
func (s *fakeStrategy) Start(req models.TunnelRequest) (domain.RunningSession, error) {
	s.mu.Lock()
	call := s.startCalls
	s.startCalls++
	s.mu.Unlock()
	return s.startFn(call)
}

// fakeRegistry ya esta definido en profile_service_test.go (mismo
// paquete): reutilizado aqui tal cual.

type fakePortChecker struct{}

func (fakePortChecker) IsAvailable(int) bool { return true }

type fakeSettingsRepo struct {
	mu       sync.Mutex
	settings models.AppSettings
}

func (r *fakeSettingsRepo) Get() (models.AppSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.settings, nil
}

func (r *fakeSettingsRepo) Save(s models.AppSettings) (models.AppSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings = s
	return s, nil
}

// fakePublisher registra los eventos "tunnel:status" en un canal, para
// que el test pueda esperar (con timeout) cada transicion en orden en
// vez de adivinar cuanto dormir.
type fakePublisher struct {
	statuses chan *models.Tunnel
}

func newFakePublisher() *fakePublisher {
	return &fakePublisher{statuses: make(chan *models.Tunnel, 32)}
}

func (p *fakePublisher) SetContext(context.Context) {}

func (p *fakePublisher) Publish(event string, payload any) {
	if event != "tunnel:status" {
		return
	}
	if t, ok := payload.(*models.Tunnel); ok {
		p.statuses <- t
	}
}

func (p *fakePublisher) next(t *testing.T) *models.Tunnel {
	t.Helper()
	select {
	case tun := <-p.statuses:
		return tun
	case <-time.After(2 * time.Second):
		t.Fatal("timeout esperando un evento tunnel:status")
		return nil
	}
}

// testReconnectBackoff es deliberadamente cortisimo: cada tunnelManager
// de prueba tiene su propio campo (no una var de paquete compartida),
// asi que no hace falta ningun defer/reset ni corre riesgo de carrera
// con goroutines de otros tests bajo -race.
const testReconnectBackoff = 5 * time.Millisecond

func newTestManager(strategy *fakeStrategy, settings models.AppSettings) (*tunnelManager, *fakePublisher) {
	publisher := newFakePublisher()
	m := &tunnelManager{
		tunnels:           make(map[string]*models.Tunnel),
		sessions:          make(map[string]domain.RunningSession),
		strategies:        fakeRegistry{strategy: strategy},
		publisher:         publisher,
		portChecker:       fakePortChecker{},
		settings:          &fakeSettingsRepo{settings: settings},
		maxReconnectTries: defaultMaxReconnectAttempts,
		reconnectBackoff:  testReconnectBackoff,
	}
	return m, publisher
}

func testRequest(favType models.FavoriteType) models.TunnelRequest {
	return models.TunnelRequest{
		FavoriteID: "fav-1",
		Type:       favType,
		LocalPort:  1234,
		RemotePort: 5678,
		InstanceID: "i-test",
	}
}

// TestReconnect_SucceedsAfterOneFailedAttempt cubre el camino feliz:
// la sesion original muere con error, el primer reintento de arranque
// falla, el segundo tiene exito -- y el tunel termina "running" de
// nuevo bajo el mismo ID.
func TestReconnect_SucceedsAfterOneFailedAttempt(t *testing.T) {
	var sessions []*fakeSession
	strategy := &fakeStrategy{favoriteType: models.FavoriteTypeSSM}
	strategy.startFn = func(call int) (domain.RunningSession, error) {
		switch call {
		case 0:
			s := newFakeSession()
			sessions = append(sessions, s)
			return s, nil
		case 1:
			return nil, fmt.Errorf("instancia no responde todavia")
		default:
			s := newFakeSession()
			sessions = append(sessions, s)
			return s, nil
		}
	}

	m, publisher := newTestManager(strategy, models.AppSettings{AutoReconnectSSM: true})

	tunnel, err := m.Start(testRequest(models.FavoriteTypeSSM))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// La sesion original se cae sola. Se esperan dos anuncios
	// "reconnecting" (el intento 1, cuyo Start falla, y el intento 2,
	// que tiene exito) antes de volver a "starting".
	sessions[0].finish(fmt.Errorf("conexion perdida"))

	for i := 0; i < 2; i++ {
		reconnecting := publisher.next(t)
		if reconnecting.Status != "reconnecting" {
			t.Fatalf("evento %d: status = %q, quiero reconnecting", i, reconnecting.Status)
		}
	}

	starting := publisher.next(t)
	if starting.Status != "starting" {
		t.Fatalf("status = %q, quiero starting (tras el reintento exitoso)", starting.Status)
	}
	if starting.ID != tunnel.ID {
		t.Fatalf("el tunel reconectado deberia conservar el mismo ID")
	}

	m.mu.Lock()
	_, stillTracked := m.tunnels[tunnel.ID]
	m.mu.Unlock()
	if !stillTracked {
		t.Fatal("el tunel deberia seguir registrado tras reconectar")
	}
}

// TestReconnect_GivesUpAfterMaxAttempts cubre que, si todos los
// intentos fallan, el tunel termina "error" (como antes de que
// existiera la reconexion) y se retira del registro.
func TestReconnect_GivesUpAfterMaxAttempts(t *testing.T) {
	strategy := &fakeStrategy{favoriteType: models.FavoriteTypeSSM}
	strategy.startFn = func(call int) (domain.RunningSession, error) {
		if call == 0 {
			return newFakeSession(), nil
		}
		return nil, fmt.Errorf("instancia no responde")
	}

	m, publisher := newTestManager(strategy, models.AppSettings{AutoReconnectSSM: true})

	tunnel, err := m.Start(testRequest(models.FavoriteTypeSSM))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	m.mu.Lock()
	session := m.sessions[tunnel.ID].(*fakeSession)
	m.mu.Unlock()
	session.finish(fmt.Errorf("conexion perdida"))

	for i := 0; i < defaultMaxReconnectAttempts; i++ {
		got := publisher.next(t)
		if got.Status != "reconnecting" {
			t.Fatalf("intento %d: status = %q, quiero reconnecting", i, got.Status)
		}
	}

	final := publisher.next(t)
	if final.Status != "error" {
		t.Fatalf("status final = %q, quiero error", final.Status)
	}

	m.mu.Lock()
	_, stillTracked := m.tunnels[tunnel.ID]
	m.mu.Unlock()
	if stillTracked {
		t.Fatal("el tunel no deberia seguir registrado tras agotar los reintentos")
	}
}

// TestReconnect_DisabledGoesStraightToError confirma que, con el
// ajuste apagado para ese tipo, una caida inesperada se comporta
// exactamente como antes de que existiera la reconexion: "error" de
// inmediato, sin ningun evento "reconnecting" de por medio.
func TestReconnect_DisabledGoesStraightToError(t *testing.T) {
	strategy := &fakeStrategy{favoriteType: models.FavoriteTypeSSH}
	strategy.startFn = func(call int) (domain.RunningSession, error) {
		return newFakeSession(), nil
	}

	m, publisher := newTestManager(strategy, models.AppSettings{AutoReconnectSSH: false})

	tunnel, err := m.Start(testRequest(models.FavoriteTypeSSH))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	m.mu.Lock()
	session := m.sessions[tunnel.ID].(*fakeSession)
	m.mu.Unlock()
	session.finish(fmt.Errorf("conexion perdida"))

	final := publisher.next(t)
	if final.Status != "error" {
		t.Fatalf("status = %q, quiero error (reconexion apagada)", final.Status)
	}
	if strategy.startCalls != 1 {
		t.Fatalf("Start se llamo %d veces, con la reconexion apagada solo deberia ser 1 (el inicial)", strategy.startCalls)
	}
}

// TestReconnect_StopDuringBackoffPreventsRestart cubre la ventana de
// carrera entre que la sesion murio y el siguiente intento arranca: si
// el usuario detiene el tunel ahi, no debe reconectarse.
func TestReconnect_StopDuringBackoffPreventsRestart(t *testing.T) {
	// Este test necesita un backoff mas largo que el default de
	// pruebas: le hace falta una ventana real donde el tunel quede en
	// "reconnecting" para poder detenerlo ahi antes del reintento.
	const backoff = 100 * time.Millisecond

	strategy := &fakeStrategy{favoriteType: models.FavoriteTypeSSM}
	restartAttempted := make(chan struct{}, 1)
	strategy.startFn = func(call int) (domain.RunningSession, error) {
		if call > 0 {
			restartAttempted <- struct{}{}
		}
		return newFakeSession(), nil
	}

	m, publisher := newTestManager(strategy, models.AppSettings{AutoReconnectSSM: true})
	m.reconnectBackoff = backoff

	tunnel, err := m.Start(testRequest(models.FavoriteTypeSSM))
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	m.mu.Lock()
	session := m.sessions[tunnel.ID].(*fakeSession)
	m.mu.Unlock()
	session.finish(fmt.Errorf("conexion perdida"))

	reconnecting := publisher.next(t)
	if reconnecting.Status != "reconnecting" {
		t.Fatalf("status = %q, quiero reconnecting", reconnecting.Status)
	}

	// El usuario detiene el tunel mientras reconnectLoop todavia esta
	// esperando el backoff.
	if err := m.Stop(tunnel.ID); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	stopped := publisher.next(t)
	if stopped.Status != "stopped" {
		t.Fatalf("status = %q, quiero stopped", stopped.Status)
	}

	select {
	case <-restartAttempted:
		t.Fatal("no deberia reintentar arrancar el tunel tras un Stop explicito")
	case <-time.After(backoff * 2):
	}
}
