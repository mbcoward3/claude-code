"""Shared helpers for the plan-ui plugin: paths, session keys, server discovery,
and a tiny HTTP client. Standard library only."""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import time
import urllib.request
import urllib.error
from pathlib import Path


def base_dir() -> Path:
    """~/.plan-ui, created if missing."""
    d = Path.home() / ".plan-ui"
    d.mkdir(parents=True, exist_ok=True)
    return d


def state_path() -> Path:
    return base_dir() / "state.json"


def server_info_path() -> Path:
    return base_dir() / "server.json"


def plugin_root() -> Path:
    """The plugin directory (parent of scripts/)."""
    return Path(__file__).resolve().parent.parent


def web_dir() -> Path:
    return plugin_root() / "web"


def canonical_file(path: str) -> str:
    """Absolute, symlink-resolved path — the stable identity of a session."""
    p = Path(path).expanduser()
    try:
        return str(p.resolve())
    except OSError:
        return str(p.absolute())


def key_for_file(canonical: str) -> str:
    return hashlib.sha256(canonical.encode()).hexdigest()[:16]


# --- server info -------------------------------------------------------------

def read_server_info() -> dict | None:
    try:
        return json.loads(server_info_path().read_text())
    except (OSError, ValueError):
        return None


def write_server_info(info: dict) -> None:
    server_info_path().write_text(json.dumps(info, indent=2))


def remove_server_info() -> None:
    try:
        server_info_path().unlink()
    except OSError:
        pass


def healthy(base_url: str) -> bool:
    try:
        with urllib.request.urlopen(base_url + "/healthz", timeout=0.5) as r:
            return r.status == 200
    except Exception:
        return False


def ensure_server() -> str:
    """Return the base URL of a healthy server, spawning one if needed."""
    info = read_server_info()
    if info and healthy(info["url"]):
        return info["url"]

    server_py = str(Path(__file__).resolve().parent / "server.py")
    # Detached so the server outlives this CLI invocation.
    subprocess.Popen(
        [sys.executable, server_py],
        stdin=subprocess.DEVNULL,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
        start_new_session=True,
    )
    deadline = time.time() + 5.0
    while time.time() < deadline:
        info = read_server_info()
        if info and healthy(info["url"]):
            return info["url"]
        time.sleep(0.05)
    raise RuntimeError("plan-ui server did not become healthy in time")


# --- HTTP client -------------------------------------------------------------

def api_post(path: str, body: dict | None, base_url: str | None = None) -> dict:
    base = base_url or ensure_server()
    data = json.dumps(body or {}).encode()
    req = urllib.request.Request(
        base + path, data=data, headers={"Content-Type": "application/json"}, method="POST"
    )
    return _do(req)


def api_get(path: str, base_url: str | None = None, timeout: float | None = None) -> dict:
    base = base_url or ensure_server()
    req = urllib.request.Request(base + path, method="GET")
    return _do(req, timeout=timeout)


def _do(req: urllib.request.Request, timeout: float | None = None) -> dict:
    try:
        # timeout=None means wait indefinitely (used for long-poll).
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.loads(r.read().decode())
    except urllib.error.HTTPError as e:
        try:
            payload = json.loads(e.read().decode())
        except Exception:
            payload = {"error": f"HTTP {e.code}"}
        raise RuntimeError(payload.get("error", f"HTTP {e.code}"))


def open_browser(url: str) -> None:
    import webbrowser
    try:
        webbrowser.open(url)
    except Exception:
        pass


def print_json(v) -> None:
    print(json.dumps(v, indent=2))
