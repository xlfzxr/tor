package client

import (
	"fmt"
	"syscall"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/privoxy"
	"github.com/xlfzxr/tor/internal/system"
	"github.com/xlfzxr/tor/internal/tor"
	"github.com/xlfzxr/tor/pkg/circuit"
	"github.com/xlfzxr/tor/pkg/control"
)

// Client is the orchestrator (diagram: pkg/client).
// Phase 1 wraps the external tor+privoxy daemons; native circuit
// manager runs alongside as state-only until cell/protocol land.
type Client struct {
	Config  config.Config
	Circuit *circuit.Manager
}

// New builds a client from resolved config.
func New(cfg config.Config) *Client {
	return &Client{Config: cfg, Circuit: circuit.New()}
}

// Start launches Tor + Privoxy detached.
func (c *Client) Start() error {
	if err := tor.Start(c.Config); err != nil {
		return err
	}
	return privoxy.Start(c.Config)
}

// Stop terminates both daemons.
func (c *Client) Stop() error { return stopStack(c.Config) }

// Stats reports daemon liveness.
func (c *Client) Stats() (torOK, privOK bool, err error) {
	torOK, err = tor.Status(c.Config)
	if err != nil {
		return false, false, err
	}
	privOK, err = privoxy.Status(c.Config)
	return torOK, privOK, err
}

// Rotate requests a new identity via ControlPort (pkg/control).
// Falls back to SIGHUP reload when ControlPort is unreachable —
// SIGHUP only reloads torrc, NOT identity, so the error tells which path ran.
func (c *Client) Rotate() error {
	if err := control.NewNym(c.Config.TorControlAddr, c.Config.TorControlPass); err == nil {
		return nil
	} else {
		ctrlErr := err
		if sigErr := system.SignalProcess("tor", syscall.SIGHUP); sigErr != nil {
			return fmt.Errorf("control NEWNYM failed (%v) and SIGHUP fallback failed: %w", ctrlErr, sigErr)
		}
		return fmt.Errorf("control NEWNYM failed (%v); sent SIGHUP reload instead (identity likely unchanged)", ctrlErr)
	}
}

func stopStack(cfg config.Config) error {
	// privoxy first so no new CONNECT lands during tor shutdown.
	_ = privoxy.Stop(cfg)
	return tor.Stop(cfg)
}
