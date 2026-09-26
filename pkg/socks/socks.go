package socks

import (
	"fmt"
	"net"
	"time"
)

// Handshake performs a SOCKS5 no-auth greeting against addr.
// Used by doctor + client readiness (Tor SOCKS 9050).
func Handshake(addr string, timeout time.Duration) error {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(timeout))
	if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return err
	}
	resp := make([]byte, 2)
	if err := readFull(c, resp); err != nil {
		return fmt.Errorf("socks5 handshake read: %w", err)
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		return fmt.Errorf("socks5 unexpected reply %x %x", resp[0], resp[1])
	}
	return nil
}

// DialVia connects to target through a SOCKS5 proxy (no-auth, CONNECT).
// Returns a tunnel ready for TLS (e.g. HTTPS :443).
func DialVia(proxyAddr, target string, timeout time.Duration) (net.Conn, error) {
	c, err := net.DialTimeout("tcp", proxyAddr, timeout)
	if err != nil {
		return nil, err
	}
	_ = c.SetDeadline(time.Now().Add(timeout))
	if _, err := c.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		_ = c.Close()
		return nil, err
	}
	hdr := make([]byte, 2)
	if err := readFull(c, hdr); err != nil {
		_ = c.Close()
		return nil, err
	}
	host, port, err := splitHostPort(target)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	req := buildConnect(host, port)
	if _, err := c.Write(req); err != nil {
		_ = c.Close()
		return nil, err
	}
	rep := make([]byte, 10)
	if err := readFull(c, rep); err != nil {
		_ = c.Close()
		return nil, err
	}
	if rep[1] != 0x00 {
		_ = c.Close()
		return nil, fmt.Errorf("socks5 connect failed: rep=%x", rep[1])
	}
	_ = c.SetDeadline(time.Time{})
	return c, nil
}

func readFull(c net.Conn, buf []byte) error {
	n := 0
	for n < len(buf) {
		m, err := c.Read(buf[n:])
		if err != nil {
			return err
		}
		n += m
	}
	return nil
}

func splitHostPort(target string) (string, int, error) {
	h, p, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, err
	}
	var port int
	_, err = fmt.Sscanf(p, "%d", &port)
	return h, port, err
}

func buildConnect(host string, port int) []byte {
	// ATYP=DOMAIN (0x03) so DNS resolves at the exit (no leak).
	out := []byte{0x05, 0x01, 0x00, 0x03, byte(len(host))}
	out = append(out, []byte(host)...)
	out = append(out, byte(port>>8), byte(port))
	return out
}
