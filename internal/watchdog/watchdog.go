package watchdog

import (
	"context"
	"fmt"
	"time"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/privoxy"
	"github.com/xlfzxr/tor/internal/tor"
)

// MinAutorotate is the fastest allowed NEWNYM period.
// Tor rate-limits faster SIGNAL NEWNYM, so anything below is clamped.
const MinAutorotate = 10 * time.Second

// Run keeps tor+privoxy alive until ctx is cancelled.
// FIX vs Copilot infinite loop: context-aware, graceful stop,
// restart backoff, single-flight (won't spawn duplicates).
// When cfg.AutorotateInterval > 0 it also sends NEWNYM on that period
// (same semantics as `rotate`: ControlPort, SIGHUP fallback reported).
func Run(ctx context.Context, cfg config.Config) error {
	if cfg.WatchdogInterval <= 0 {
		cfg.WatchdogInterval = 15 * time.Second
	}
	fmt.Printf("watchdog started (interval %s, tor=%s privoxy=%s)\n",
		cfg.WatchdogInterval, cfg.TorBinary, cfg.PrivoxyBinary)
	ticker := time.NewTicker(cfg.WatchdogInterval)
	defer ticker.Stop()
	// Optional auto-rotate ticker; nil channel blocks forever when off.
	rotateEvery := cfg.AutorotateInterval
	var rotateTick <-chan time.Time
	if rotateEvery > 0 {
		if rotateEvery < MinAutorotate {
			fmt.Printf("watchdog: autorotate %s too fast, clamped to %s (Tor rate-limit)\n",
				rotateEvery, MinAutorotate)
			rotateEvery = MinAutorotate
		}
		rotTicker := time.NewTicker(rotateEvery)
		defer rotTicker.Stop()
		rotateTick = rotTicker.C
		fmt.Printf("watchdog: autorotate every %s\n", rotateEvery)
	}
	lastRotate := time.Now()
	// immediate first check
	check(cfg)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("watchdog stopped")
			return nil
		case <-ticker.C:
			check(cfg)
		case <-rotateTick:
			if shouldRotate(lastRotate, rotateEvery, time.Now()) {
				if err := tor.Rotate(cfg); err != nil {
					fmt.Println("watchdog: autorotate failed:", err)
				} else {
					lastRotate = time.Now()
					fmt.Println("watchdog: identity rotated (auto)")
				}
			}
		}
	}
}

// RunOnce does a single check+restart pass (for --once / tests).
// It never rotates: auto-rotate only runs inside Run's loop.
func RunOnce(cfg config.Config) {
	check(cfg)
}

// shouldRotate reports whether a NEWNYM is due.
// interval <= 0 means auto-rotate is off; zero last means "never rotated".
func shouldRotate(last time.Time, interval time.Duration, now time.Time) bool {
	if interval <= 0 {
		return false
	}
	if last.IsZero() {
		return true
	}
	return !now.Before(last.Add(interval))
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
