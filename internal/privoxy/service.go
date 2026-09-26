package privoxy

import (
	"fmt"
	"os/exec"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/system"
)

// Start launches privoxy detached.
// FIX vs Copilot version: privoxy takes configfile as positional arg,
// there is NO "-f" flag (see `privoxy --help`).
func Start(cfg config.Config) error {
	if cfg.PrivoxyBinary == "" {
		return fmt.Errorf("privoxy binary not configured")
	}
	if _, err := exec.LookPath(cfg.PrivoxyBinary); err != nil {
		return fmt.Errorf("privoxy binary %q not found in PATH: %w", cfg.PrivoxyBinary, err)
	}
	if ok, err := system.IsRunning("privoxy"); err != nil {
		return err
	} else if ok {
		return nil
	}
	args := []string{}
	if cfg.PrivoxyConfigPath != "" {
		args = append(args, cfg.PrivoxyConfigPath)
	}
	return system.StartDaemon(cfg.LogDir, "privoxy", cfg.PrivoxyBinary, args...)
}

func Stop(cfg config.Config) error {
	return system.StopProcess("privoxy")
}

func Status(cfg config.Config) (bool, error) {
	return system.IsRunning("privoxy")
}
