# Changelog torstack

Format: `Added / Changed / Fixed` per versi. Tag rilis: `vX.Y.Z`.

## [0.2.0] — 2026-09-26

### Added

- Event bus realtime (`lib/events.sh` + `state/events.jsonl`): semua
  komponen emit event, dashboard terima via SSE tanpa polling.
- `bin/tor-events`: listener Tor ControlPort (CIRC/STREAM/BW/NOTICE).
- `bin/tor-autorotate`: scheduler rotate + JS hook
  (`hooks/pre-rotate.mjs`, `hooks/post-rotate.mjs`).
- `bin/tor-dashboard` + `torstack web [--port N] [--host ADDR] [--no-open]`:
  web UI localhost (status, rotate, logs, edit hook).
- `torstack events [n] [--follow] [--type TYPE]`,
  `torstack logs events [n]`.
- `lib/torctl.py`: Tor ControlPort client (1 file, dipakai
  tor-rotate + tor-events).
- `lib/shim.sh`: panggil antar-script via bash eksplisit (abaikan shebang).
- Lint JS: biome (`bun run check`), lint shell: `tools/lint-shell.sh`
  (`bun run test` = keduanya). Berlaku di CI.
- `torstack version`, `LICENSE` (MIT), CHANGELOG ini.

### Changed

- `lib/dashboard-page.ts`: HTML dashboard dipisah dari server.
- Tarball rilis: dibangun dari `git archive` (selalu sinkron dengan repo).

### Fixed

- Shebang portabel: yang di-commit `#!/usr/bin/env bash`, bootstrap
  rewrite ke bash lokal saat install (jangan commit hasil rewrite).
- Template `conf/*.example` portable (`{{ROOT}}`, `{{CONFDIR_LINE}}`),
  conf real selalu di-render ulang saat start.
- ROOT ikut symlink install + override `TORSTACK_ROOT`.

## [0.1.0] — awal

- `torstack` CLI: start/stop/restart/status/rotate/logs.
- `tor-rotate` (SIGNAL NEWNYM), `tor-watchdog`, `lib/health.sh`,
  `lib/fallback.sh`, `installer.sh`, `profiles/*.env`.
