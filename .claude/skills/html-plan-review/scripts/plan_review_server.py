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
  server.json               {port, pid} of the running server
"""

import argparse
import json
import os
import socket
import sys
import threading
import time
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

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


def make_handler(state_dir: Path, server_ref):
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

        def _body(self):
            length = int(self.headers.get("Content-Length") or 0)
            if length <= 0 or length > 10_000_000:
                return None
            try:
                return json.loads(self.rfile.read(length).decode("utf-8"))
            except ValueError:
                return None

        def do_GET(self):
            path = self.path.split("?", 1)[0]
            if path in ("/", "/index.html"):
                try:
                    html = (ASSETS_DIR / "app.html").read_bytes()
                except OSError:
                    self._send(500, b"app.html missing", "text/plain")
                    return
                self._send(200, html, "text/html; charset=utf-8")
            elif path == "/api/health":
                self._json({"ok": True, "dir": str(state_dir)})
            elif path == "/api/state":
                submitted = sorted(
                    int(p.stem.split("-")[-1])
                    for p in state_dir.glob("feedback-round-*.json")
                    if p.stem.split("-")[-1].isdigit()
                )
                plan = read_json(state_dir / "plan.json")
                # The plan body lives in a raw HTML sidecar so agents can
                # author and revise it with targeted edits, free of JSON
                # string escaping. plan.json stays a tiny metadata file.
                body_file = state_dir / "plan.html"
                if plan is not None and body_file.exists():
                    try:
                        plan["bodyHtml"] = body_file.read_text(encoding="utf-8")
                    except OSError:
                        pass
                self._json({
                    "plan": plan,
                    "draft": read_json(state_dir / "annotations.json"),
                    "submittedRounds": submitted,
                })
            else:
                self._send(404, b"not found", "text/plain")

        def do_POST(self):
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


def existing_server_url(state_dir: Path):
    info = read_json(state_dir / "server.json")
    if not info or "port" not in info:
        return None
    url = f"http://127.0.0.1:{info['port']}"
    try:
        with urllib.request.urlopen(url + "/api/health", timeout=1.5) as resp:
            data = json.loads(resp.read().decode("utf-8"))
        if data.get("ok") and data.get("dir") == str(state_dir):
            return url
    except Exception:
        return None
    return None


def cmd_serve(args):
    state_dir = Path(args.dir).resolve()
    state_dir.mkdir(parents=True, exist_ok=True)

    url = existing_server_url(state_dir)
    if url:
        print(f"Plan review already running at {url}", flush=True)
        return 0

    port = args.port
    server = None
    for candidate in range(port, port + 50):
        try:
            server = ThreadingHTTPServer(("127.0.0.1", candidate), BaseHTTPRequestHandler)
            server.server_close()
            server_ref = []
            handler = make_handler(state_dir, server_ref)
            server = ThreadingHTTPServer(("127.0.0.1", candidate), handler)
            server_ref.append(server)
            port = candidate
            break
        except socket.error:
            server = None
    if server is None:
        print(f"No free port in {args.port}-{args.port + 49}", file=sys.stderr)
        return 1

    write_json(state_dir / "server.json", {"port": port, "pid": os.getpid()})
    print(f"Plan review server: http://127.0.0.1:{port}", flush=True)
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
    deadline = time.time() + args.timeout if args.timeout > 0 else None
    while True:
        if target.exists():
            data = read_json(target)
            if data is not None:
                print(json.dumps(data, indent=2, ensure_ascii=False))
                return 0
        if deadline and time.time() > deadline:
            print(f"Timed out waiting for {target.name}", file=sys.stderr)
            return 2
        time.sleep(1.5)


def cmd_stop(args):
    state_dir = Path(args.dir).resolve()
    url = existing_server_url(state_dir)
    if not url:
        print("No running server found.")
        return 0
    try:
        req = urllib.request.Request(url + "/api/shutdown", data=b"{}", method="POST")
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
