#!/usr/bin/env bash
# lib/shim.sh — runtime shim: panggil script via bash eksplisit, abaikan shebang.
# Masalah: #!/usr/bin/env bash mati di Termux (tak ada /usr/bin/env),
# shebang hardcode Termux mati di Linux. Bootstrap rewrite shebang (build-time)
# menutup kasus eksekusi langsung, tapi panggilan ANTAR-script harus lewat sini.
# usage:
#   . "$ROOT/lib/shim.sh"          # sediakan $TORSTACK_BASH
#   "$TORSTACK_BASH" "$ROOT/bin/tor-rotate" --once
#   run_script "$ROOT/bin/tor-rotate" --once   # versi exec (gantikan proses)
# env: TORSTACK_BASH (override manual), PREFIX (Termux)
if [ -z "${TORSTACK_BASH:-}" ]; then
  for __shim_b in "${PREFIX:+$PREFIX/bin/bash}" \
      /data/data/com.termux/files/usr/bin/bash /bin/bash /usr/bin/bash bash; do
    [ -n "${__shim_b:-}" ] || continue
    if command -v "$__shim_b" >/dev/null 2>&1 || [ -x "$__shim_b" ]; then
      TORSTACK_BASH="$__shim_b"
      break
    fi
  done
  unset __shim_b
  : "${TORSTACK_BASH:=bash}"
fi
export TORSTACK_BASH

# run_script <script> [args...] — exec via bash eksplisit (tak kembali)
run_script() { exec "$TORSTACK_BASH" "$@"; }
# call_script <script> [args...] — panggil via bash eksplisit (kembali)
call_script() { "$TORSTACK_BASH" "$@"; }
