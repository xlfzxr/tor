#!/usr/bin/env bash
# lib/fallback.sh — retry / rotate / direct (dengan warning)
# usage:
#   . "$ROOT/lib/fallback.sh"
#   fetch_with_fallback "https://api.ipify.org"
# env: ALLOW_DIRECT=0/1 (default 0 = jangan direct), MAX_RETRY=2
ALLOW_DIRECT="${ALLOW_DIRECT:-0}"
MAX_RETRY="${MAX_RETRY:-2}"
SOCKS_HOST="${TOR_SOCKS:-127.0.0.1:9050}"
# runtime shim — rotate dipanggil via bash eksplisit (abaikan shebang)
_FB_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." 2>/dev/null && pwd || echo "")"
if [ -n "$_FB_ROOT" ] && [ -f "$_FB_ROOT/lib/shim.sh" ]; then . "$_FB_ROOT/lib/shim.sh" 2>/dev/null || true; fi
: "${TORSTACK_BASH:=bash}"

_warn() { printf '[fallback][WARN] %s\n' "$*" >&2; }
_info() { printf '[fallback] %s\n' "$*" >&2; }

_fetch_socks() { curl -s --max-time "${CURL_TIMEOUT:-30}" --socks5-hostname "$SOCKS_HOST" "$@"; }
_fetch_direct() { curl -s --max-time "${CURL_TIMEOUT:-30}" "$@"; }

fetch_with_fallback() {
  local url="$1"; shift || true
  local attempt=0
  while [ "$attempt" -lt "$MAX_RETRY" ]; do
    attempt=$((attempt+1))
    _info "try $attempt/$MAX_RETRY via tor: $url"
    if _fetch_socks "$url" "$@"; then return 0; fi
    _warn "gagal via tor (attempt $attempt)"
    # rotate tiap gagal (kecuali attempt terakhir)
    if [ "$attempt" -lt "$MAX_RETRY" ] && [ -x "$(dirname "${BASH_SOURCE[0]}")/../bin/tor-rotate" ]; then
      _info "rotate identity..."
      "$TORSTACK_BASH" "$(dirname "${BASH_SOURCE[0]}")/../bin/tor-rotate" >/dev/null 2>&1 || _warn "rotate gagal, lanjut retry"
    fi
    sleep 2
  done
  if [ "$ALLOW_DIRECT" = "1" ]; then
    _warn "SEMUA via-tor gagal — FALLBACK DIRECT (tanpa anonimitas!) untuk: $url"
    _fetch_direct "$url" "$@"
    return $?
  else
    _warn "SEMUA via-tor gagal. Set ALLOW_DIRECT=1 untuk fallback direct (tidak anonim)."
    return 1
  fi
}
