// Package ssh agrupa todo lo necesario para tuneles de port-forwarding
// local (-L) sobre una conexion SSH nativa (golang.org/x/crypto/ssh):
// el TunnelStrategy (ver strategy.go) y la sesion en ejecucion que
// arranca. A diferencia de internal/infrastructure/ssm, no depende de
// ningun binario externo.
package ssh

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// session adapta un listener local + cliente SSH al puerto
// domain.RunningSession: por cada conexion aceptada en el listener
// abre un canal remoto a traves del cliente y copia bytes en ambas
// direcciones. Como no hay un proceso de sistema detras (a diferencia
// de la sesion SSM), los "stdout"/"stderr" que expone son lineas de
// log sinteticas escritas a un io.Pipe.
type session struct {
	listener net.Listener
	client   *ssh.Client

	stdoutW *io.PipeWriter
	stdoutR *io.PipeReader
	stderrW *io.PipeWriter
	stderrR *io.PipeReader

	done      chan error
	closeOnce sync.Once
}

func newSession(listener net.Listener, client *ssh.Client) *session {
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	return &session{
		listener: listener,
		client:   client,
		stdoutR:  stdoutR,
		stdoutW:  stdoutW,
		stderrR:  stderrR,
		stderrW:  stderrW,
		done:     make(chan error, 1),
	}
}

func (s *session) Stdout() io.Reader { return s.stdoutR }
func (s *session) Stderr() io.Reader { return s.stderrR }
func (s *session) Wait() error       { return <-s.done }

// Kill cierra el listener local y el cliente SSH, lo que desbloquea
// acceptLoop y termina cualquier reenvio en curso.
func (s *session) Kill() error {
	s.closeOnce.Do(func() {
		_ = s.listener.Close()
		_ = s.client.Close()
	})
	return nil
}

func (s *session) logf(format string, args ...any) {
	fmt.Fprintf(s.stdoutW, format+"\n", args...)
}

func (s *session) errorf(format string, args ...any) {
	fmt.Fprintf(s.stderrW, format+"\n", args...)
}

// acceptLoop acepta conexiones en el listener local hasta que este se
// cierra (por Kill, o por una falla real de red) y reenvia cada una
// hacia remoteAddr a traves del cliente SSH.
func (s *session) acceptLoop(remoteAddr string) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.finish(err)
			return
		}
		go s.forward(conn, remoteAddr)
	}
}

func (s *session) finish(acceptErr error) {
	_ = s.client.Close()

	var result error
	if !errors.Is(acceptErr, net.ErrClosed) {
		s.errorf("tunel SSH detenido de forma inesperada: %v", acceptErr)
		result = acceptErr
	}

	_ = s.stdoutW.Close()
	_ = s.stderrW.Close()
	s.done <- result
}

// forward copia bytes en ambas direcciones entre la conexion local
// aceptada y un canal nuevo abierto a traves del cliente SSH hacia
// remoteAddr. Cada conexion local aceptada abre su propio canal, tal
// como hace "ssh -L".
func (s *session) forward(local net.Conn, remoteAddr string) {
	defer local.Close()

	remote, err := s.client.Dial("tcp", remoteAddr)
	if err != nil {
		s.errorf("no se pudo abrir el canal remoto hacia %s: %v", remoteAddr, err)
		return
	}
	defer remote.Close()

	s.logf("conexion aceptada, reenviando hacia %s", remoteAddr)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(remote, local)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(local, remote)
	}()
	wg.Wait()
}
