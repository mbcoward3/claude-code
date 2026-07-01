#!/usr/bin/env python3
"""ExitPlanMode review hook (plannotator-style).

Wired as a Claude Code PermissionRequest hook matching ExitPlanMode. When Claude
finishes planning, this reads the plan from stdin, opens the plan-ui review UI,
and blocks until the human approves or sends feedback — then emits the permission
decision on stdout.

stdout is reserved for the decision JSON only; all status goes to stderr.
"""

from __future__ import annotations

import json
import sys

import common


def emit(behavior: str, message: str = ""):
    """Print the PermissionRequest decision and exit."""
    decision = {"behavior": behavior}
    if message:
        # Deliver the human's feedback to the model. Field names vary across
        # Claude Code versions; set the documented one plus common fallbacks.
        decision["message"] = message
    out = {
        "hookSpecificOutput": {
            "hookEventName": "PermissionRequest",
            "decision": decision,
        }
    }
    if message:
        out["systemMessage"] = message
    print(json.dumps(out))
    sys.exit(0)


def main():
    try:
        payload = json.loads(sys.stdin.read() or "{}")
    except ValueError:
        # Can't parse input: don't block the user, fall through to normal flow.
        sys.exit(0)

    plan = (payload.get("tool_input") or {}).get("plan") or ""
    session_id = payload.get("session_id") or "plan"
    label = f"plan:{session_id}"

    if not plan.strip():
        # Nothing to review; allow normally.
        emit("allow")

    try:
        base = common.ensure_server()
        resp = common.api_post(
            "/api/session",
            {"mode": "plan", "label": label, "markdown": plan},
            base_url=base,
        )
        url = resp.get("session", {}).get("url")
        key = resp.get("session", {}).get("key")
        if url:
            print(f"[plan-ui] review your plan at {url}", file=sys.stderr)
            common.open_browser(url)

        # Block until the human decides in the browser.
        decision = common.api_get(f"/api/{key}/await-decision", base_url=base, timeout=None)
    except Exception as e:  # noqa: BLE001
        # If the review layer fails, don't wedge planning — allow and note why.
        print(f"[plan-ui] review unavailable ({e}); allowing plan.", file=sys.stderr)
        emit("allow")
        return

    if decision.get("decision") == "approve":
        emit("allow")
    else:
        feedback = decision.get("feedback") or "The user requested changes to the plan."
        emit("deny", feedback)


if __name__ == "__main__":
    main()
