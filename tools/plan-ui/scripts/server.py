#!/usr/bin/env python3
"""plan-ui local server — standard library only.

Serves the review UI, injects the annotation SDK into the plan artifact, runs the
long-poll feedback channel for agents, streams live updates to the browser over
SSE, and (for the ExitPlanMode hook path) blocks on a human approve/deny decision.
"""

from __future__ import annotations

import json
import os
import queue
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlparse, parse_qs

import common
from mdrender import md_to_html

# --- status constants --------------------------------------------------------

STATUS_OPEN = "open"
STATUS_FEEDBACK = "feedback"
STATUS_ENDED = "ended"

GATE_PENDING = "pending"
GATE_PASSED = "passed"
GATE_FAILED = "failed"

PRESENCE_WAITING = "waiting"
PRESENCE_LISTENING = "listening"
PRESENCE_WORKING = "working"


def now_ts() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


class Store:
    """In-memory sessions + live coordination, persisted to ~/.plan-ui/state.json."""

    def __init__(self):
        self._lock = threading.Lock()
        self._cond = threading.Condition(self._lock)
        self.sessions: dict[str, dict] = {}
        self.sse: dict[str, list[queue.Queue]] = {}
        self._load()

    # persistence -------------------------------------------------------------

    def _load(self):
        try:
            self.sessions = json.loads(common.state_path().read_text())
        except (OSError, ValueError):
            self.sessions = {}

    def _persist(self):
        try:
            tmp = common.state_path().with_suffix(".tmp")
            tmp.write_text(json.dumps(self.sessions, indent=2))
            tmp.replace(common.state_path())
        except OSError:
            pass

    # sessions ----------------------------------------------------------------

    def upsert(self, key: str, file: str, url: str, mode: str, markdown: str | None) -> dict:
        with self._cond:
            sess = self.sessions.get(key)
            if sess is None:
                sess = {
                    "key": key, "file": file, "url": url, "mode": mode,
                    "status": STATUS_OPEN, "gate": GATE_PENDING, "presence": PRESENCE_WAITING,
                    "pending_prompts": [], "layout_warnings": [], "chat": [],
                    "markdown": markdown, "decision": None,
                }
                self.sessions[key] = sess
            else:
                sess["url"] = url
                sess["mode"] = mode
                sess["status"] = STATUS_OPEN
                sess["gate"] = GATE_PENDING
                sess["decision"] = None
                if markdown is not None:
                    sess["markdown"] = markdown
            sess["updated_at"] = now_ts()
            self._persist()
            self._cond.notify_all()
            return dict(sess)

    def get(self, key: str) -> dict | None:
        with self._lock:
            s = self.sessions.get(key)
            return dict(s) if s else None

    def end(self, key: str) -> dict | None:
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return None
            sess["status"] = STATUS_ENDED
            sess["presence"] = PRESENCE_WAITING
            sess["updated_at"] = now_ts()
            self._persist()
            self._cond.notify_all()
            return dict(sess)

    # feedback ----------------------------------------------------------------

    def queue_prompts(self, key: str, prompts: list[dict]):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return
            for p in prompts:
                p.setdefault("ts", now_ts())
                sess["pending_prompts"].append(p)
                sess["chat"].append({
                    "role": "human", "content": p.get("text", ""),
                    "action": p.get("action", ""), "target": p.get("target"), "ts": p["ts"],
                })
            if sess["status"] != STATUS_ENDED:
                sess["status"] = STATUS_FEEDBACK
            sess["updated_at"] = now_ts()
            self._persist()
            self._cond.notify_all()
        # Mirror each annotation to the browser so the shell can display it and,
        # in plan mode, fold it into the review decision.
        for p in prompts:
            self.broadcast(key, "human", json.dumps(p))

    def has_feedback(self, key: str) -> bool:
        sess = self.sessions.get(key)
        if not sess:
            return False
        return bool(sess["pending_prompts"] or sess["layout_warnings"] or sess["status"] == STATUS_ENDED)

    def take_feedback(self, key: str):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return [], []
            prompts = sess["pending_prompts"]
            warnings = sess["layout_warnings"]
            sess["pending_prompts"] = []
            sess["layout_warnings"] = []
            if sess["status"] != STATUS_ENDED:
                sess["status"] = STATUS_OPEN
            sess["updated_at"] = now_ts()
            self._persist()
            return prompts, warnings

    def wait_feedback(self, key: str, timeout: float | None) -> bool:
        """Block until feedback is available or timeout; returns True if available."""
        with self._cond:
            if self.has_feedback(key):
                return True
            self._cond.wait(timeout=timeout)
            return self.has_feedback(key)

    def add_agent_reply(self, key: str, text: str):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return
            sess["chat"].append({"role": "agent", "content": text, "ts": now_ts()})
            sess["updated_at"] = now_ts()
            self._persist()
        self.broadcast(key, "agent-reply", text)

    def set_presence(self, key: str, presence: str):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return
            sess["presence"] = presence
            sess["updated_at"] = now_ts()
        self.broadcast(key, "presence", presence)

    # gate --------------------------------------------------------------------

    def report_gate(self, key: str, warnings: list[dict]):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return
            if not warnings:
                sess["gate"] = GATE_PASSED
                sess["updated_at"] = now_ts()
                self._persist()
                notify = ("gate", GATE_PASSED)
                wake = False
            else:
                sess["gate"] = GATE_FAILED
                sess["layout_warnings"] = warnings
                if sess["status"] != STATUS_ENDED:
                    sess["status"] = STATUS_FEEDBACK
                sess["updated_at"] = now_ts()
                self._persist()
                notify = ("gate", GATE_FAILED)
                wake = True
                self._cond.notify_all()
        self.broadcast(key, *notify)

    # hook decision -----------------------------------------------------------

    def set_decision(self, key: str, decision: str, feedback: str):
        with self._cond:
            sess = self.sessions.get(key)
            if not sess:
                return
            sess["decision"] = {"decision": decision, "feedback": feedback, "ts": now_ts()}
            sess["updated_at"] = now_ts()
            self._persist()
            self._cond.notify_all()
        self.broadcast(key, "decision", decision)

    def wait_decision(self, key: str, timeout: float | None) -> dict | None:
        with self._cond:
            sess = self.sessions.get(key)
            if sess and sess.get("decision"):
                return sess["decision"]
            self._cond.wait(timeout=timeout)
            sess = self.sessions.get(key)
            return sess.get("decision") if sess else None

    # SSE ---------------------------------------------------------------------

    def subscribe(self, key: str) -> queue.Queue:
        q: queue.Queue = queue.Queue(maxsize=32)
        with self._lock:
            self.sse.setdefault(key, []).append(q)
        return q

    def unsubscribe(self, key: str, q: queue.Queue):
        with self._lock:
            if key in self.sse and q in self.sse[key]:
                self.sse[key].remove(q)

    def broadcast(self, key: str, event: str, data: str):
        with self._lock:
            clients = list(self.sse.get(key, []))
        for q in clients:
            try:
                q.put_nowait((event, data))
            except queue.Full:
                pass

    def any_open(self) -> bool:
        with self._lock:
            return any(s["status"] != STATUS_ENDED for s in self.sessions.values())


STORE = Store()
WATCHERS: dict[str, threading.Event] = {}
_watch_lock = threading.Lock()

# Idle tracking: the server exits after IDLE_AFTER seconds with no API activity,
# no connected browsers, and no agent poll blocked waiting.
IDLE_AFTER = 30 * 60
_activity_lock = threading.Lock()
_last_active = time.time()
_active_polls = 0


def touch():
    global _last_active
    with _activity_lock:
        _last_active = time.time()


def poll_begin():
    global _active_polls
    with _activity_lock:
        _active_polls += 1


def poll_end():
    global _active_polls, _last_active
    with _activity_lock:
        _active_polls -= 1
        _last_active = time.time()


def is_idle() -> bool:
    with _activity_lock:
        if _active_polls > 0:
            return False
        if time.time() - _last_active < IDLE_AFTER:
            return False
    with STORE._lock:
        if any(STORE.sse.values()):
            return False
    return True


def start_watch(key: str, canonical: str):
    """Broadcast a reload when the artifact file changes (mtime poll, stdlib only)."""
    with _watch_lock:
        if key in WATCHERS:
            return
        stop = threading.Event()
        WATCHERS[key] = stop

    def loop():
        last = os.path.getmtime(canonical) if os.path.exists(canonical) else 0
        while not stop.wait(0.5):
            try:
                m = os.path.getmtime(canonical)
            except OSError:
                continue
            if m > last:
                last = m
                STORE.broadcast(key, "reload", "")

    threading.Thread(target=loop, daemon=True).start()


def stop_watch(key: str):
    with _watch_lock:
        stop = WATCHERS.pop(key, None)
    if stop:
        stop.set()


# --- artifact rendering ------------------------------------------------------

INJECT = """
<!-- plan-ui: injected runtime (local, no CDN) -->
<script>window.__PLAN_UI__ = {{ key: "{key}", mode: "{mode}" }};</script>
<script src="/assets/tailwind.js"></script>
<link rel="stylesheet" href="/assets/daisyui.css">
<link rel="stylesheet" href="/assets/chrome.css">
<script defer src="/assets/sdk.js"></script>
"""


def transform_artifact(sess: dict) -> bytes:
    """Inject the plan-ui runtime into an agent-authored HTML file."""
    with open(sess["file"], "r", encoding="utf-8") as f:
        html = f.read()
    inject = INJECT.format(key=sess["key"], mode="artifact")
    low = html.lower()
    i = low.find("<head")
    if i >= 0:
        end = low.find(">", i)
        if end >= 0:
            return (html[: end + 1] + inject + html[end + 1:]).encode()
    i = low.find("</head>")
    if i >= 0:
        return (html[:i] + inject + html[i:]).encode()
    return ('<!doctype html><html><head><meta charset="utf-8">' + inject +
            "</head><body>" + html + "</body></html>").encode()


PLAN_PROSE = """
<style>
  body { margin: 0; background: #fff; }
  .plan-doc { max-width: 820px; margin: 0 auto; padding: 48px 32px 120px;
    font: 16px/1.65 ui-sans-serif, system-ui, -apple-system, sans-serif; color: #1c2230; }
  .plan-doc h1 { font-size: 1.9em; margin: 0 0 .5em; }
  .plan-doc h2 { font-size: 1.4em; margin: 1.6em 0 .5em; padding-bottom: .2em; border-bottom: 1px solid #e6e8ec; }
  .plan-doc h3 { font-size: 1.15em; margin: 1.3em 0 .4em; }
  .plan-doc p, .plan-doc li { margin: .5em 0; }
  .plan-doc ul, .plan-doc ol { padding-left: 1.4em; }
  .plan-doc code { background: #f2f3f5; padding: .12em .35em; border-radius: 4px; font-size: .9em; }
  .plan-doc pre { background: #0e1017; color: #e8ecf2; padding: 14px 16px; border-radius: 8px; overflow-x: auto; }
  .plan-doc pre code { background: none; padding: 0; color: inherit; }
  .plan-doc a { color: #c15412; }
  .plan-doc blockquote { margin: .6em 0; padding: .2em 1em; border-left: 3px solid #e0771b; color: #55606f; }
  .plan-doc hr { border: none; border-top: 1px solid #e6e8ec; margin: 1.6em 0; }
</style>
"""


def render_plan(sess: dict) -> bytes:
    """Render a plan-mode session (markdown) to a reviewable HTML document."""
    inject = INJECT.format(key=sess["key"], mode="plan")
    body = md_to_html(sess.get("markdown") or "")
    return (
        '<!doctype html><html><head><meta charset="utf-8">'
        '<meta name="viewport" content="width=device-width, initial-scale=1">'
        + inject + PLAN_PROSE +
        '</head><body><article class="plan-doc">' + body + "</article></body></html>"
    ).encode()


# --- HTTP handler ------------------------------------------------------------

class Handler(BaseHTTPRequestHandler):
    server_version = "plan-ui"

    def log_message(self, *args):
        pass  # quiet

    # helpers
    def _json(self, code: int, obj):
        body = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _err(self, code: int, msg: str):
        self._json(code, {"error": msg})

    def _body(self) -> dict:
        length = int(self.headers.get("Content-Length", 0))
        if not length:
            return {}
        try:
            return json.loads(self.rfile.read(length).decode())
        except ValueError:
            return {}

    def _send_bytes(self, data: bytes, ctype: str):
        self.send_response(200)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    # routing
    def do_GET(self):
        u = urlparse(self.path)
        path, q = u.path, parse_qs(u.query)
        if path == "/healthz":
            return self._json(200, {"ok": True})
        if path == "/api/poll":
            return self.handle_poll(q)
        parts = path.strip("/").split("/")
        if len(parts) == 3 and parts[0] == "api" and parts[2] == "state":
            return self.handle_state(parts[1])
        if len(parts) == 3 and parts[0] == "api" and parts[2] == "await-decision":
            return self.handle_await_decision(parts[1], q)
        if len(parts) == 2 and parts[0] == "s":
            return self.handle_shell(parts[1])
        if len(parts) == 2 and parts[0] == "artifact":
            return self.handle_artifact(parts[1])
        if len(parts) == 2 and parts[0] == "events":
            return self.handle_events(parts[1])
        if len(parts) == 2 and parts[0] == "assets":
            return self.handle_asset(parts[1])
        self._err(404, "not found")

    def do_POST(self):
        u = urlparse(self.path)
        path = u.path
        if path == "/api/session":
            return self.handle_session()
        if path == "/api/end":
            return self.handle_end()
        if path == "/api/stop":
            return self.handle_stop()
        parts = path.strip("/").split("/")
        if len(parts) == 3 and parts[0] == "api":
            key, action = parts[1], parts[2]
            if action == "agent-reply":
                return self.handle_agent_reply(key)
            if action == "feedback":
                return self.handle_feedback(key)
            if action == "gate":
                return self.handle_gate(key)
            if action == "decision":
                return self.handle_decision(key)
        self._err(404, "not found")

    # --- agent API ---
    def handle_session(self):
        touch()
        b = self._body()
        file = b.get("file")
        markdown = b.get("markdown")
        mode = b.get("mode", "artifact")
        if mode == "plan":
            label = b.get("label") or "plan-review"
            key = common.key_for_label(label)
            canonical = label
        else:
            if not file:
                return self._err(400, "missing file")
            canonical = common.canonical_file(file)
            if not os.path.exists(canonical):
                return self._err(400, "file does not exist: " + canonical)
            key = common.key_for_file(canonical)
        url = f"{BASE_URL}/s/{key}"
        sess = STORE.upsert(key, canonical, url, mode, markdown)
        if mode == "artifact":
            start_watch(key, canonical)
        nxt = (f"Open {url} in a browser, then run `plan-ui poll {file}` to wait for "
               "feedback. The poll blocks silently until the human responds — never kill it.")
        self._json(200, {"session": _view(sess), "next_step": nxt})

    def handle_end(self):
        b = self._body()
        file = b.get("file")
        key = b.get("key")
        if not key and file:
            key = common.key_for_file(common.canonical_file(file))
        sess = STORE.end(key) if key else None
        if not sess:
            return self._err(404, "no session")
        stop_watch(key)
        STORE.broadcast(key, "ended", "")
        self._json(200, {"session": _view(sess), "next_step": "Session ended."})

    def handle_poll(self, q):
        touch()
        file = (q.get("file") or [""])[0]
        if not file:
            return self._err(400, "missing file")
        key = common.key_for_file(common.canonical_file(file))
        if not STORE.get(key):
            return self._err(404, "no session for file; run `plan-ui open` first")
        reply = (q.get("agent_reply") or [""])[0]
        if reply:
            STORE.add_agent_reply(key, reply)
        timeout_ms = int((q.get("timeout_ms") or ["0"])[0] or 0)
        STORE.set_presence(key, PRESENCE_LISTENING)

        poll_begin()
        try:
            deadline = time.time() + timeout_ms / 1000 if timeout_ms > 0 else None
            while True:
                remaining = None if deadline is None else max(0, deadline - time.time())
                got = STORE.wait_feedback(key, remaining if remaining is None else min(remaining, 30))
                if got:
                    STORE.set_presence(key, PRESENCE_WORKING)
                    prompts, warnings = STORE.take_feedback(key)
                    sess = STORE.get(key)
                    return self._json(200, {
                        "prompts": prompts, "layout_warnings": warnings,
                        "session": _view(sess), "next_step": _poll_next(sess, file),
                    })
                if deadline is not None and time.time() >= deadline:
                    STORE.set_presence(key, PRESENCE_WAITING)
                    return self._json(200, {
                        "prompts": [], "layout_warnings": [], "timed_out": True,
                        "next_step": "No feedback yet. Poll again to keep waiting.",
                    })
        finally:
            poll_end()

    def handle_agent_reply(self, key):
        STORE.add_agent_reply(key, self._body().get("text", ""))
        self._json(200, {"ok": True})

    def handle_feedback(self, key):
        touch()
        prompts = self._body().get("prompts", [])
        STORE.queue_prompts(key, prompts)
        self._json(200, {"ok": True, "queued": len(prompts)})

    def handle_gate(self, key):
        STORE.report_gate(key, self._body().get("warnings", []))
        self._json(200, {"ok": True})

    def handle_state(self, key):
        sess = STORE.get(key)
        if not sess:
            return self._err(404, "unknown session")
        self._json(200, _full_view(sess))

    # --- hook decision ---
    def handle_decision(self, key):
        touch()
        b = self._body()
        STORE.set_decision(key, b.get("decision", "deny"), b.get("feedback", ""))
        self._json(200, {"ok": True})

    def handle_await_decision(self, key, q):
        touch()
        timeout_ms = int((q.get("timeout_ms") or ["0"])[0] or 0)
        poll_begin()
        try:
            deadline = time.time() + timeout_ms / 1000 if timeout_ms > 0 else None
            while True:
                remaining = None if deadline is None else max(0, deadline - time.time())
                d = STORE.wait_decision(key, remaining if remaining is None else min(remaining, 30))
                if d:
                    return self._json(200, d)
                if deadline is not None and time.time() >= deadline:
                    return self._json(200, {"decision": "timeout", "feedback": ""})
        finally:
            poll_end()

    def handle_stop(self):
        self._json(200, {"server": {"status": "stopping"}})
        threading.Thread(target=self.server.shutdown, daemon=True).start()

    # --- browser surfaces ---
    def handle_shell(self, key):
        sess = STORE.get(key)
        if not sess:
            return self._err(404, "unknown session")
        shell = (common.web_dir() / "shell.html").read_text().replace("__PLAN_UI_KEY__", key)
        self._send_bytes(shell.encode(), "text/html; charset=utf-8")

    def handle_artifact(self, key):
        sess = STORE.get(key)
        if not sess:
            return self._err(404, "unknown session")
        try:
            data = render_plan(sess) if sess["mode"] == "plan" else transform_artifact(sess)
        except OSError as e:
            return self._err(500, f"cannot read artifact: {e}")
        self._send_bytes(data, "text/html; charset=utf-8")

    def handle_asset(self, name):
        p = common.web_dir() / name
        if not p.is_file() or ".." in name:
            return self._err(404, "not found")
        ctype = {
            ".js": "application/javascript; charset=utf-8",
            ".css": "text/css; charset=utf-8",
            ".html": "text/html; charset=utf-8",
        }.get(p.suffix, "application/octet-stream")
        self._send_bytes(p.read_bytes(), ctype)

    def handle_events(self, key):
        if not STORE.get(key):
            return self._err(404, "unknown session")
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.end_headers()
        q = STORE.subscribe(key)
        try:
            sess = STORE.get(key)
            self._sse("presence", sess["presence"])
            self._sse("gate", sess["gate"])
            while True:
                try:
                    event, data = q.get(timeout=20)
                    self._sse(event, data)
                except queue.Empty:
                    self.wfile.write(b": ping\n\n")
                    self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError):
            pass
        finally:
            STORE.unsubscribe(key, q)

    def _sse(self, event: str, data: str):
        out = f"event: {event}\n"
        for line in str(data).split("\n"):
            out += f"data: {line}\n"
        out += "\n"
        self.wfile.write(out.encode())
        self.wfile.flush()


def _view(sess: dict) -> dict:
    return {k: sess[k] for k in ("key", "file", "url", "status", "gate", "presence", "mode")}


def _full_view(sess: dict) -> dict:
    v = _view(sess)
    v["chat"] = sess["chat"]
    return v


def _poll_next(sess: dict, file: str) -> str:
    if sess and sess["status"] == STATUS_ENDED:
        return "The session was ended by the human. Stop polling."
    return (f"Apply the feedback to the plan file, then run `plan-ui poll {file} "
            "--agent-reply \"<what you changed>\"` to show your response and wait again.")


BASE_URL = ""


def main():
    global BASE_URL
    httpd = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    port = httpd.server_address[1]
    BASE_URL = f"http://127.0.0.1:{port}"
    common.write_server_info({"pid": os.getpid(), "port": port, "url": BASE_URL})

    def idle_loop():
        while True:
            time.sleep(60)
            if is_idle():
                httpd.shutdown()
                return

    threading.Thread(target=idle_loop, daemon=True).start()
    try:
        httpd.serve_forever()
    finally:
        common.remove_server_info()


if __name__ == "__main__":
    main()
