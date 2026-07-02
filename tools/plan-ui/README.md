# plan-ui

A plugin that gives agents and humans a **rich, browser-based surface to
collaborate on a plan**. The agent authors the plan as an HTML artifact and
drives `plan-ui open / poll / end`; the human annotates specific steps or lines
inline in the browser; the agent applies the feedback and the loop repeats until
the plan is approved.

One skill triggers it everywhere: the same `SKILL.md` works natively in both
**Claude Code** and **Codex** (both support the Agent Skills standard).

Pure Python standard library — **no pip installs, no build step, no CDN**. The
plugin runs in place.

> Inspired by [lavish-axi](https://github.com/kunchenguid/lavish-axi)'s
> artifact-review model and [plannotator](https://github.com/backnotprop/plannotator)'s
> plan-review UX, rebuilt as a single skill-triggered tool.

## Layout

The directory follows the [Agent Skills](https://agentskills.io) conventions —
`SKILL.md` at the root with the workflow instructions, `scripts/` for
executable code, `references/` for docs the agent loads as needed, `assets/`
for files used in output:

```
plan-ui/
  SKILL.md                      the skill: trigger description + review workflow
  references/playbook.md        plan authoring rules (content, layout, design language)
  scripts/
    plan_ui.py                  CLI: open / poll / end / stop / playbook / serve
    server.py                   stdlib HTTP server: sessions, SSE, long-poll, gate, watch
    common.py                   paths, session keys, server discovery, HTTP client
    install-codex.sh            one-command Codex install (symlink into ~/.agents/skills)
  assets/
    sdk.js                      review layer injected into the plan document
    chrome.css                  styles for the injected review UI
  .claude-plugin/plugin.json    Claude Code plugin manifest
  agents/openai.yaml            Codex skill metadata (display name, invocation policy)
  bin/plan-ui                   launcher (added to PATH while the plugin is enabled)
  tests/smoke.sh                end-to-end test of the loop
```

Because a skill is *defined* as a directory with a root `SKILL.md` plus
supporting files, and Claude Code loads a single-skill plugin from a root
`SKILL.md`, this one directory is simultaneously a spec-shaped Agent Skill and
a valid Claude Code plugin.

## Commands

| Command | Purpose |
| --- | --- |
| `plan-ui open <file> [--no-open]` | Serve a plan artifact and open it in the browser |
| `plan-ui poll <file> [--agent-reply T] [--timeout-ms N]` | Long-poll for feedback (blocks; never kill it) |
| `plan-ui end <file>` | End a session |
| `plan-ui stop` | Shut down the background server |
| `plan-ui playbook` | Print the plan-authoring playbook |

Every agent-facing command prints JSON with a `next_step` field, so the loop is
self-describing.

## Reviewing a plan (the human side)

The plan document is the whole UI — there is no sidebar. The reviewer:

- **Clicks any element or selects text** to attach a comment to that exact spot.
  Every queued annotation stays **visibly marked in place** (highlight + numbered
  chip); clicking a marker reopens it to **edit or delete** before sending.
- Uses the **floating toolbar** to act: *Send annotations* ships everything back
  to the agent; *Approve plan* signs off. A status line tracks the round-trip:
  queued count → "feedback sent" → "agent is working…" → "plan updated — review
  the changes" when the revision live-reloads.
- Unsent annotations survive reloads; agent replies appear above the toolbar.

## How it works

Sessions are keyed by the plan file's canonical path — no session ids to track.
The first command spawns a background `server.py`; subsequent commands and the
browser talk to it over HTTP. Feedback reaches a waiting `poll` through a
long-poll; the browser gets live updates (reload, agent replies, presence, gate
status) over Server-Sent Events. All state lives under `~/.plan-ui/`.

The **layout gate** masks the plan in the browser until an automated audit
(horizontal overflow, clipped text) passes; failures are reported back to the
agent as `layout_warnings`.

**Harness-agnostic by design:** the core (server, CLI, review SDK) has no
knowledge of any specific agent product, and the trigger is a standard
Agent-Skills `SKILL.md`. Any harness that can run a shell command can drive
`scripts/plan_ui.py` the same way.

## Install

**Claude Code** — from this repo's marketplace:

```
/plugin marketplace add mbcoward3/claude-code
/plugin install plan-ui@plan-ui
```

or load locally for development:

```
claude --plugin-dir ./tools/plan-ui
```

Enabling the plugin puts `plan-ui` on `PATH` and registers the skill.

**Codex** — Codex supports the same Agent Skills standard, and this directory
*is* a valid Codex skill: root `SKILL.md` (the only required file) plus the
optional `agents/openai.yaml` metadata (display name, brand color, implicit
invocation policy). Install with one command:

```
bash scripts/install-codex.sh
```

which symlinks the directory to `~/.agents/skills/plan-ui` — the user-scope
skills location per the Codex docs (set `SKILLS_DIR` to override, e.g.
`.agents/skills` for a repo-scoped install). Codex then triggers the skill
implicitly when a task matches its description, or explicitly via `$plan-ui`.
Since Codex does not manage `PATH`, the skill instructs the agent to call
`python3 <skill-dir>/scripts/plan_ui.py …` directly.

(The `agents/` directory name is also used by Claude Code plugins for custom
agent definitions, but those are `.md` files — the lone `openai.yaml` is
ignored by Claude Code.)

## Requirements

- **Python 3.8+** (tested on 3.11). Standard library only — no pip installs.
- A browser to view the review UI.

There are **no CSS frameworks** — no Tailwind, no CDN, no vendored bundles.
Agents author plans as fully self-contained HTML with an embedded `<style>`
block, following the compact design language in the playbook. This keeps the
plugin tiny and means an artifact renders identically whether opened through
plan-ui or directly from disk. The background server exits on its own after 30
idle minutes.

## Tests

```
bash tests/smoke.sh
```

Runs the loop end-to-end (session, SDK injection, feedback→poll delivery,
layout-gate warnings, approve action) against an isolated HOME.
