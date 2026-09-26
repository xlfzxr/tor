#!/usr/bin/env python3
"""lib/torctl.py — SATU file = Tor ControlPort client (cookie auth).

Dipakai bin/tor-rotate (newnym sekali jalan) dan bin/tor-events (listen
satu sesi; loop reconnect milik bash pemanggil). Tanpa dependensi
apa pun selain stdlib.

usage:
  python3 lib/torctl.py newnym --host 127.0.0.1 --port 9051 --cookie FILE
  python3 lib/torctl.py listen --host 127.0.0.1 --port 9051 --cookie FILE \\
      --out state/events.jsonl [--source tor-events]

env fallback (bila flag tak diisi): TOR_EV_HOST, TOR_EV_PORT,
TOR_EV_COOKIE, TOR_EV_OUT. Format event JSONL identik dengan emit()
lib/events.sh: {"ts":<ms>,"type":"...","source":"...","data":{...}}.
"""
import argparse
import binascii
import json
import os
import socket
import sys
import time

SOURCE_DEFAULT = "tor-events"


def die(msg, code=1):
    print(f"[torctl] {msg}", file=sys.stderr, flush=True)
    sys.exit(code)


def read_cookie(path):
    try:
        with open(path, "rb") as f:
            return binascii.hexlify(f.read()).decode()
    except OSError as e:
        die(f"no cookie ({path}): {e}")


class Control:
    """Satu koneksi control-port (line protocol CRLF)."""

    def __init__(self, host, port, hexcookie, timeout=10):
        self.hexcookie = hexcookie
        self.sock = socket.create_connection((host, port), timeout=timeout)
        self.f = self.sock.makefile("rw", newline="\r\n",
                                    encoding="utf-8", errors="replace")

    def cmd(self, line):
        self.f.write(line + "\r\n")
        self.f.flush()
        return self.f.readline().strip()

    def authenticate(self):
        r = self.cmd(f"AUTHENTICATE {self.hexcookie}")
        if not r.startswith("250"):
            die(f"AUTH gagal: {r[:200]}")
        return r

    def close(self):
        try:
            self.f.write("QUIT\r\n")
            self.f.flush()
        except OSError:
            pass
        try:
            self.sock.close()
        except OSError:
            pass


def cmd_newnym(args):
    hexcookie = read_cookie(args.cookie)
    ctl = Control(args.host, args.port, hexcookie)
    try:
        ctl.authenticate()
        r = ctl.cmd("SIGNAL NEWNYM")
        if not r.startswith("250"):
            die(f"NEWNYM gagal: {r[:200]}")
        print("OK NEWNYM", flush=True)
    finally:
        ctl.close()


def emit(out_path, source, typ, data):
    line = json.dumps({"ts": int(time.time() * 1000), "type": typ,
                       "source": source, "data": data},
                      separators=(",", ":"))
    try:
        with open(out_path, "a") as f:
            f.write(line + "\n")
    except OSError as e:
        print(f"[torctl] emit fail: {e}", flush=True)


def cmd_listen(args):
    hexcookie = read_cookie(args.cookie)
    try:
        ctl = Control(args.host, args.port, hexcookie, timeout=30)
    except OSError as e:
        print(f"[torctl] connect {args.host}:{args.port} gagal: {e} "
              f"— pemanggil retry", flush=True)
        sys.exit(1)
    try:
        ctl.authenticate()
    except SystemExit:
        emit(args.out, args.source, "tor-events.auth_fail", {})
        raise
    r = ctl.cmd("SETEVENTS CIRC STREAM BW STATUS_CLIENT NOTICE")
    if not r.startswith("250"):
        print(f"[torctl] SETEVENTS gagal: {r[:200]}", flush=True)
        sys.exit(1)
    print("[torctl] listening CIRC/STREAM/BW/STATUS_CLIENT/NOTICE",
          flush=True)
    emit(args.out, args.source, "tor-events.listening", {})

    last_bw_emit = 0
    pending_bw = None
    ctl.sock.settimeout(30)
    try:
        while True:
            try:
                line = ctl.f.readline()
            except socket.timeout:
                # flush BW tertunda biar dashboard tetap hidup
                now = int(time.time() * 1000)
                if pending_bw and now - last_bw_emit >= 5000:
                    emit(args.out, args.source, "net.bw", pending_bw)
                    last_bw_emit = now
                    pending_bw = None
                continue
            if not line:
                print("[torctl] EOF — pemanggil reconnect", flush=True)
                emit(args.out, args.source, "tor-events.disconnect", {})
                sys.exit(1)
            line = line.strip()
            if not line.startswith("650"):
                continue
            body = line[4:]
            if body.startswith("BW "):
                try:
                    _, down, up = body.split()[:3]
                    pending_bw = {"down": int(down), "up": int(up)}
                    now = int(time.time() * 1000)
                    if now - last_bw_emit >= 5000:
                        emit(args.out, args.source, "net.bw", pending_bw)
                        last_bw_emit = now
                        pending_bw = None
                except (ValueError, IndexError):
                    pass
            elif "BOOTSTRAP" in body:
                prog = -1
                for tok in body.split():
                    if tok.startswith("BOOTSTRAPPED="):
                        try:
                            prog = int(tok.split("=")[1])
                        except ValueError:
                            pass
                emit(args.out, args.source, "tor.bootstrap",
                     {"progress": prog, "raw": body[:200]})
            elif body.startswith("CIRC "):
                parts = body.split()
                status = parts[2] if len(parts) > 2 else "?"
                if status in ("BUILT", "FAILED", "CLOSED", "EXTENDED"):
                    emit(args.out, args.source, "tor.circ",
                         {"status": status, "raw": body[:220]})
            elif body.startswith("STREAM "):
                if ("SUCCEEDED" in body or "FAILED" in body
                        or "CLOSED" in body):
                    emit(args.out, args.source, "net.stream",
                         {"raw": body[:220]})
            elif body.startswith("NOTICE "):
                emit(args.out, args.source, "tor.notice",
                     {"raw": body[:250]})
            elif body.startswith("STATUS_CLIENT "):
                emit(args.out, args.source, "tor.status",
                     {"raw": body[:250]})
    except Exception as e:  # noqa: BLE001 — loop abadi, semua error = reconnect
        print(f"[torctl] loop error: {e} — pemanggil reconnect", flush=True)
        emit(args.out, args.source, "tor-events.error",
             {"msg": str(e)[:200]})
        sys.exit(1)
    finally:
        ctl.close()


def _common(p):
    # opsi koneksi boleh di depan ATAU belakang subcommand
    p.add_argument("--host", default=os.environ.get("TOR_EV_HOST",
                                                    "127.0.0.1"))
    p.add_argument("--port", type=int,
                   default=int(os.environ.get("TOR_EV_PORT", "9051")))
    p.add_argument("--cookie", default=os.environ.get("TOR_EV_COOKIE", ""))
    return p


def main(argv=None):
    p = argparse.ArgumentParser(prog="torctl.py",
                                description="Tor ControlPort client")
    _common(p)
    sub = p.add_subparsers(dest="cmd", required=True)
    pn = sub.add_parser("newnym", help="SIGNAL NEWNYM sekali jalan")
    _common(pn)
    pl = sub.add_parser("listen", help="SETEVENTS satu sesi (exit saat putus)")
    _common(pl)
    pl.add_argument("--out", default=os.environ.get("TOR_EV_OUT", ""),
                    help="file events.jsonl")
    pl.add_argument("--source", default=SOURCE_DEFAULT)
    args = p.parse_args(argv)
    if not args.cookie:
        die("cookie kosong (--cookie / TOR_EV_COOKIE)")
    if args.cmd == "newnym":
        cmd_newnym(args)
    elif args.cmd == "listen":
        if not args.out:
            die("out kosong (--out / TOR_EV_OUT)")
        cmd_listen(args)


if __name__ == "__main__":
    main()
