package netlayer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

func localTCP(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func localTLS(t *testing.T) (string, *tls.Config, func()) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("tls listen: %v", err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				if tc, ok := conn.(*tls.Conn); ok {
					_ = tc.Handshake()
				}
			}(c)
		}
	}()
	pool := x509.NewCertPool()
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	pool.AddCert(parsed)
	clientCfg := &tls.Config{ServerName: "localhost", RootCAs: pool, MinVersion: tls.VersionTLS12}
	return ln.Addr().String(), clientCfg, func() { _ = ln.Close() }
}

func TestDialTCPOK(t *testing.T) {
	addr, stop := localTCP(t)
	defer stop()
	c, err := DialTCP(addr, 3*time.Second)
	if err != nil {
		t.Fatalf("DialTCP: %v", err)
	}
	_ = c.Close()
}

func TestDialTCPErr(t *testing.T) {
	if _, err := DialTCP("127.0.0.1:1", 500*time.Millisecond); err == nil {
		t.Fatalf("expected error dialing closed port")
	}
}

func TestDialTLSWithConfigOK(t *testing.T) {
	addr, cfg, stop := localTLS(t)
	defer stop()
	c, err := DialTLSWithConfig(addr, cfg, 3*time.Second)
	if err != nil {
		t.Fatalf("DialTLSWithConfig: %v", err)
	}
	_ = c.Close()
}

func TestDialTLSErr(t *testing.T) {
	// Unroutable/closed port.
	if _, err := DialTLS("127.0.0.1:1", "localhost", 500*time.Millisecond); err == nil {
		t.Fatalf("expected error dialing closed port")
	}
	// Plain TCP endpoint is not TLS: handshake must fail.
	addr, stop := localTCP(t)
	defer stop()
	if _, err := DialTLS(addr, "localhost", 2*time.Second); err == nil {
		t.Fatalf("expected TLS handshake error against plaintext")
	}
}
