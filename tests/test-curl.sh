#!/usr/bin/env bash
# tests/test-curl.sh — unit test REAL usage (bukan mock): curl direct vs
# socks5 tor vs http privoxy, plus lib/health.sh, lib/fallback.sh, tor-rotate.
# usage: bash tests/test-curl.sh
# catatan Termux: tidak ada /usr/bin/env -> jalankan via `bash tests/...`
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
. "$ROOT/lib/health.sh"
# shellcheck disable=SC1091
. "$ROOT/lib/fallback.sh"

SOCKS="${TOR_SOCKS:-127.0.0.1:9050}"
HTTP="${TOR_HTTP_PROXY:-http://127.0.0.1:8118}"
T=15 # curl max-time per request

PASS=0
FAIL=0

ok()   { PASS=$((PASS+1)); printf '  \033[1;32mPASS\033[0m %s\n' "$*"; }
fail() { FAIL=$((FAIL+1)); printf '  \033[1;31mFAIL\033[0m %s\n' "$*"; }
skip() { printf '  \033[1;33mSKIP\033[0m %s\n' "$*"; }

is_ipv4() { [[ "${1:-}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; }

section() { printf '\n== %s ==\n' "$*"; }

# 1. direct (bypass proxy env global) — IP asli ISP
section "1. direct curl (noproxy)"
DIRECT_IP="$(curl -s --noproxy '*' --max-time "$T" https://api.ipify.org 2>/dev/null || true)"
if is_ipv4 "$DIRECT_IP"; then ok "direct IP=$DIRECT_IP"; else fail "direct gagal/ bukan IPv4 (got: ${DIRECT_IP:0:80})"; fi

# 2. via socks5 tor — harus IPv4 valid DAN beda dari direct
section "2. socks5 tor ($SOCKS)"
SOCKS_IP="$(curl -s --max-time "$T" --socks5-hostname "$SOCKS" https://api.ipify.org 2>/dev/null || true)"
if is_ipv4 "$SOCKS_IP"; then ok "socks IP=$SOCKS_IP"; else fail "socks gagal (got: ${SOCKS_IP:0:80})"; fi
if is_ipv4 "$SOCKS_IP" && is_ipv4 "$DIRECT_IP" && [ "$SOCKS_IP" != "$DIRECT_IP" ]; then
  ok "socks IP beda dari direct (anonim)"
elif is_ipv4 "$SOCKS_IP" && [ "$SOCKS_IP" = "$DIRECT_IP" ]; then
  fail "socks IP SAMA dengan direct — bocor!"
fi

# 3. via http privoxy — IPv4 valid (exit boleh beda circuit, jadi cukup valid)
section "3. http privoxy ($HTTP)"
HTTP_IP="$(curl -s --max-time "$T" -x "$HTTP" https://api.ipify.org 2>/dev/null || true)"
if is_ipv4 "$HTTP_IP"; then ok "http-proxy IP=$HTTP_IP"; else fail "http-proxy gagal (got: ${HTTP_IP:0:80})"; fi

# 4. lib/health.sh — check_tor + check_api (real network)
section "4. lib/health.sh"
if check_tor >/dev/null 2>&1; then ok "check_tor"; else fail "check_tor"; fi
if check_api >/dev/null 2>&1; then ok "check_api"; else fail "check_api"; fi

# 5. lib/fallback.sh — sukses via tor
section "5. fetch_with_fallback (sukses via tor)"
FB_OUT="$(MAX_RETRY=2 ALLOW_DIRECT=0 fetch_with_fallback "https://api.ipify.org" 2>/dev/null || true)"
if is_ipv4 "$FB_OUT"; then ok "fallback via tor IP=$FB_OUT"; else fail "fallback via tor (got: ${FB_OUT:0:80})"; fi

# 6. lib/fallback.sh — socks mati: gagal tertutup (ALLOW_DIRECT=0), lalu
#    fallback direct (ALLOW_DIRECT=1) berhasil
section "6. fetch_with_fallback (socks mati)"
export TOR_SOCKS="127.0.0.1:9" SOCKS_HOST="127.0.0.1:9"
if MAX_RETRY=2 ALLOW_DIRECT=0 fetch_with_fallback "https://api.ipify.org" >/dev/null 2>&1; then
  fail "seharusnya GAGAL saat socks mati + ALLOW_DIRECT=0"
else
  ok "gagal tertutup saat socks mati + ALLOW_DIRECT=0"
fi
FB_DIRECT="$(MAX_RETRY=1 ALLOW_DIRECT=1 fetch_with_fallback "https://api.ipify.org" 2>/dev/null || true)"
if [ -n "$FB_DIRECT" ]; then ok "ALLOW_DIRECT=1 fallback berhasil"; else fail "ALLOW_DIRECT=1 tetap gagal"; fi
unset TOR_SOCKS
SOCKS_HOST="${TOR_SOCKS:-127.0.0.1:9050}"

# 7. tor-rotate — NEWNYM beneran, stamp ter-update, IP sesudahnya valid.
# Cookie diambil dari TOR_COOKIE_FILE bila diset (mis. tor milik install
# lain), kalau tidak ada cookie sama sekali test ini di-SKIP (bukan FAIL).
section "7. tor-rotate (NEWNYM real)"
COOKIE_EFF="${TOR_COOKIE_FILE:-$ROOT/state/tor-data/control_auth_cookie}"
if [ ! -f "$COOKIE_EFF" ]; then
  skip "tanpa cookie control ($COOKIE_EFF) — tor repo tidak jalan, lewati rotate"
else
BEFORE_ROTATE="$(cat "$ROOT/state/last-rotate" 2>/dev/null || echo 0)"
ROT_OUT="${TMPDIR:-$ROOT/state}/.torstack-rot-out.txt"
# via `bash` eksplisit: checkout repo pakai shebang #!/usr/bin/env bash
# (standar Linux); di Termux /usr/bin/env tidak ada, jadi exec langsung
# gagal — copy hasil `bun bootstrap.mjs` sudah di-rewrite ke bash lokal.
if ROTATE_WAIT=1 bash "$ROOT/bin/tor-rotate" >"$ROT_OUT" 2>&1; then
  if grep -q "OK NEWNYM" "$ROT_OUT"; then ok "tor-rotate OK NEWNYM"; else fail "tor-rotate tanpa OK NEWNYM"; fi
else
  fail "tor-rotate exit != 0"
fi
AFTER_ROTATE="$(cat "$ROOT/state/last-rotate" 2>/dev/null || echo 0)"
if [ -n "$AFTER_ROTATE" ] && [ "$AFTER_ROTATE" != "$BEFORE_ROTATE" ]; then
  ok "stamp last-rotate ter-update ($AFTER_ROTATE)"
else
  fail "stamp last-rotate tidak berubah"
fi
sleep 5 # beri waktu circuit baru
NEW_IP="$(curl -s --max-time "$T" --socks5-hostname "$SOCKS" https://api.ipify.org 2>/dev/null || true)"
if is_ipv4 "$NEW_IP"; then ok "IP pasca-rotate valid ($NEW_IP)"; else fail "IP pasca-rotate invalid (got: ${NEW_IP:0:80})"; fi
rm -f "$ROT_OUT"
fi # cookie ada

# 8. ROOT resolution: symlink install + TORSTACK_ROOT override (tanpa network)
section "8. ROOT via symlink / TORSTACK_ROOT"
LINKDIR="${TMPDIR:-$ROOT/state}/linktest"
mkdir -p "$LINKDIR"
ln -sf "$ROOT/bin/torstack" "$LINKDIR/torstack"
LINK_ROOT="$(bash "$LINKDIR/torstack" status 2>/dev/null | awk -F': ' '/^root :/{print $2}')"
if [ "$LINK_ROOT" = "$ROOT" ]; then ok "symlink resolve ke ROOT repo ($LINK_ROOT)"; else fail "symlink ROOT salah (got: $LINK_ROOT)"; fi
OVR_ROOT="$(TORSTACK_ROOT=/foo/bar bash "$ROOT/bin/torstack" status 2>/dev/null | awk -F': ' '/^root :/{print $2}')"
if [ "$OVR_ROOT" = "/foo/bar" ]; then ok "TORSTACK_ROOT override bekerja"; else fail "TORSTACK_ROOT diabaikan (got: $OVR_ROOT)"; fi
rm -rf "$LINKDIR"

printf '\n==== hasil: %d PASS, %d FAIL ====\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]