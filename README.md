# tor

Installer + client Tor (Tor + Privoxy) untuk Termux / Linux — **murni Go**,
single binary tanpa runtime Node/Python. MIT licensed.

## Fitur

- `start` / `stop` / `restart` Tor + Privoxy (detached, log ke `LogDir`)
- `status` dengan pencocokan proses eksak
- `doctor [-v]` — TCP + handshake SOCKS5 (9050) + cek HTTP proxy (8118) + tunnel HTTPS `:443`
- `env [--write] [--json] [--shell bash|fish]` — cetak/tulis proxy env (XDG)
- `watchdog [--once]` — auto-restart sadar-konteks
- `rotate` — `SIGNAL NEWNYM` via ControlPort (fallback SIGHUP dilaporkan, tidak diam-diam)

## Syarat

Go 1.22+, `tor`, `privoxy`.

Termux:

```bash
pkg update && pkg install -y git golang tor privoxy
```

Debian/Ubuntu:

```bash
sudo apt update && sudo apt install -y git golang tor privoxy
```

## Install

```bash
git clone https://github.com/xlfzxr/tor.git
cd tor
make install   # build + pasang ke ~/.local/bin/torstack
torstack doctor -v
```

## Pakai

```bash
torstack start
torstack status
torstack doctor -v
torstack rotate
torstack env                    # export proxy untuk shell ini
eval "$(torstack env)"          # suntik proxy ke terminal ini
torstack watchdog               # Ctrl-C untuk berhenti
torstack --config configs/default.json status
```

## Verifikasi DNS leak

1. `torstack doctor -v` — semua cek harus pass (termasuk `https :443 via proxy`).
2. `eval "$(torstack env)"` — suntik proxy ke terminal ini.
3. Buka https://dnsleaktest.com/ → **Extended test** lewat browser yang
   memakai proxy → server yang muncul harus milik exit relay Tor,
   bukan ISP kamu.

## Konfigurasi

Prioritas: flags > env > file JSON `--config` > default bawaan.

| Flag | Env | Default |
|---|---|---|
| `--tor-binary` | `TORSTACK_TOR_BINARY` | `tor` (via PATH) |
| `--privoxy-binary` | `TORSTACK_PRIVOXY_BINARY` | `privoxy` |
| `--tor-config` | `TORSTACK_TOR_CONFIG` | `/etc/tor/torrc`, path Termux, … |
| `--privoxy-config` | `TORSTACK_PRIVOXY_CONFIG` | `/etc/privoxy/config`, path Termux, … |
| `--watch-interval N` | `TORSTACK_WATCH_INTERVAL` | `15` (detik) |
| `--control-addr` | `TORSTACK_CONTROL_ADDR` | `127.0.0.1:9051` |
| `--control-pass` | `TORSTACK_CONTROL_PASSWORD` | kosong |
| `--debug` | `TORSTACK_DEBUG=1` | false |

Contoh: `configs/default.json`.

## Dev

```bash
go vet ./...
go build -o torstack ./cmd/torstack
go test ./...
make cross   # linux-amd64/arm64, android-arm64 di dist/
```

Struktur:

```
tor/
├── cmd/torstack/      # CLI: start/stop/status/rotate/doctor/env/watchdog
├── internal/          # config, tor, privoxy, health, watchdog, env, system
├── pkg/
│   ├── cell/          # Tor fixed cell 512/514 encode/decode
│   ├── protocol/      # handshake Versions/Netinfo
│   ├── netlayer/      # DialTCP/DialTLS relay
│   ├── circuit/       # manager + expiry MaxCircuitDirtiness
│   ├── client/        # orchestrator start/stop/stats/rotate
│   ├── control/       # ControlPort NEWNYM
│   ├── socks/         # SOCKS5 handshake + CONNECT
│   ├── onion/         # onion-service config
│   ├── pt/            # pluggable transports
│   └── fw/            # egress policy
└── configs/           # contoh JSON config
```

## CI & Rilis

- `Go`: `go build` + `go vet` + `go test` di Ubuntu
- tag `v*` → tarball dari `git archive` + GitHub Release
