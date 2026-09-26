package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xlfzxr/tor/internal/config"
	"github.com/xlfzxr/tor/internal/env"
	"github.com/xlfzxr/tor/internal/health"
	"github.com/xlfzxr/tor/internal/watchdog"
	"github.com/xlfzxr/tor/pkg/client"
)

var (
	flagConfig       = flag.String("config", "", "path to JSON config file")
	flagDebug        = flag.Bool("debug", false, "verbose debug output")
	flagWatchInt     = flag.Int("watch-interval", 0, "watchdog interval in seconds")
	flagTorBin       = flag.String("tor-binary", "", "override tor binary")
	flagPrivoxyBin   = flag.String("privoxy-binary", "", "override privoxy binary")
	flagTorConfig    = flag.String("tor-config", "", "override torrc path")
	flagPrivoxyConf  = flag.String("privoxy-config", "", "override privoxy config path")
	flagControlAddr  = flag.String("control-addr", "", "override tor ControlPort addr")
	flagControlPass  = flag.String("control-pass", "", "override tor ControlPort password")
	flagWatchOnce    = flag.Bool("once", false, "watchdog: single check pass and exit")
	flagAutorotate   = flag.Int("autorotate", 0, "watchdog: NEWNYM period in seconds (0 = off, min 10)")
	flagDoctorVerb   = flag.Bool("v", false, "doctor: verbose protocol checks")
)

func main() {
	flag.Usage = printUsage
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}
	if cfg.Debug {
		fmt.Fprintf(os.Stderr, "debug: tor=%s privoxy=%s torrc=%s privoxyConf=%s logs=%s\n",
			cfg.TorBinary, cfg.PrivoxyBinary, cfg.TorConfigPath, cfg.PrivoxyConfigPath, cfg.LogDir)
	}

	cmd := strings.TrimSpace(args[0])
	rest := args[1:]
	// allow `doctor -v` as positional too
	verbose := *flagDoctorVerb
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		if a == "-v" || a == "--verbose" {
			verbose = true
		}
		if a == "--once" {
			*flagWatchOnce = true
		}
		if a == "--autorotate" && i+1 < len(rest) {
			if n, err := strconv.Atoi(rest[i+1]); err == nil {
				*flagAutorotate = n
			}
			i++
		} else if strings.HasPrefix(a, "--autorotate=") {
			if n, err := strconv.Atoi(strings.TrimPrefix(a, "--autorotate=")); err == nil {
				*flagAutorotate = n
			}
		}
	}

	switch cmd {
	case "start":
		if err := startStack(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "start failed: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := stopStack(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "stop failed: %v\n", err)
			os.Exit(1)
		}
	case "restart":
		if err := stopStack(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "restart (stop) failed: %v\n", err)
			os.Exit(1)
		}
		time.Sleep(2 * time.Second)
		if err := startStack(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "restart (start) failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		s, err := stackStatus(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "status failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(s)
	case "rotate":
		if err := client.New(cfg).Rotate(); err != nil {
			// Rotate returns descriptive fallback error; NEWNYM success returns nil.
			fmt.Fprintf(os.Stderr, "rotate: %v\n", err)
			if strings.Contains(err.Error(), "SIGHUP reload instead") {
				os.Exit(0) // fallback still did something
			}
			os.Exit(1)
		}
		fmt.Println("Tor identity rotated via ControlPort NEWNYM.")
	case "doctor":
		if err := healthCheck(cfg, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "doctor failed: %v\n", err)
			os.Exit(1)
		}
	case "env":
		if err := runEnv(cfg, rest); err != nil {
			fmt.Fprintf(os.Stderr, "env failed: %v\n", err)
			os.Exit(1)
		}
	case "watchdog":
		if *flagWatchOnce {
			watchdog.RunOnce(cfg)
			fmt.Println("watchdog: single pass done")
			return
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := watchdog.Run(ctx, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "watchdog failed: %v\n", err)
			os.Exit(1)
		}
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func loadConfig() (config.Config, error) {
	cfg, err := config.Load(*flagConfig)
	if err != nil {
		return cfg, err
	}
	if *flagDebug {
		cfg.Debug = true
	}
	if *flagWatchInt > 0 {
		cfg.WatchdogSeconds = *flagWatchInt
		cfg.WatchdogInterval = time.Duration(*flagWatchInt) * time.Second
	}
	if *flagAutorotate > 0 {
		cfg.AutorotateSeconds = *flagAutorotate
		cfg.AutorotateInterval = time.Duration(*flagAutorotate) * time.Second
	}
	if *flagTorBin != "" {
		cfg.TorBinary = *flagTorBin
	}
	if *flagPrivoxyBin != "" {
		cfg.PrivoxyBinary = *flagPrivoxyBin
	}
	if *flagTorConfig != "" {
		cfg.TorConfigPath = *flagTorConfig
	}
	if *flagPrivoxyConf != "" {
		cfg.PrivoxyConfigPath = *flagPrivoxyConf
	}
	if *flagControlAddr != "" {
		cfg.TorControlAddr = *flagControlAddr
	}
	if *flagControlPass != "" {
		cfg.TorControlPass = *flagControlPass
	}
	return cfg, nil
}

func printUsage() {
	fmt.Println("Usage: torstack [--config FILE] [--debug] <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  start     Start Tor + Privoxy (detached, logs to LogDir)")
	fmt.Println("  stop      Stop Tor + Privoxy")
	fmt.Println("  restart   Restart Tor + Privoxy")
	fmt.Println("  status    Show service status")
	fmt.Println("  rotate    New Tor identity via ControlPort NEWNYM")
	fmt.Println("  doctor [-v]  Health checks (verbose with -v)")
	fmt.Println("  env [--write] [--json] [--shell bash|fish]  Print/write proxy env (XDG)")
	fmt.Println("  watchdog [--once] [--autorotate SECS]  Alive + optional NEWNYM loop (Ctrl-C stops)")
	fmt.Println("  help      Show this help")
	fmt.Println("Global flags:")
	flag.PrintDefaults()
	fmt.Println("Env overrides: TORSTACK_TOR_BINARY, TORSTACK_PRIVOXY_BINARY,")
	fmt.Println("  TORSTACK_TOR_CONFIG, TORSTACK_PRIVOXY_CONFIG, TORSTACK_LOG_DIR,")
	fmt.Println("  TORSTACK_SOCKS_ADDR, TORSTACK_PRIVOXY_ADDR, TORSTACK_CONTROL_ADDR,")
	fmt.Println("  TORSTACK_CONTROL_PASSWORD, TORSTACK_WATCH_INTERVAL, TORSTACK_AUTOROTATE, TORSTACK_DEBUG=1")
}

func startStack(cfg config.Config) error {
	// Orchestrator path: pkg/client wraps tor+privoxy daemons.
	c := client.New(cfg)
	if err := c.Start(); err != nil {
		return fmt.Errorf("tor start: %w", err)
	}
	fmt.Printf("Tor + Privoxy started. Logs: %s/{tor,privoxy}.log\n", cfg.LogDir)
	return nil
}

func stopStack(cfg config.Config) error {
	c := client.New(cfg)
	if err := c.Stop(); err != nil {
		// Preserve legacy UX: privoxy errors go to stderr, tor error returned.
		fmt.Fprintf(os.Stderr, "stop: %v\n", err)
		return fmt.Errorf("tor stop: %w", err)
	}
	fmt.Println("Tor + Privoxy stopped.")
	return nil
}

func stackStatus(cfg config.Config) (string, error) {
	c := client.New(cfg)
	torOK, privOK, err := c.Stats()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Tor: ")
	if torOK {
		b.WriteString("running")
	} else {
		b.WriteString("stopped")
	}
	b.WriteString(" | Privoxy: ")
	if privOK {
		b.WriteString("running")
	} else {
		b.WriteString("stopped")
	}
	return b.String(), nil
}

func healthCheck(cfg config.Config, verbose bool) error {
	torOK, privOK, err := client.New(cfg).Stats()
	if err != nil {
		return err
	}
	if !verbose {
		if !torOK {
			return fmt.Errorf("tor is not running")
		}
		if !privOK {
			return fmt.Errorf("privoxy is not running")
		}
		if err := health.CheckPort(cfg.TorSOCKSAddr, 3*time.Second); err != nil {
			return fmt.Errorf("tor %s unreachable: %w", cfg.TorSOCKSAddr, err)
		}
		if err := health.CheckPort(cfg.PrivoxyAddr, 3*time.Second); err != nil {
			return fmt.Errorf("privoxy %s unreachable: %w", cfg.PrivoxyAddr, err)
		}
		fmt.Printf("OK: Tor (%s) and Privoxy (%s) reachable.\n", cfg.TorSOCKSAddr, cfg.PrivoxyAddr)
		return nil
	}
	results := health.CheckAll(cfg, torOK, privOK)
	failed := 0
	for _, r := range results {
		mark := "OK  "
		if !r.OK {
			mark = "FAIL"
			// control port is optional
			if strings.Contains(r.Name, "optional") {
				mark = "WARN"
			} else {
				failed++
			}
		}
		fmt.Printf("[%s] %-28s %6.0fms  %s\n", mark, r.Name, float64(r.Latency.Microseconds())/1000.0, r.Detail)
	}
	fmt.Printf("processes: tor=%v privoxy=%v\n", torOK, privOK)
	if !torOK || !privOK {
		failed++
	}
	if failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	fmt.Println("doctor: all checks passed")
	fmt.Println("hint: verify DNS leak at https://dnsleaktest.com/ (Extended test) via proxied browser")
	return nil
}

func runEnv(cfg config.Config, rest []string) error {
	write := false
	asJSON := false
	shell := ""
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--write":
			write = true
		case "--json":
			asJSON = true
		case "--shell":
			if i+1 < len(rest) {
				i++
				shell = rest[i]
			}
		}
	}
	if asJSON {
		js, err := env.RenderJSON(cfg)
		if err != nil {
			return err
		}
		fmt.Print(js)
	} else {
		if shell == "" {
			shell = "posix"
		}
		fmt.Print(env.RenderShell(cfg, shell))
	}
	if write {
		shPath, jsonPath, err := env.Write(cfg)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s\nwrote %s\n", shPath, jsonPath)
		fmt.Fprintf(os.Stderr, "use: eval \"$(torstack env)\"  # inject HTTPS :443 TCP ke terminal ini\n")
		fmt.Fprintf(os.Stderr, "use: source %s  # persistent per-session\n", shPath)
	}
	return nil
}

// Ensure signal import is used on all platforms.
var _ = signal.Notify
