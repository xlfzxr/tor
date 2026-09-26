#!/usr/bin/env bash
# installer.sh — install torstack system-wide / Termux ($PREFIX)
# usage:
#   ./installer.sh [--prefix DIR] [--repo URL] [--no-logrotate]
# env:
#   REPO_URL   default https://github.com/xlfzxr/tor.git
#   PREFIX     Termux: $PREFIX, Linux: "" (= /usr/local, /etc, ...)
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/xlfzxr/tor.git}"
DO_LOGROTATE=1
CUSTOM_PREFIX=""

usage() {
    echo "usage: $0 [--prefix DIR] [--repo URL] [--no-logrotate]"
    echo "  --prefix DIR   install root (default: \$PREFIX atau /usr/local)"
    echo "  --repo URL     repo torstack (default: $REPO_URL)"
    echo "  --no-logrotate lewati setup logrotate"
}

while [ $# -gt 0 ]; do
    case "$1" in
        --prefix) CUSTOM_PREFIX="${2:-}"; shift 2 ;;
        --repo) REPO_URL="${2:-}"; shift 2 ;;
        --no-logrotate) DO_LOGROTATE=0; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown arg: $1" >&2; usage; exit 1 ;;
    esac
done

BASE="${CUSTOM_PREFIX:-${PREFIX:-/usr/local}}"
if [ -n "$CUSTOM_PREFIX" ]; then
    # custom prefix (test / install non-root): semua di bawah BASE
    INSTALL_DIR="$BASE/opt/torstack"
    BIN_DIR="$BASE/bin"
    CONF_DIR="$BASE/etc/torstack"
    LOG_DIR="$BASE/var/log/torstack"
    STATE_DIR="$BASE/var/lib/torstack"
    IS_TERMUX=0
elif [ -n "${PREFIX:-}" ]; then
    # Termux layout
    INSTALL_DIR="$BASE/opt/torstack"
    BIN_DIR="$BASE/bin"
    CONF_DIR="$BASE/etc/torstack"
    LOG_DIR="$BASE/var/log/torstack"
    STATE_DIR="$BASE/var/lib/torstack"
    IS_TERMUX=1
else
    # Linux layout (BASE=/usr/local default)
    INSTALL_DIR="$BASE/opt/torstack"
    BIN_DIR="$BASE/bin"
    CONF_DIR="/etc/torstack"
    LOG_DIR="/var/log/torstack"
    STATE_DIR="/var/lib/torstack"
    IS_TERMUX=0
fi

log() { printf '\033[1;32m[install]\033[0m %s\n' "$*"; }
err() { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; }

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || { err "Missing dependency: $1"; exit 1; }
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

main() {
    log "Repo: $REPO_URL"
    log "Target: INSTALL=$INSTALL_DIR BIN=$BIN_DIR CONF=$CONF_DIR"

    log "Checking dependencies..."
    require_cmd tor
    require_cmd privoxy
    require_cmd python3
    command -v bun >/dev/null 2>&1 || log "WARN: bun tidak ada — lewati regenerate via bootstrap (torstack start tetap auto-render)"

    log "Creating directories..."
    mkdir -p "$INSTALL_DIR" "$BIN_DIR" "$CONF_DIR" "$LOG_DIR" "$STATE_DIR"

    log "Copying files..."
    cp -r "$SCRIPT_DIR/bin" "$SCRIPT_DIR/lib" "$SCRIPT_DIR/conf" "$SCRIPT_DIR/profiles" "$SCRIPT_DIR/hooks" "$INSTALL_DIR/"
    # contoh template ikut ter-copy via conf/; pastikan executable bit.
    # .py/.ts = library (dipanggil via python3/bun), bukan executable.
    chmod 755 "$INSTALL_DIR/bin/"* "$INSTALL_DIR/lib/"*.sh 2>/dev/null || true
    chmod 644 "$INSTALL_DIR/lib/"*.py "$INSTALL_DIR/lib/"*.ts 2>/dev/null || true

    # regenerate conf untuk path target bila bun tersedia
    if command -v bun >/dev/null 2>&1 && [ -f "$SCRIPT_DIR/bootstrap.mjs" ]; then
        log "Regenerating conf via bootstrap.mjs..."
        bun "$SCRIPT_DIR/bootstrap.mjs" "$INSTALL_DIR" >/dev/null
    else
        log "WARN: pakai conf dari repo; akan di-render ulang saat 'torstack start'"
    fi
    cp "$INSTALL_DIR/conf/torrc" "$CONF_DIR/torrc" 2>/dev/null || cp "$SCRIPT_DIR/conf/torrc.example" "$CONF_DIR/torrc.example" || true
    cp "$INSTALL_DIR/conf/privoxy.conf" "$CONF_DIR/privoxy.conf" 2>/dev/null || true

    log "Linking binaries (symlink -> \$INSTALL_DIR, agar ROOT tetap benar)..."
    for b in torstack tor-watchdog tor-rotate tor-autorotate tor-dashboard tor-events; do
        ln -sf "$INSTALL_DIR/bin/$b" "$BIN_DIR/$b"
    done

    if [ "$DO_LOGROTATE" -eq 1 ]; then
        if [ "$IS_TERMUX" -eq 1 ]; then
            log "Termux: lewati logrotate system (tidak ada /etc/logrotate.d). Pakai 'torstack logs' + watchdog."
        elif [ -d /etc/logrotate.d ] && [ -w /etc/logrotate.d ]; then
            log "Setting up log rotation..."
            cat > /etc/logrotate.d/torstack <<EOF
$LOG_DIR/*.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 640 root root
}
EOF
        else
            log "WARN: /etc/logrotate.d tidak writable — lewati (jalankan dengan sudo bila perlu)"
        fi
    fi

    log "Installation complete."
    log "Run 'torstack start' to launch Tor + Privoxy."
    log "Config: $CONF_DIR/torrc, $CONF_DIR/privoxy.conf (runtime: $INSTALL_DIR/conf/)"
    log "Logs: $LOG_DIR/ (dev default: <repo>/logs/)"
}

main "$@"
