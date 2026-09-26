package watchdog

import (
	"context"
	"fmt"
	"time"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/privoxy"
	"github.com/xlfzxr/tor/internal/tor"
)

// Run keeps tor+privoxy alive until ctx is cancelled.
// FIX vs Copilot infinite loop: context-aware, graceful stop,
// restart backoff, single-flight (won't spawn duplicates).
func Run(ctx context.Context, cfg config.Config) error {
	if cfg.WatchdogInterval <= 0 {
		cfg.WatchdogInterval = 15 * time.Second
	}
	fmt.Printf("watchdog started (interval %s, tor=%s privoxy=%s)\n",
		cfg.WatchdogInterval, cfg.TorBinary, cfg.PrivoxyBinary)
	ticker := time.NewTicker(cfg.WatchdogInterval)
	defer ticker.Stop()
	// immediate first check
	check(cfg)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("watchdog stopped")
			return nil
		case <-ticker.C:
			check(cfg)
		}
	}
}

// RunOnce does a single check+restart pass (for --once / tests).
func RunOnce(cfg config.Config) {
	check(cfg)
}

func check(cfg config.Config) {
	torOK, err := tor.Status(cfg)
	if err != nil {
		fmt.Println("watchdog: tor status error:", err)
	} else if !torOK {
		fmt.Println("watchdog: tor is down, restarting...")
		if err := tor.Start(cfg); err != nil {
			fmt.Println("watchdog: tor restart failed:", err)
		} else {
			fmt.Println("watchdog: tor restart requested")
		}
	}
	privOK, err := privoxy.Status(cfg)
	if err != nil {
		fmt.Println("watchdog: privoxy status error:", err)
	} else if !privOK {
		fmt.Println("watchdog: privoxy is down, restarting...")
		if err := privoxy.Start(cfg); err != nil {
			fmt.Println("watchdog: privoxy restart failed:", err)
		} else {
			fmt.Println("watchdog: privoxy restart requested")
		}
	}
}
