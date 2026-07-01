# plan-ui

A small CLI that gives agents and humans a **rich, browser-based surface to
collaborate on a plan**. The agent authors a plan as an HTML artifact; the human
opens it in a browser, annotates specific steps or lines inline, and sends
feedback; the agent polls for that feedback and iterates until the plan is
approved.

Think of it as a richer replacement for plain-text "plan mode": instead of a
wall of prose, the plan is a document the human can mark up part-by-part.

> Inspired by the artifact-review model of
> [lavish-axi](https://github.com/kunchenguid/lavish-axi), narrowed to the single
> job of collaborating on a plan, and rebuilt as a self-contained Go binary.

## How it works

```
agent writes plan.html
        │
        ▼
  plan-ui open plan.html ──▶ local HTTP server (embedded assets, no CDN)
        │                          │
        │                          ├─ serves the plan in an iframe + chat panel
        │                          ├─ injects the annotation SDK + Tailwind/DaisyUI
        │                          └─ runs a layout gate (masks until it renders cleanly)
        ▼
  human annotates in the browser  ──▶  queued as feedback
        │
        ▼
  plan-ui poll plan.html ──▶ returns { prompts[], layout_warnings[], next_step }
        │
        ▼
  agent edits plan.html (live-reloads) ──▶ poll --agent-reply "…" ──▶ repeat
        │
        ▼
  plan-ui end plan.html
```

Sessions are keyed by the plan file's canonical path — there are no session ids
to track. Every command just takes the file.

## Commands

| Command | Purpose |
| --- | --- |
| `plan-ui open <file> [--no-open]` | Serve a plan artifact and open it in the browser |
| `plan-ui poll <file> [--agent-reply TEXT] [--timeout-ms N]` | Long-poll for human feedback (blocks silently; never kill it) |
| `plan-ui end <file>` | End a session |
| `plan-ui stop` | Shut down the background server |
| `plan-ui playbook` | Print the plan-authoring playbook |

Every agent-facing command prints JSON with a `next_step` field, so the loop is
self-describing — the tool tells the agent what to do next.

### Feedback shape

```json
{
  "prompts": [
    { "text": "make this concrete",
      "action": "comment",
      "target": { "selector": "li:nth-of-type(2)", "quoted_text": "do the thing" } }
  ],
  "layout_warnings": [],
  "next_step": "Apply the feedback to the plan file, then run `plan-ui poll …`"
}
```

`target` locates exactly which element or text the comment refers to, so edits
can be precise. `action` is `comment`, `approve`, `request-changes`, or a custom
id from a `data-plan-action` button in the plan.

## Layout gate

The browser runs an automated layout audit (horizontal overflow, clipped text)
and the plan stays **masked until it passes**. Failures are reported back to the
agent on the next `poll` as `layout_warnings`, so a broken layout never reaches
the human silently.

## Design system

Tailwind CSS v4 (in-browser JIT via `@tailwindcss/browser`) and DaisyUI v5 are
**vendored into the binary** and served locally. No CDN is required or contacted
at runtime — the plan renders fully offline, and identically whether opened
directly in a browser or through plan-ui (the SDK is injected at serve time, not
saved into the file).

## Build

```
go build -o plan-ui .
```

Requirements:

- **Runtime:** none beyond the binary itself and a browser to view the UI. The
  only Go module dependency is [`kong`](https://github.com/alecthomas/kong) for
  CLI parsing; everything else is the standard library.
- **Build-time (assets):** Node/npm is used **once** to vendor
  `@tailwindcss/browser` and `daisyui` into `assets/` (see
  `scripts/vendor-assets.sh`). This is not needed to run the tool, only to
  refresh those two files.

## Layout of the source

```
main.go            Kong CLI definition (commands, flags)
commands.go        client-side command implementations + server discovery/spawn
server.go          HTTP server: session API, SSE, long-poll, file watching, lifecycle
session.go         session model + persisted store (~/.plan-ui/state.json)
coordination.go    in-memory poll waiters + SSE fan-out
gate.go            layout-gate state machine
transform.go       injects the SDK + styles into the served artifact
assets.go          go:embed of the browser assets
paths.go           ~/.plan-ui paths, canonical file resolution, session keys
httputil.go        JSON / SSE helpers
detach_*.go        platform-specific detached-process spawn
assets/
  shell.html       outer frame: artifact iframe + conversation/reply panel
  sdk.js           annotation SDK injected into the artifact iframe
  chrome.css       styles for the injected SDK UI
  playbook.md      plan-authoring guidance (printed by `plan-ui playbook`)
  tailwind.js      vendored Tailwind JIT runtime
  daisyui.css      vendored DaisyUI
skill/
  SKILL.md         the agent-facing skill that triggers plan-ui use
```

## For agents

The `skill/SKILL.md` file is a thin trigger: it tells an agent *when* to reach
for plan-ui and points it at `plan-ui playbook`, which carries the full loop and
authoring rules. In short: author an HTML plan, `open` it, `poll` for feedback,
edit, reply and poll again, then `end`.
