#!/usr/bin/env python3
"""Local plan-review server for the html-plan-review skill.

Serves an interactive HTML review page for an implementation plan and
collects the user's annotations (comments, per-section verdicts, overall
approve / request-changes). Python 3.8+ stdlib only — no dependencies.

Commands:

  serve --dir DIR [--port PORT]
      Start the review server for state directory DIR. Blocks, so run it
      in the background. Prints the URL to stdout. If a healthy server is
      already running for DIR, prints its URL and exits 0 immediately.

  wait --dir DIR --round N [--timeout SECS]
      Block until the user submits feedback for round N, then print the
      feedback JSON to stdout and exit 0. Exits 2 on timeout (0 = wait
      forever, the default).

  stop --dir DIR
      Ask a running server for DIR to shut down.

State directory layout:
  plan.json                 the current plan (written by the agent each round)
  annotations.json          the user's in-progress draft (auto-saved by the UI)
  feedback-round-N.json     submitted feedback for round N (written on submit)
  server.json               {port, pid, token} of the running server
"""

import argparse
import hmac
import json
import os
import secrets
import socket
import sys
import threading
import time
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

# Feedback JSON is written with ensure_ascii=False and printed back to the
# agent; on Windows a non-console stdout defaults to the legacy code page
# (e.g. cp1252), which dies on characters like "→". Pin UTF-8 explicitly.
for _stream in (sys.stdout, sys.stderr):
    if hasattr(_stream, "reconfigure"):
        _stream.reconfigure(encoding="utf-8", errors="replace")

ASSETS_DIR = Path(__file__).resolve().parent.parent / "assets"


def read_json(path, default=None):
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    except (OSError, ValueError):
        return default


def write_json(path, obj):
    tmp = str(path) + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(obj, f, indent=2, ensure_ascii=False)
    os.replace(tmp, path)


def make_handler(state_dir: Path, server_ref, token: str):
    class Handler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def _send(self, code, body: bytes, ctype="application/json; charset=utf-8"):
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(body)

        def _json(self, obj, code=200):
            self._send(code, json.dumps(obj).encode("utf-8"))

        def _host_ok(self):
            # Reject DNS-rebinding: a hostile domain resolving to 127.0.0.1
            # would arrive with its own Host header.
            host = (self.headers.get("Host") or "").rsplit(":", 1)[0]
            return host in ("127.0.0.1", "localhost", "[::1]")

        def _authed(self):
            # Plan content and feedback endpoints require the per-server
            # token (in the page URL as ?t=…, echoed back as a header),
            # so other local processes / hostile web pages can't read the
            # plan or forge a verdict.
            supplied = (
                urllib.parse.parse_qs(
                    urllib.parse.urlparse(self.path).query
                ).get("t", [None])[0]
                or self.headers.get("X-Plan-Token")
                or ""
            )
            return hmac.compare_digest(supplied, token)

        def _body(self):
            length = int(self.headers.get("Content-Length") or 0)
            if length <= 0 or length > 10_000_000:
                return None
            try:
                return json.loads(self.rfile.read(length).decode("utf-8"))
            except ValueError:
                return None

        def do_GET(self):
            if not self._host_ok():
                self._send(403, b"forbidden", "text/plain")
                return
            path = self.path.split("?", 1)[0]
            if path in ("/", "/index.html"):
                try:
                    html = (ASSETS_DIR / "app.html").read_bytes()
                except OSError:
                    self._send(500, b"app.html missing", "text/plain")
                    return
                self._send(200, html, "text/html; charset=utf-8")
            elif path == "/favicon.ico":
                self._send(204, b"", "image/x-icon")
            elif path == "/api/health":
                self._json({"ok": True, "dir": str(state_dir)})
            elif path == "/api/state":
                if not self._authed():
                    self._json({"error": "missing or bad token"}, 403)
                    return
                plan_path = state_dir / "plan.json"
                try:
                    plan_mtime = plan_path.stat().st_mtime
                except OSError:
                    plan_mtime = 0.0
                # Feedback older than the current plan.json is left over
                # from an earlier review in a reused state dir — ignore it.
                submitted = []
                verdicts = {}
                for p in state_dir.glob("feedback-round-*.json"):
                    stem = p.stem.split("-")[-1]
                    if not stem.isdigit():
                        continue
                    try:
                        if p.stat().st_mtime < plan_mtime:
                            continue
                    except OSError:
                        continue
                    submitted.append(int(stem))
                    fb = read_json(p)
                    if fb and fb.get("verdict"):
                        verdicts[stem] = fb["verdict"]
                plan = read_json(plan_path)
                # The plan body lives in a raw HTML sidecar so agents can
                # author and revise it with targeted edits, free of JSON
                # string escaping. plan.json stays a tiny metadata file.
                body_file = state_dir / "plan.html"
                if plan is not None and body_file.exists():
                    try:
                        plan["bodyHtml"] = body_file.read_text(encoding="utf-8")
                    except OSError:
                        pass
                if plan is not None and not plan.get("updatedAt") and plan_mtime:
                    plan["updatedAt"] = time.strftime(
                        "%Y-%m-%d %H:%M", time.localtime(plan_mtime))
                self._json({
                    "plan": plan,
                    "draft": read_json(state_dir / "annotations.json"),
                    "submittedRounds": sorted(submitted),
                    "verdicts": verdicts,
                })
            else:
                self._send(404, b"not found", "text/plain")

        def do_POST(self):
            if not self._host_ok():
                self._send(403, b"forbidden", "text/plain")
                return
            if not self._authed():
                self._json({"error": "missing or bad token"}, 403)
                return
            path = self.path.split("?", 1)[0]
            if path == "/api/draft":
                body = self._body()
                if body is None:
                    self._json({"error": "bad body"}, 400)
                    return
                write_json(state_dir / "annotations.json", body)
                self._json({"ok": True})
            elif path == "/api/submit":
                body = self._body()
                if body is None or not isinstance(body.get("round"), int):
                    self._json({"error": "bad body; integer 'round' required"}, 400)
                    return
                rnd = body["round"]
                body["submittedAt"] = time.strftime("%Y-%m-%dT%H:%M:%S%z")
                write_json(state_dir / f"feedback-round-{rnd}.json", body)
                # Draft is consumed by submission.
                write_json(state_dir / "annotations.json", {})
                self._json({"ok": True, "round": rnd})
            elif path == "/api/shutdown":
                self._json({"ok": True})
                threading.Thread(target=server_ref[0].shutdown, daemon=True).start()
            else:
                self._send(404, b"not found", "text/plain")

        def log_message(self, fmt, *args):
            pass  # keep background-task output quiet

    return Handler


def existing_server(state_dir: Path):
    """Return (base_url, token) of a healthy server for this dir, else None."""
    info = read_json(state_dir / "server.json")
    if not info or "port" not in info:
        return None
    base = f"http://127.0.0.1:{info['port']}"
    try:
        with urllib.request.urlopen(base + "/api/health", timeout=1.5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        if data.get("ok") and data.get("dir") == str(state_dir):
            return base, info.get("token", "")
    except Exception:
        return None
    return None


def review_url(base: str, token: str):
    return f"{base}/?t={token}" if token else base


def cmd_serve(args):
    state_dir = Path(args.dir).resolve()
    state_dir.mkdir(parents=True, exist_ok=True)

    existing = existing_server(state_dir)
    if existing:
        print(f"Plan review already running at {review_url(*existing)}", flush=True)
        return 0

    token = secrets.token_urlsafe(16)
    port = args.port
    server = None
    for candidate in range(port, port + 50):
        try:
            server = ThreadingHTTPServer(("127.0.0.1", candidate), BaseHTTPRequestHandler)
            server.server_close()
            server_ref = []
            handler = make_handler(state_dir, server_ref, token)
            server = ThreadingHTTPServer(("127.0.0.1", candidate), handler)
            server_ref.append(server)
            port = candidate
            break
        except socket.error:
            server = None
    if server is None:
        print(f"No free port in {args.port}-{args.port + 49}", file=sys.stderr)
        return 1

    write_json(state_dir / "server.json",
               {"port": port, "pid": os.getpid(), "token": token})
    print(f"Plan review server: {review_url(f'http://127.0.0.1:{port}', token)}",
          flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        try:
            (state_dir / "server.json").unlink()
        except OSError:
            pass
    return 0


def cmd_wait(args):
    state_dir = Path(args.dir).resolve()
    target = state_dir / f"feedback-round-{args.round}.json"
    plan_path = state_dir / "plan.json"
    deadline = time.time() + args.timeout if args.timeout > 0 else None
    while True:
        if target.exists():
            # Ignore feedback predating the current plan.json — leftovers
            # from an earlier review in a reused state dir. A genuine
            # submission rewrites the file, refreshing its mtime.
            stale = False
            try:
                stale = target.stat().st_mtime < plan_path.stat().st_mtime
            except OSError:
                pass
            data = None if stale else read_json(target)
            if data is not None:
                print(json.dumps(data, indent=2, ensure_ascii=False))
                return 0
        if deadline and time.time() > deadline:
            print(f"Timed out waiting for {target.name}", file=sys.stderr)
            return 2
        time.sleep(1.5)


def cmd_stop(args):
    state_dir = Path(args.dir).resolve()
    existing = existing_server(state_dir)
    if not existing:
        print("No running server found.")
        return 0
    base, token = existing
    try:
        req = urllib.request.Request(base + "/api/shutdown", data=b"{}",
                                     method="POST",
                                     headers={"X-Plan-Token": token})
        urllib.request.urlopen(req, timeout=2)
        print("Server stopped.")
    except Exception as exc:
        print(f"Shutdown request failed: {exc}", file=sys.stderr)
        return 1
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("serve", help="start the review server (blocks)")
    p.add_argument("--dir", required=True)
    p.add_argument("--port", type=int, default=4173)
    p.set_defaults(fn=cmd_serve)

    p = sub.add_parser("wait", help="block until feedback for a round is submitted")
    p.add_argument("--dir", required=True)
    p.add_argument("--round", type=int, required=True)
    p.add_argument("--timeout", type=float, default=0, help="seconds; 0 = forever")
    p.set_defaults(fn=cmd_wait)

    p = sub.add_parser("stop", help="stop a running server")
    p.add_argument("--dir", required=True)
    p.set_defaults(fn=cmd_stop)

    args = parser.parse_args()
    sys.exit(args.fn(args))


if __name__ == "__main__":
    main()
