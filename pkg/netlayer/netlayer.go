package netlayer

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/xlfzxr/tor/pkg/protocol"
)

// DialTCP opens a plain TCP connection to a relay.
func DialTCP(addr string, timeout time.Duration) (net.Conn, error) {
	if timeout <= 0 {
		timeout = protocol.DialTimeout
	}
	return net.DialTimeout("tcp", addr, timeout)
}

// DialTLS opens a TLS-wrapped relay connection.
func DialTLS(addr, serverName string, timeout time.Duration) (*tls.Conn, error) {
	return DialTLSWithConfig(addr, protocol.ClientConfig(serverName), timeout)
}

// DialTLSWithConfig is DialTLS with an explicit tls.Config.
// It exists so tests can inject trust for a local self-signed relay.
func DialTLSWithConfig(addr string, cfg *tls.Config, timeout time.Duration) (*tls.Conn, error) {
	if timeout <= 0 {
		timeout = protocol.DialTimeout
	}
	raw, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	c := tls.Client(raw, cfg)
	_ = c.SetDeadline(time.Now().Add(timeout))
	if err := c.Handshake(); err != nil {
		_ = raw.Close()
		return nil, err
	}
	_ = c.SetDeadline(time.Time{})
	return c, nil
}
