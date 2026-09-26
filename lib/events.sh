#!/usr/bin/env bash
# lib/events.sh — event bus pusat torstack (client-only).
# Semua emitter append JSONL ke state/events.jsonl, dashboard tail + broadcast SSE.
# Format: {"ts":<ms>,"type":"...","source":"...","data":{...}}
# usage: . "$ROOT/lib/events.sh"; emit rotate.ok '{"old":"1.1.1.1","new":"2.2.2.2"}'
# env: TORSTACK_ROOT / ROOT (wajib salah satu), TORSTACK_SOURCE (opsional override source)
: "${ROOT:?lib/events.sh butuh \$ROOT}"

_EVENTS_FILE="${TORSTACK_EVENTS:-$ROOT/state/events.jsonl}"
_EVENTS_MAX_LINES="${TORSTACK_EVENTS_MAX_LINES:-5000}"
_EVENTS_TRIM_TO="${TORSTACK_EVENTS_TRIM_TO:-2000}"

_events_ensure() { mkdir -p "$(dirname "$_EVENTS_FILE")" 2>/dev/null || true; }

_events_trim() {
  # trim murah: hanya bila file > ~1MB, potong ke N baris terakhir
  local size lines tmp
  size="$(wc -c <"$_EVENTS_FILE" 2>/dev/null || echo 0)"
  [ "${size:-0}" -gt 1048576 ] || return 0
  lines="$(wc -l <"$_EVENTS_FILE" 2>/dev/null || echo 0)"
  [ "${lines:-0}" -gt "$_EVENTS_MAX_LINES" ] || return 0
  tmp="$_EVENTS_FILE.tmp.$$"
  tail -n "$_EVENTS_TRIM_TO" "$_EVENTS_FILE" > "$tmp" 2>/dev/null && mv "$tmp" "$_EVENTS_FILE"
  rm -f "$tmp" 2>/dev/null || true
}

# emit <type> [json-data] [source]
emit() {
  local type="${1:?emit butuh <type>}"
  # NOTE: jangan pakai ${2:-{}} — kena brace expansion di bash (nambah '}').
  local data="${2-}"
  [ -n "$data" ] || data="{}"
  local src="${3-}"
  [ -n "$src" ] || src="${TORSTACK_SOURCE:-${0##*/}}"
  # validasi ringan: data harus mulai dengan { atau [ atau " ; selain itu bungkus jadi string
  case "$data" in
    \{*|\[*|\"*|*[0-9]*|true|false|null) : ;;
    *) data="\"$data\"" ;;
  esac
  local ts
  ts="$(date +%s%3N 2>/dev/null || date +%s000)"
  # pastikan numerik
  case "$ts" in *[!0-9]*) ts="$(date +%s)000" ;; esac
  _events_ensure
  printf '{"ts":%s,"type":"%s","source":"%s","data":%s}\n' \
    "$ts" "$type" "$src" "$data" >> "$_EVENTS_FILE" 2>/dev/null || true
  # trim di background agar tidak blokir rotate cepat (best effort)
  ( _events_trim ) >/dev/null 2>&1 &
  return 0
}

# helper: emit dengan key=value sederhana -> {"k":"v",...} tanpa butuh jq
# usage: emit_kv rotate.ok old=1.1.1.1 new=2.2.2.2
emit_kv() {
  local type="$1"; shift
  local json="{"
  local first=1 kv k v
  for kv in "$@"; do
    k="${kv%%=*}"; v="${kv#*=}"
    # escape " dan \
    v="${v//\\/\\\\}"; v="${v//\"/\\\"}"
    [ "$first" -eq 1 ] && first=0 || json="$json,"
    json="$json\"$k\":\"$v\""
  done
  json="$json}"
  emit "$type" "$json"
}

if [ "${BASH_SOURCE[0]:-}" = "${0}" ]; then
  # dijalankan langsung: lib/events.sh <type> [json]
  emit "$@"
fi
