package tor

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/system"
	"github.com/xlfzxr/tor/pkg/control"
)

// Start launches tor detached. Uses "-f torrc" only when the file exists.
func Start(cfg config.Config) error {
	if cfg.TorBinary == "" {
		return fmt.Errorf("tor binary not configured")
	}
	if _, err := exec.LookPath(cfg.TorBinary); err != nil {
		if _, statErr := exec.Command("sh", "-c", "command -v "+cfg.TorBinary).Output(); statErr != nil {
			return fmt.Errorf("tor binary %q not found in PATH: %w", cfg.TorBinary, err)
		}
	}
	if ok, err := system.IsRunning("tor"); err != nil {
		return err
	} else if ok {
		return nil // already up
	}
	args := []string{}
	if cfg.TorConfigPath != "" {
		args = append(args, "-f", cfg.TorConfigPath)
	}
	return system.StartDaemon(cfg.LogDir, "tor", cfg.TorBinary, args...)
}

func Stop(cfg config.Config) error {
	return system.StopProcess("tor")
}

func Status(cfg config.Config) (bool, error) {
	return system.IsRunning("tor")
}

// Rotate requests a new Tor identity via ControlPort SIGNAL NEWNYM.
// Delegates to pkg/control (single source of truth); SIGHUP fallback kept
// because SIGHUP only reloads torrc, NOT identity — caller must know.
func Rotate(cfg config.Config) error {
	if err := control.NewNym(cfg.TorControlAddr, cfg.TorControlPass); err == nil {
		return nil
	} else {
		ctrlErr := err
		if sigErr := system.SignalProcess("tor", syscall.SIGHUP); sigErr != nil {
			return fmt.Errorf("control NEWNYM failed (%v) and SIGHUP fallback failed: %w", ctrlErr, sigErr)
		}
		return fmt.Errorf("control NEWNYM failed (%v); sent SIGHUP reload instead (identity likely unchanged)", ctrlErr)
	}
}
