# tor

Installer + client Tor (Tor + Privoxy) untuk Termux / Linux. MIT licensed.

Alur verifikasi: `doctor -v` cek handshake SOCKS5 + proxy HTTP + tunnel
HTTPS `:443`, lalu pastikan tidak ada DNS leak via
[Extended test di dnsleaktest.com](https://dnsleaktest.com/) lewat browser
yang memakai proxy ini.

> **Go (utama, direkomendasikan):** single binary tanpa runtime Node/Python —
> `go build -o torstack ./cmd/torstack`. Lihat [Go CLI](#go-cli-satu-binary).
> **Legacy shell/JS:** `bin/torstack` + dashboard masih tersedia di bawah
> ([Legacy](#legacy-shelljs-cli)).

## Go CLI (satu binary)

Manajemen lifecycle Tor + Privoxy, health check protokol asli, watchdog
auto-restart, dan rotasi identitas via ControlPort `SIGNAL NEWNYM`.

### Fitur

- `start` / `stop` / `restart` Tor + Privoxy (detached `setsid`, log ke `LogDir`)
- `status` dengan pencocokan proses eksak (`/proc` → `pgrep -x` → `ps`)
- `doctor [-v]` — TCP + handshake SOCKS5 (9050) + cek HTTP proxy (8118)
- `watchdog [--once]` — auto-restart sadar-konteks (Ctrl-C berhenti bersih)
- `rotate` — `SIGNAL NEWNYM` via ControlPort (fallback SIGHUP dilaporkan, tidak diam-diam)
- Flags + JSON config + override env `TORSTACK_*`, default sadar-Termux

### Syarat

Go 1.22+, `tor`, `privoxy`.

Termux:

```bash
pkg update && pkg install -y git golang tor privoxy
```

Debian/Ubuntu:

```bash
sudo apt update && sudo apt install -y git golang tor privoxy
```

### Install cepat (copy-paste)

```bash
curl -fsSL https://raw.githubusercontent.com/xlfzxr/tor/main/install.sh | bash
torstack doctor -v
```

Manual:

```bash
git clone https://github.com/xlfzxr/tor.git
cd tor
go vet ./...
go build -o torstack ./cmd/torstack
install -m 755 torstack ~/.local/bin/torstack
```

### Pakai

```bash
torstack start
torstack status
torstack doctor -v
torstack rotate
torstack watchdog            # Ctrl-C untuk berhenti
torstack watchdog --once     # sekali jalan
torstack --config configs/default.json status
torstack --watch-interval 10 --debug doctor -v
```

### Verifikasi DNS leak

1. `torstack doctor -v` — semua cek harus pass (termasuk `https :443 via proxy`).
2. `eval "$(torstack env)"` — suntik proxy ke terminal ini.
3. Buka https://dnsleaktest.com/ → **Extended test** lewat browser yang
   memakai proxy → server yang muncul harus milik exit relay Tor,
   bukan ISP kamu.

### Konfigurasi

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

### Dev Go

```bash
go vet ./...
go build -o torstack ./cmd/torstack
./torstack doctor -v
make cross   # linux-amd64/arm64, android-arm64 di dist/
```

---

## Legacy shell/JS CLI

`torstack` CLI: start/stop/status/rotate/logs, plus watchdog, autorotate,
web dashboard realtime (SSE), health check, dan fallback helper.

### Struktur (1 file = 1 fungsi)

```
torstack/
├── bootstrap.mjs          # generator tree (sumber utama install)
├── installer.sh           # installer system-wide
├── package.json biome.json# tooling lint JS (biome) — dev only
├── tools/
│   ├── biome              # wrapper biome untuk Termux (glibc loader)
│   └── lint-shell.sh      # bash -n + shellcheck + py_compile
├── bin/
│   ├── torstack           # CLI utama (dispatcher subcommand)
│   ├── tor-rotate         # orkestrasi NEWNYM (protokol di lib/torctl.py)
│   ├── tor-watchdog       # jaga tor+privoxy hidup
│   ├── tor-events         # daemon listener ControlPort (via torctl.py)
│   ├── tor-autorotate     # scheduler rotate + JS hook runner
│   └── tor-dashboard      # web UI server (Bun; halaman di lib/...)
├── lib/
│   ├── torctl.py          # Tor ControlPort client (newnym/listen)
│   ├── events.sh          # event bus: emit/emit_kv -> state/events.jsonl
│   ├── health.sh          # check_tor, check_api, check_ip
│   ├── fallback.sh        # fetch_with_fallback (retry/rotate/direct)
│   ├── shim.sh            # panggil antar-script via bash eksplisit
│   └── dashboard-page.ts  # halaman HTML dashboard (di-import server)
├── hooks/
│   ├── pre-rotate.mjs     # guard sebelum NEWNYM (exit!=0 = batal)
│   └── post-rotate.mjs    # hook user tiap rotate sukses (editable)
├── conf/
│   ├── torrc.example        # TEMPLATE portable (di-commit)
│   ├── privoxy.conf.example # TEMPLATE portable (di-commit)
│   ├── torrc                # GENERATED — jangan edit/commit!
│   └── privoxy.conf         # GENERATED — jangan edit/commit!
├── profiles/
│   ├── opencode.env       # HTTP_PROXY 8118 + timeout long
│   └── crawl.env          # SOCKS5 9050 langsung
└── tests/
    └── test-curl.sh       # integration test (butuh tor jalan)
```

### Quickstart legacy (Termux)

```bash
# 1. generate conf + shebang lokal (wajib setelah clone fresh)
bun ~/torstack/bootstrap.mjs ~/torstack

# 2. start
~/torstack/bin/torstack start
~/torstack/bin/torstack status

# 3. pakai proxy
export HTTPS_PROXY=http://127.0.0.1:8118 HTTP_PROXY=http://127.0.0.1:8118
curl https://api.ipify.org

# rotate identity
~/torstack/bin/torstack rotate

# dashboard realtime (buka browser otomatis di Termux)
~/torstack/bin/torstack web

# lihat event bus / log listener
~/torstack/bin/torstack events --follow
~/torstack/bin/torstack logs events 20
```

> **Catatan:** `conf/torrc` dan `conf/privoxy.conf` adalah file GENERATED
> (di-ignore git) yang selalu di-render ulang dari `*.example` setiap
> `torstack start` / `bun bootstrap.mjs`. Aman di-clone di mesin mana pun
> (Termux, Ubuntu, dll) — `confdir` privoxy terdeteksi otomatis bila ada,
> dihapus bila tidak ada.
>
> **Kustomisasi conf:** edit `conf/*.example` (bukan file generated —
> edit manual di sana akan tertimpa saat `start`), lalu jalankan
> `torstack start` untuk me-render ulang.
>
> **Shebang:** yang di-commit `#!/usr/bin/env bash` (standar Linux).
> Bootstrap me-rewrite ke bash lokal saat install — jangan commit hasilnya
> (`git checkout -- bin lib` untuk membuangnya).

### Dependensi legacy

`tor`, `privoxy`, `python3` (wajib — `tor-rotate`/`tor-events` via
`lib/torctl.py`), `bun` (opsional, untuk `bootstrap.mjs` + dashboard;
tanpa bun, conf tetap di-render saat `start`).

### Install legacy system-wide

```bash
./installer.sh --help
./installer.sh                      # Termux: pakai $PREFIX otomatis
sudo ./installer.sh                 # Linux: /usr/local + /etc/torstack
```

Binary ter-install sebagai **symlink** ke `$INSTALL_DIR/bin`, jadi `ROOT`
tetap menunjuk ke tree install (ikut symlink via `readlink -f`).
Untuk layout custom: `TORSTACK_ROOT=/path/ke/tree torstack status`.

### Profiles

```bash
set -a; . ~/torstack/profiles/opencode.env; set +a  # untuk opencode CLI
set -a; . ~/torstack/profiles/crawl.env; set +a     # untuk crawler via SOCKS
```

### Dev legacy: lint & test

```bash
bun install          # biome (di Termux: binary via glibc, lihat tools/biome)
bun run test         # shell lint + biome check (dipakai CI juga)
bun run check:all    # + cek format bin/tor-dashboard
bash tests/test-curl.sh   # integration (butuh tor jalan; SKIP bila tanpa cookie)
```

### CI & Rilis

- `lint-shell`: `bash -n` + `shellcheck` (10 file) + `py_compile`
- `lint-js`: `biome check` (dashboard advisory)
- `test-health`: smoke tanpa network
- tag `v*` → tarball dari `git archive` + GitHub Release
- versi: `torstack version` (sinkron `package.json` + tag)
