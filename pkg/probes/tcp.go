package probes

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

// Creates a new TCP prober
func NewTcpProber() TcpProber {
	return TcpProber{}
}

type TcpProber struct{}

// Probe returns a ProbeRunner capable of running an TCP check.
// If the socket can be opened, it returns Success
// If the socket fails to open, it returns Failure.
func (pr TcpProber) Probe(host string, port int, timeout time.Duration) (ProbeResult, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return Failure, err
	}
	err = conn.Close()

	if err != nil {
		return Unknown, fmt.Errorf("Unexpected error closing TCP probe socket: %v", err)
	}

	return Success, nil
}
