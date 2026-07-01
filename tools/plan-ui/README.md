# plan-ui

A Claude Code / Codex plugin that gives agents and humans a **rich, browser-based
surface to collaborate on a plan**. It has two entry paths into one review UI:

1. **ExitPlanMode review** (Claude Code, automatic) — when the agent finishes
   native plan mode, a hook intercepts it, renders the plan, and opens a review
   UI. You **approve** or **request changes**, and the feedback goes back to the
   model. (Inspired by [plannotator](https://github.com/backnotprop/plannotator).)
2. **HTML artifact loop** (Claude Code **and** Codex, manual) — the agent authors
   a plan as an HTML file and drives `plan-ui open / poll / end`; you annotate
   specific steps or lines inline and it iterates. (Inspired by
   [lavish-axi](https://github.com/kunchenguid/lavish-axi).)

Pure Python standard library — **no pip installs, no build step, no CDN**. The
plugin runs in place.

## Layout

```
plan-ui/
  .claude-plugin/plugin.json    plugin manifest
  hooks/hooks.json              PermissionRequest → ExitPlanMode → hook.py
  skills/plan-ui/SKILL.md       trigger + pointer to `plan-ui playbook`
  bin/plan-ui                   launcher (added to PATH while the plugin is enabled)
  scripts/
    plan_ui.py                  CLI: open / poll / end / stop / playbook / serve
    server.py                   stdlib HTTP server: sessions, SSE, long-poll, gate, watch
    hook.py                     ExitPlanMode entry: plan → review UI → decision
    mdrender.py                 dependency-free Markdown → HTML (for plan mode)
    common.py                   paths, session keys, server discovery, HTTP client
  web/
    shell.html                  review frame: artifact iframe + conversation/decision panel
    sdk.js                      annotation SDK injected into the artifact
    chrome.css                  styles for the injected SDK UI
    tailwind.js / daisyui.css   vendored design system (served locally)
  playbook.md                   plan-authoring guidance (`plan-ui playbook`)
```

## Commands (artifact loop)

| Command | Purpose |
| --- | --- |
| `plan-ui open <file> [--no-open]` | Serve a plan artifact and open it in the browser |
| `plan-ui poll <file> [--agent-reply T] [--timeout-ms N]` | Long-poll for feedback (blocks; never kill it) |
| `plan-ui end <file>` | End a session |
| `plan-ui stop` | Shut down the background server |
| `plan-ui playbook` | Print the plan-authoring playbook |

Every agent-facing command prints JSON with a `next_step` field, so the loop is
self-describing.

## How it works

Sessions are keyed by the plan file's canonical path (artifact mode) or the
Claude session id (plan mode). The first command spawns a background
`server.py`; subsequent commands and the browser talk to it over HTTP. Feedback
reaches a waiting `poll` through a long-poll; the browser gets live updates
(reload, agent replies, presence, gate status) over Server-Sent Events. All
state lives under `~/.plan-ui/`.

The **layout gate** masks the plan in the browser until an automated audit
(horizontal overflow, clipped text) passes; failures are reported back to the
agent as `layout_warnings`.

## Install

**Claude Code** — load locally for development:

```
claude --plugin-dir ./tools/plan-ui
```

or install from a marketplace once published. Enabling the plugin puts `plan-ui`
on `PATH`, registers the skill, and wires the ExitPlanMode hook.

**Codex** — the artifact loop works from any harness that can run a shell
command. Point Codex at `scripts/plan_ui.py` (see `codex/`).

## Requirements

- **Python 3.8+** (tested on 3.11). Standard library only.
- A browser to view the review UI.

`tailwind.js` and `daisyui.css` are placeholders pending a one-time vendor step;
plan mode renders without them (it ships its own prose styles), and artifact mode
works with whatever styles the agent's HTML brings.
