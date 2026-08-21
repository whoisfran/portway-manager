package infrastructure

import (
	"fmt"
	"net"

	"portway-manager/internal/domain"
)

// tcpPortAvailabilityChecker verifica un puerto intentando escuchar
// brevemente en el (bind + close inmediato). No es una garantia
// absoluta -- otro proceso podria tomar el puerto justo despues del
// chequeo -- pero detecta el caso comun de un puerto ya ocupado antes
// de invocar el CLI de AWS.
type tcpPortAvailabilityChecker struct{}

func NewTCPPortAvailabilityChecker() domain.PortAvailabilityChecker {
	return &tcpPortAvailabilityChecker{}
}

func (c *tcpPortAvailabilityChecker) IsAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
