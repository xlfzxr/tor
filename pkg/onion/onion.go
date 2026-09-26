package onion

import (
	"fmt"
	"strings"
)

// Service is a placeholder for onion-service (hidden service) support.
// Hosting .onion locally comes after native circuit + cell layers.
type Config struct {
	Nickname string
	Ports    map[int]string // virtPort -> target addr
	DataDir  string
}

// Validate checks minimal fields.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Nickname) == "" {
		return fmt.Errorf("nickname required")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("datadir required")
	}
	if len(c.Ports) == 0 {
		return fmt.Errorf("at least one port mapping required")
	}
	for virt, target := range c.Ports {
		if virt < 1 || virt > 65535 {
			return fmt.Errorf("virtport %d out of range 1-65535", virt)
		}
		if strings.TrimSpace(target) == "" {
			return fmt.Errorf("virtport %d: empty target", virt)
		}
	}
	return nil
}
