package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime paths and tunables.
// Stdlib only — no external YAML/TOML dependency.
type Config struct {
	TorBinary         string        `json:"tor_binary"`
	PrivoxyBinary     string        `json:"privoxy_binary"`
	TorConfigPath     string        `json:"tor_config"`
	PrivoxyConfigPath string        `json:"privoxy_config"`
	LogDir            string        `json:"log_dir"`
	WatchdogInterval  time.Duration `json:"-"`
	// raw for JSON (seconds) so config file stays simple
	WatchdogSeconds int `json:"watchdog_interval_seconds"`
	// AutorotateInterval is the NEWNYM period; <=0 disables auto-rotate.
	AutorotateInterval time.Duration `json:"-"`
	// raw for JSON (seconds); 0 = off (default: manual `rotate` only)
	AutorotateSeconds int           `json:"autorotate_seconds"`
	TorSOCKSAddr    string `json:"tor_socks_addr"`
	PrivoxyAddr     string `json:"privoxy_addr"`
	TorControlAddr  string `json:"tor_control_addr"`
	TorControlPass  string `json:"tor_control_password"`
	Debug           bool   `json:"debug"`
}

// Default returns sane defaults for Linux + Termux.
func Default() Config {
	termux := IsTermux()
	logDir := filepath.Join(os.TempDir(), "torstack")
	if termux {
		if h, err := os.UserHomeDir(); err == nil {
			logDir = filepath.Join(h, ".torstack", "logs")
		}
	}
	cfg := Config{
		TorBinary:         lookupBinary("tor"),
		PrivoxyBinary:     lookupBinary("privoxy"),
		LogDir:            logDir,
		WatchdogInterval:  15 * time.Second,
		WatchdogSeconds:   15,
		TorSOCKSAddr:      "127.0.0.1:9050",
		PrivoxyAddr:       "127.0.0.1:8118",
		TorControlAddr:    "127.0.0.1:9051",
		TorControlPass:    "",
		Debug:             false,
	}
	cfg.TorConfigPath = firstExisting(
		os.Getenv("TORSTACK_TOR_CONFIG"),
		"/etc/tor/torrc",
		"/data/data/com.termux/files/usr/etc/tor/torrc",
		"/usr/local/etc/tor/torrc",
	)
	cfg.PrivoxyConfigPath = firstExisting(
		os.Getenv("TORSTACK_PRIVOXY_CONFIG"),
		"/etc/privoxy/config",
		"/data/data/com.termux/files/usr/etc/privoxy/config",
		"/usr/local/etc/privoxy/config",
	)
	applyEnv(&cfg)
	return cfg
}

// Load reads a JSON file (if path != "") then applies env overrides.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	// Allow simple flat YAML (key: value) by converting to JSON-ish?
	// We only parse JSON; if it looks like YAML, try minimal converter.
	if !json.Valid(data) {
		data = flatYAMLToJSON(data)
	}
	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}
	merge(&cfg, fileCfg)
	if cfg.WatchdogSeconds > 0 {
		cfg.WatchdogInterval = time.Duration(cfg.WatchdogSeconds) * time.Second
	}
	if cfg.AutorotateSeconds > 0 {
		cfg.AutorotateInterval = time.Duration(cfg.AutorotateSeconds) * time.Second
	}
	applyEnv(&cfg)
	return cfg, nil
}

func merge(dst *Config, src Config) {
	if src.TorBinary != "" {
		dst.TorBinary = src.TorBinary
	}
	if src.PrivoxyBinary != "" {
		dst.PrivoxyBinary = src.PrivoxyBinary
	}
	if src.TorConfigPath != "" {
		dst.TorConfigPath = src.TorConfigPath
	}
	if src.PrivoxyConfigPath != "" {
		dst.PrivoxyConfigPath = src.PrivoxyConfigPath
	}
	if src.LogDir != "" {
		dst.LogDir = src.LogDir
	}
	if src.WatchdogSeconds > 0 {
		dst.WatchdogSeconds = src.WatchdogSeconds
		dst.WatchdogInterval = time.Duration(src.WatchdogSeconds) * time.Second
	}
	if src.AutorotateSeconds > 0 {
		dst.AutorotateSeconds = src.AutorotateSeconds
		dst.AutorotateInterval = time.Duration(src.AutorotateSeconds) * time.Second
	}
	if src.TorSOCKSAddr != "" {
		dst.TorSOCKSAddr = src.TorSOCKSAddr
	}
	if src.PrivoxyAddr != "" {
		dst.PrivoxyAddr = src.PrivoxyAddr
	}
	if src.TorControlAddr != "" {
		dst.TorControlAddr = src.TorControlAddr
	}
	if src.TorControlPass != "" {
		dst.TorControlPass = src.TorControlPass
	}
	if src.Debug {
		dst.Debug = true
	}
}

// applyEnv overrides config from TORSTACK_* environment variables.
// FIX vs Copilot version: uses strings.ToUpper(name) (full upper),
// not just first-letter upper ("Tor" bug).
func applyEnv(cfg *Config) {
	if v := os.Getenv("TORSTACK_TOR_BINARY"); v != "" {
		cfg.TorBinary = v
	}
	if v := os.Getenv("TORSTACK_PRIVOXY_BINARY"); v != "" {
		cfg.PrivoxyBinary = v
	}
	if v := os.Getenv("TORSTACK_TOR_CONFIG"); v != "" {
		cfg.TorConfigPath = v
	}
	if v := os.Getenv("TORSTACK_PRIVOXY_CONFIG"); v != "" {
		cfg.PrivoxyConfigPath = v
	}
	if v := os.Getenv("TORSTACK_LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("TORSTACK_SOCKS_ADDR"); v != "" {
		cfg.TorSOCKSAddr = v
	}
	if v := os.Getenv("TORSTACK_PRIVOXY_ADDR"); v != "" {
		cfg.PrivoxyAddr = v
	}
	if v := os.Getenv("TORSTACK_CONTROL_ADDR"); v != "" {
		cfg.TorControlAddr = v
	}
	if v := os.Getenv("TORSTACK_CONTROL_PASSWORD"); v != "" {
		cfg.TorControlPass = v
	}
	if v := os.Getenv("TORSTACK_WATCH_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.WatchdogSeconds = n
			cfg.WatchdogInterval = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("TORSTACK_AUTOROTATE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.AutorotateSeconds = n
			cfg.AutorotateInterval = time.Duration(n) * time.Second
		}
	}
	if v := os.Getenv("TORSTACK_DEBUG"); v == "1" || strings.EqualFold(v, "true") {
		cfg.Debug = true
	}
}

func lookupBinary(name string) string {
	// FIX: full upper-case env name, e.g. TORSTACK_TOR_BINARY
	if p := os.Getenv("TORSTACK_" + strings.ToUpper(name) + "_BINARY"); p != "" {
		return p
	}
	if bin, err := exec.LookPath(name); err == nil {
		return bin
	}
	return name
}

func firstExisting(values ...string) string {
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	return ""
}

// IsTermux reports whether we run inside Termux on Android.
func IsTermux() bool {
	prefix := os.Getenv("PREFIX")
	if strings.Contains(prefix, "com.termux") {
		return true
	}
	if _, err := os.Stat("/data/data/com.termux/files/usr/bin"); err == nil {
		return true
	}
	return false
}

// flatYAMLToJSON converts a very simple flat "key: value" YAML doc
// into JSON so users can write configs/default.yaml without deps.
// Only supports top-level scalar keys; nested structures are ignored.
func flatYAMLToJSON(yamlData []byte) []byte {
	m := map[string]string{}
	for _, line := range strings.Split(string(yamlData), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)
		if k != "" {
			m[k] = v
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return yamlData
	}
	// Remarshal numbers/bools best-effort via second pass is overkill;
	// JSON loader tolerates strings for paths/addrs. Interval handled via env/JSON.
	return out
}
