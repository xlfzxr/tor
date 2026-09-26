#!/usr/bin/env bash
# tools/lint-shell.sh — lint semua shell script (dipakai CI + `bun run test`).
# Daftar file = semua yang executable shell di bin/ + lib/*.sh + installer.sh.
# tor-dashboard dikecualikan (itu Bun/JS, di-lint biome via stdin).
set -euo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

FILES=(
  bin/torstack
  bin/tor-rotate
  bin/tor-watchdog
  bin/tor-events
  bin/tor-autorotate
  lib/health.sh
  lib/fallback.sh
  lib/events.sh
  lib/shim.sh
  installer.sh
)

fail=0
echo "[lint-shell] versions: bash=$(bash --version | head -n1) | $(command -v shellcheck >/dev/null && shellcheck --version | head -n2 | tr '\n' ' ' || echo 'shellcheck MISSING') | $(command -v python3 >/dev/null && python3 --version 2>&1 || echo 'python3 MISSING')"
echo "[lint-shell] bash -n ..."
for f in "${FILES[@]}"; do
  if [ ! -f "$f" ]; then
    echo "HILANG: $f"
    fail=1
    continue
  fi
  bash -n "$f" || fail=1
done

if command -v python3 >/dev/null 2>&1; then
  echo "[lint-shell] python compile (lib/*.py) ..."
  python3 -m py_compile lib/*.py || fail=1
else
  echo "[lint-shell] python3 tidak ada, skip py_compile"
fi

if command -v shellcheck >/dev/null 2>&1; then
  echo "[lint-shell] shellcheck ..."
  shellcheck -x "${FILES[@]}" || fail=1
else
  echo "[lint-shell] shellcheck tidak ada, skip (install: pkg install shellcheck / apt install shellcheck)"
fi

if [ "$fail" -eq 0 ]; then
  echo "[lint-shell] OK: ${#FILES[@]} files"
else
  echo "[lint-shell] GAGAL" >&2
fi
exit "$fail"
