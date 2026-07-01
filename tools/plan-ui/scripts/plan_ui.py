#!/usr/bin/env python3
"""plan-ui CLI — the agent-facing entry point.

Commands:
  open <file> [--no-open]                 Serve a plan artifact and open the browser
  poll <file> [--agent-reply T] [--timeout-ms N]   Long-poll for human feedback
  end <file>                              End a session
  stop                                    Shut down the background server
  playbook                                Print the plan-authoring playbook
  serve                                   Run the HTTP server (used internally)

Agent-facing commands print JSON with a "next_step" field.
"""

from __future__ import annotations

import argparse
import sys

import common


def cmd_open(args):
    resp = common.api_post("/api/session", {"file": args.file})
    common.print_json(resp)
    if not args.no_open:
        url = resp.get("session", {}).get("url")
        if url:
            common.open_browser(url)


def cmd_poll(args):
    q = f"/api/poll?file={_esc(args.file)}"
    if args.agent_reply:
        q += f"&agent_reply={_esc(args.agent_reply)}"
    if args.timeout_ms:
        q += f"&timeout_ms={args.timeout_ms}"
    # No client timeout: the long-poll may wait indefinitely.
    common.print_json(common.api_get(q, timeout=None))


def cmd_end(args):
    common.print_json(common.api_post("/api/end", {"file": args.file}))


def cmd_stop(args):
    info = common.read_server_info()
    if not info:
        return common.print_json({"server": {"status": "not-running"}})
    try:
        common.api_post("/api/stop", None, base_url=info["url"])
        common.print_json({"server": {"status": "stopped"}})
    except Exception:
        common.print_json({"server": {"status": "not-running"}})


def cmd_playbook(args):
    print((common.plugin_root() / "references" / "playbook.md").read_text())


def cmd_serve(args):
    import server
    server.main()


def _esc(s: str) -> str:
    from urllib.parse import quote
    return quote(s, safe="")


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="plan-ui", description="Collaborate with agents on a plan in a rich browser UI.")
    sub = p.add_subparsers(dest="command", required=True)

    o = sub.add_parser("open", help="Serve a plan artifact and open it in the browser.")
    o.add_argument("file")
    o.add_argument("--no-open", action="store_true", help="Create the session without launching a browser.")
    o.set_defaults(func=cmd_open)

    pl = sub.add_parser("poll", help="Wait (long-poll) for human feedback.")
    pl.add_argument("file")
    pl.add_argument("--agent-reply", default="", help="Show this message to the human before waiting again.")
    pl.add_argument("--timeout-ms", type=int, default=0, help="Return after N ms if no feedback (0 = wait forever).")
    pl.set_defaults(func=cmd_poll)

    e = sub.add_parser("end", help="End a session.")
    e.add_argument("file")
    e.set_defaults(func=cmd_end)

    s = sub.add_parser("stop", help="Shut down the background server.")
    s.set_defaults(func=cmd_stop)

    pb = sub.add_parser("playbook", help="Print the plan-authoring playbook.")
    pb.set_defaults(func=cmd_playbook)

    sv = sub.add_parser("serve", help="Run the HTTP server (used internally).")
    sv.set_defaults(func=cmd_serve)

    return p


def main():
    args = build_parser().parse_args()
    try:
        args.func(args)
    except Exception as e:  # noqa: BLE001 — surface a clean error to the agent
        print(f"error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
