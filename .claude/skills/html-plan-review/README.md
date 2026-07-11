# html-plan-review

An alternative to reviewing Claude Code plans as walls of markdown: this
skill has the agent publish its plan to an **interactive local HTML review
page** where you can

- comment on any plan section (click **💬 Comment**),
- select any text span and comment on exactly that phrase — the highlight
  stays visible in the document,
- mark each section **✓ Looks good** / **✗ Needs changes**,
- approve the plan or request changes from a sticky bottom bar.

Your annotations auto-save as you type (reload-safe, always visible with a
"pending — not sent" badge until you submit). When you hit **Request
changes**, the feedback is delivered straight back into the Claude Code
session through a tiny local web server; Claude revises the plan, replies
to each of your comments PR-review-style (open/resolved threads), and the
page refreshes itself with the new round. **Approve plan** ends the loop
and Claude proceeds to implementation.

## Install

Personal skill (all projects):

```sh
cp -r .claude/skills/html-plan-review ~/.claude/skills/
```

Or keep it in a project's `.claude/skills/` for that project only.

Requirements: `python3` on PATH (stdlib only, no pip installs). No npm, no
build step; the review page is a single self-contained HTML file.

## Usage

The skill auto-triggers whenever Claude presents a plan for approval
(including plan mode). You can also ask for it explicitly:

> Plan the auth refactor and give me the HTML review page.

Claude replies with a URL like `http://127.0.0.1:4173` — open it, annotate,
submit. To make plan mode always use it, add a line to your `~/.claude/CLAUDE.md`:

> When presenting any plan for approval, use the html-plan-review skill.

## How it works

```
Claude                          plan_review_server.py            your browser
  │  writes plan.json  ──────►  .claude/plan-review/<slug>/
  │  serve (background) ──────► http://127.0.0.1:4173  ────────►  review page
  │  wait --round N (background)                                   │ annotate
  │        ▲                                                       │ auto-saves draft
  │        └── feedback-round-N.json ◄── POST /api/submit ◄────────┘ Send/Approve
  │  wakes, revises plan.json (threads + replies), round N+1 …
```

State lives in `.claude/plan-review/<slug>/` — gitignore it, or commit it
as a decision record of what was proposed, challenged, and approved.

## Limitations

- Your browser must be able to reach the machine running Claude Code, so
  this is for **local** CLI/desktop sessions (or a tunnel/port-forward to a
  remote box). In Claude Code on the web the container's localhost isn't
  reachable; the skill falls back to a normal markdown plan.
- The server binds `127.0.0.1` only and serves a single review directory.

## Files

```
SKILL.md                        agent protocol + JSON schemas
scripts/plan_review_server.py   stdlib HTTP server: serve / wait / stop
assets/app.html                 the review UI (vanilla JS, light/dark)
examples/plan.json              sample plan for a quick demo
```

Quick demo without Claude:

```sh
python3 scripts/plan_review_server.py serve --dir /tmp/demo-review &
cp examples/plan.json /tmp/demo-review/
open http://127.0.0.1:4173
```
