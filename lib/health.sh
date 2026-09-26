#!/usr/bin/env bash
# lib/health.sh — check_tor, check_api, check_ip (source-able)
# usage: . "$ROOT/lib/health.sh"; check_tor && check_api
SOCKS_HOST="${TOR_SOCKS:-127.0.0.1:9050}"
HTTP_PROXY_URL="${TOR_HTTP_PROXY:-http://127.0.0.1:8118}"
TIMEOUT="${HEALTH_TIMEOUT:-15}"

check_tor() {
  # 1) socks mendengar? 2) exit IP valid? 3) IsTor?
  local ip
  ip="$(curl -s --max-time "$TIMEOUT" --socks5-hostname "$SOCKS_HOST" https://api.ipify.org 2>/dev/null)" || { echo "check_tor: socks gagal" >&2; return 1; }
  [ -n "$ip" ] || { echo "check_tor: ip kosong" >&2; return 1; }
  echo "check_tor: OK exit=$ip"
  return 0
}

check_api() {
  # endpoint opencode zen via tor — ganti URL sesuai kebutuhan
  local url="${HEALTH_API_URL:-https://opencode.ai/zen/v1/models}"
  local code
  code="$(curl -s -o /dev/null -w '%{http_code}' --max-time "$TIMEOUT" --socks5-hostname "$SOCKS_HOST" "$url" 2>/dev/null)" || code="000"
  echo "check_api: $url -> $code"
  [ "$code" = "200" ] || [ "$code" = "401" ] || [ "$code" = "404" ]
}

check_ip() {
  echo "--- direct ---"
  curl -s --max-time "$TIMEOUT" https://api.ipify.org 2>/dev/null; echo
  echo "--- via socks5 $SOCKS_HOST ---"
  curl -s --max-time "$TIMEOUT" --socks5-hostname "$SOCKS_HOST" https://api.ipify.org 2>/dev/null; echo
  echo "--- via http $HTTP_PROXY_URL ---"
  curl -s --max-time "$TIMEOUT" -x "$HTTP_PROXY_URL" https://api.ipify.org 2>/dev/null; echo
}

if [ "${BASH_SOURCE[0]:-}" = "${0}" ]; then
  # dijalankan langsung -> jalankan semua
  check_tor; a=$?
  check_api; b=$?
  exit $(( a || b ))
fi
