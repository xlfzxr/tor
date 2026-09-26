package control

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// NewNym sends SIGNAL NEWNYM (rotate identity) via ControlPort.
func NewNym(addr, password string) error {
	if addr == "" {
		return fmt.Errorf("empty control address")
	}
	c, err := net.DialTimeout("tcp", addr, 4*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	r := bufio.NewReader(c)
	if password != "" {
		fmt.Fprintf(c, "AUTHENTICATE \"%s\"\r\n", password)
	} else {
		fmt.Fprintf(c, "AUTHENTICATE\r\n")
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "250") {
		return fmt.Errorf("control auth rejected: %s", strings.TrimSpace(line))
	}
	fmt.Fprintf(c, "SIGNAL NEWNYM\r\n")
	line, err = r.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(line, "250") {
		return fmt.Errorf("control NEWNYM rejected: %s", strings.TrimSpace(line))
	}
	fmt.Fprintf(c, "QUIT\r\n")
	return nil
}

// Ping checks ControlPort reachability (optional — Tor may disable it).
func Ping(addr string, timeout time.Duration) error {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	return c.Close()
}
