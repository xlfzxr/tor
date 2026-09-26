package health

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/pkg/socks"
)

// Result is one verbose doctor check line.
type Result struct {
	Name    string
	OK      bool
	Latency time.Duration
	Detail  string
}

// CheckPort is a plain TCP dial (kept for compat).
func CheckPort(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

// CheckSOCKS5 delegates to pkg/socks (single source of truth).
// Kept here for compat; new code should call pkg/socks.Handshake directly.
func CheckSOCKS5(addr string, timeout time.Duration) error {
	return socks.Handshake(addr, timeout)
}

// CheckHTTPProxy sends a minimal HTTP request through the proxy address.
// Privoxy is an HTTP proxy on 8118; a direct GET / must return an HTTP status line.
func CheckHTTPProxy(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	fmt.Fprintf(conn, "GET http://example.com/ HTTP/1.0\r\nHost: example.com\r\n\r\n")
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "HTTP/") {
		return fmt.Errorf("not an HTTP proxy, got: %s", strings.TrimSpace(line))
	}
	return nil
}

// CheckHTTPSConnect verifies HTTPS :443 TCP via the HTTP proxy (CONNECT tunnel).
// This is what makes Termux terminal "auto inject" work for curl/git/npm:
// HTTPS_PROXY=http://privoxy -> CONNECT example.com:443 -> expect 200.
func CheckHTTPSConnect(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	fmt.Fprintf(conn, "CONNECT example.com:443 HTTP/1.1\r\nHost: example.com:443\r\n\r\n")
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	// Privoxy returns "HTTP/1.0 200 Connection established" on success.
	if !strings.HasPrefix(line, "HTTP/") || !strings.Contains(line, "200") {
		return fmt.Errorf("CONNECT :443 failed, got: %s", strings.TrimSpace(line))
	}
	return nil
}

// CheckAll runs process + protocol checks for doctor -v.
func CheckAll(cfg config.Config, torOK, privOK bool) []Result {
	timeout := 4 * time.Second
	out := []Result{}

	t0 := time.Now()
	tcpErr := CheckPort(cfg.TorSOCKSAddr, timeout)
	out = append(out, Result{"tor tcp " + cfg.TorSOCKSAddr, tcpErr == nil && torOK, time.Since(t0), errStr(tcpErr)})

	t0 = time.Now()
	socksErr := CheckSOCKS5(cfg.TorSOCKSAddr, timeout)
	out = append(out, Result{"tor socks5 handshake", socksErr == nil, time.Since(t0), errStr(socksErr)})

	t0 = time.Now()
	ptcpErr := CheckPort(cfg.PrivoxyAddr, timeout)
	out = append(out, Result{"privoxy tcp " + cfg.PrivoxyAddr, ptcpErr == nil && privOK, time.Since(t0), errStr(ptcpErr)})

	t0 = time.Now()
	httpErr := CheckHTTPProxy(cfg.PrivoxyAddr, timeout)
	out = append(out, Result{"privoxy http proxy", httpErr == nil, time.Since(t0), errStr(httpErr)})

	t0 = time.Now()
	httpsErr := CheckHTTPSConnect(cfg.PrivoxyAddr, timeout)
	out = append(out, Result{"https :443 via proxy", httpsErr == nil, time.Since(t0), errStr(httpsErr)})

	t0 = time.Now()
	ctrlErr := CheckPort(cfg.TorControlAddr, 2*time.Second)
	// ControlPort is optional: report but don't fail overall on it.
	out = append(out, Result{"tor control " + cfg.TorControlAddr + " (optional)", ctrlErr == nil, time.Since(t0), errStr(ctrlErr)})

	return out
}

func errStr(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	n := 0
	for n < len(buf) {
		m, err := conn.Read(buf[n:])
		if err != nil {
			return n, err
		}
		n += m
	}
	return n, nil
}
