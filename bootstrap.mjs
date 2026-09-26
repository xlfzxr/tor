#!/usr/bin/env bun
// bootstrap.mjs — install/copy torstack tree ke target ROOT.
// Sumber utama = file repo (sibling script ini). TIDAK ada template embedded
// di sini (dulu ada, stale, menimpa fix) — semua disalin dari repo.
// usage:
//   bun ~/torstack/bootstrap.mjs [~/torstack]
//   bun ./bootstrap.mjs ~/torstack
//   bun ./bootstrap.mjs /usr/local/opt/torstack
import { chmodSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";

const scriptDir = import.meta.dir;
const targetArg = process.argv[2];
const ROOT = path.resolve(targetArg || scriptDir || path.join(os.homedir(), "torstack"));
const SRC = scriptDir;

console.log(`[torstack] src  = ${SRC}`);
console.log(`[torstack] target ROOT = ${ROOT}`);

// shebang portabel: Termux tidak punya /usr/bin/env, jadi script yang
// di-install di-rewrite ke bash lokal. Yang di-commit tetap
// `#!/usr/bin/env bash` (standar Linux). Jangan commit hasil rewrite!
const candidates = [
  process.env.PREFIX ? `${process.env.PREFIX}/bin/bash` : null,
  "/data/data/com.termux/files/usr/bin/bash",
  "/bin/bash",
  "/usr/bin/bash",
].filter(Boolean);
let BASH_BIN = null;
for (const c of candidates) {
  try {
    if (existsSync(c)) {
      BASH_BIN = c;
      break;
    }
  } catch {}
}
console.log(`[torstack] bash = ${BASH_BIN ?? "(keep #!/usr/bin/env bash)"}`);

// file sumber (relatif terhadap SRC) -> { target, mode }
const COPY = {
  "bin/torstack": { mode: 0o755 },
  "bin/tor-watchdog": { mode: 0o755 },
  "bin/tor-rotate": { mode: 0o755 },
  "bin/tor-events": { mode: 0o755 },
  "bin/tor-autorotate": { mode: 0o755 },
  "bin/tor-dashboard": { mode: 0o755 },
  "hooks/post-rotate.mjs": { mode: 0o644 },
  "hooks/pre-rotate.mjs": { mode: 0o644 },
  "lib/health.sh": { mode: 0o755 },
  "lib/events.sh": { mode: 0o755 },
  "lib/shim.sh": { mode: 0o755 },
  "lib/fallback.sh": { mode: 0o755 },
  "lib/torctl.py": { mode: 0o644 },
  "lib/dashboard-page.ts": { mode: 0o644 },
  "profiles/opencode.env": { mode: 0o644 },
  "profiles/crawl.env": { mode: 0o644 },
  "conf/torrc.example": { mode: 0o644 },
  "conf/privoxy.conf.example": { mode: 0o644 },
};

for (const d of ["bin", "conf", "lib", "profiles", "logs", "state", "hooks"]) {
  mkdirSync(path.join(ROOT, d), { recursive: true });
}
mkdirSync(path.join(ROOT, "state", "tor-data"), { recursive: true });

let wrote = 0;
for (const [rel, { mode }] of Object.entries(COPY)) {
  const src = path.join(SRC, rel);
  if (!existsSync(src)) {
    console.error(`[torstack][err] sumber hilang: ${src}`);
    process.exit(1);
  }
  const dst = path.join(ROOT, rel);
  mkdirSync(path.dirname(dst), { recursive: true });
  let out = readFileSync(src, "utf8");
  if (BASH_BIN && (rel.startsWith("bin/") || rel.startsWith("lib/"))) {
    out = out.replace("#!/usr/bin/env bash", `#!${BASH_BIN}`);
  }
  if (!out.endsWith("\n")) out += "\n";
  writeFileSync(dst, out);
  chmodSync(dst, mode);
  wrote++;
  console.log(`  copy ${rel} (${mode.toString(8)})`);
}

// render conf real dari *.example (portable: ROOT lokal + confdir autodetect)
function renderConf() {
  const torEx = readFileSync(path.join(ROOT, "conf", "torrc.example"), "utf8");
  writeFileSync(path.join(ROOT, "conf", "torrc"), torEx.replaceAll("{{ROOT}}", ROOT));

  let confdir = "";
  for (const d of [
    process.env.PREFIX ? `${process.env.PREFIX}/etc/privoxy` : null,
    "/etc/privoxy",
    "/usr/share/privoxy",
    "/usr/local/etc/privoxy",
  ].filter(Boolean)) {
    try {
      if (existsSync(d)) {
        confdir = d;
        break;
      }
    } catch {}
  }
  const privoxyEx = readFileSync(path.join(ROOT, "conf", "privoxy.conf.example"), "utf8");
  writeFileSync(
    path.join(ROOT, "conf", "privoxy.conf"),
    privoxyEx
      .replaceAll("{{ROOT}}", ROOT)
      .replaceAll("{{CONFDIR_LINE}}", confdir ? `confdir ${confdir}` : ""),
  );
  console.log(`  render conf/torrc + conf/privoxy.conf (confdir=${confdir || "-"})`);
}
renderConf();
wrote += 2;

// sentuh logs & state (jangan timpa last-rotate bila sudah ada)
for (const f of [
  "logs/tor.log",
  "logs/privoxy.log",
  "logs/crawl.log",
  "logs/watchdog.log",
  "logs/tor-events.log",
  "state/events.jsonl",
]) {
  const p = path.join(ROOT, f);
  if (!existsSync(p)) writeFileSync(p, "");
}
for (const f of ["state/tor.pid", "state/privoxy.pid"]) {
  const p = path.join(ROOT, f);
  if (!existsSync(p)) writeFileSync(p, "");
}
if (!existsSync(path.join(ROOT, "state", "last-rotate")))
  writeFileSync(path.join(ROOT, "state", "last-rotate"), "");

console.log(`[torstack] OK: ${wrote} files -> ${ROOT}`);
console.log(`next: ${ROOT}/bin/torstack start && ${ROOT}/bin/torstack status`);
