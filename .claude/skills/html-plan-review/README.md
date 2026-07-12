# html-plan-review

An alternative to reviewing Claude Code plans as walls of markdown: this
skill has the agent publish its plan to an **interactive local HTML review
page** where you can

- select **any text anywhere** — summary, a table cell, inside a tab —
  and either **💬 comment** on that exact phrase or **✗ cross it out**
  (strike-through = "remove this", with an optional reason); highlights
  and strikes stay visible in the document,
- comment on or flag a whole section when the plan uses sections
  (sections are optional — free-form documents are fully annotatable),
- review everything you've staged in a **drawer** popped up from the
  footer bar ("N staged"), where each annotation can be edited, deleted,
  or jumped to — the document itself stays clean apart from highlights,
- approve the plan or request changes from the same sticky bottom bar.

Plans render with rich components, not just prose: auto-styled tables,
**multi-tab sections** for alternatives, callouts for risks, collapsible
detail blocks, and side-by-side comparison columns.

The review is **critique-only**: anything you don't touch counts as
approved, so a plan you like is a single click of **Approve plan**. Your
annotations auto-save as you type (reload-safe; highlights stay in the
document and everything staged is listed in the footer drawer until you
submit). When you hit **Request
changes**, the feedback is delivered straight back into the Claude Code
session through a tiny local web server and the page shows a
"waiting for Claude to make changes…" spinner. When the revision is ready
the page replaces itself with the clean latest version — revision counter
incremented, "awaiting your review" banner, no inline change-tracking.
**Approve plan** ends the loop and Claude proceeds to implementation.

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

Claude replies with a URL like `http://127.0.0.1:4173/?t=…` — open it,
annotate, submit. The `?t=` token gates the plan content and feedback
endpoints, so keep the URL intact. To make plan mode always use it, add a line to your `~/.claude/CLAUDE.md`:

> When presenting any plan for approval, use the html-plan-review skill.

## How it works

```
Claude                          plan_review_server.py            your browser
  │  writes plan.json  ──────►  .claude/plan-review/<slug>/
  │  serve (background) ──────► http://127.0.0.1:4173  ────────►  review page
  │  wait --round N (background)                                   │ annotate
  │        ▲                                                       │ auto-saves draft
  │        └── feedback-round-N.json ◄── POST /api/submit ◄────────┘ Send/Approve
  │  wakes, revises plan.json (clean latest version), round N+1 …
```

State lives in `.claude/plan-review/<slug>/` — gitignore it, or commit it
as a decision record of what was proposed, challenged, and approved.

## Limitations

- Your browser must be able to reach the machine running Claude Code, so
  this is for **local** CLI/desktop sessions (or a tunnel/port-forward to a
  remote box). In Claude Code on the web the container's localhost isn't
  reachable; the skill falls back to a normal markdown plan.
- The server binds `127.0.0.1` only and serves a single review directory.
  Content and feedback endpoints additionally require the per-server
  `?t=` token, and non-localhost `Host` headers are rejected.
- Text inside an inactive tab can't be selected for annotation until you
  switch to that tab.

## Token efficiency

The plan body is authored as **raw multi-line HTML in `plan.html`** —
no JSON string escaping — with a ~6-line `plan.json` for metadata.
Revisions are targeted `Edit` calls on `plan.html` plus a round bump,
not whole-plan regeneration, so each review round costs a few hundred
output tokens instead of thousands. Feedback returns as compact
structured JSON (cheaper and less ambiguous than parsing prose
comments about a markdown plan). The component vocabulary lives in
`reference.md`, read only when authoring.

## Files

```
SKILL.md                        lean agent protocol + feedback schema
reference.md                    component vocabulary + house style
scripts/plan_review_server.py   stdlib HTTP server: serve / wait / stop
assets/app.html                 the review UI (vanilla JS, light/dark)
examples/plan.html              sample plan body (the house style)
examples/plan.json              sample metadata
```

Quick demo without Claude:

```sh
python3 scripts/plan_review_server.py serve --dir /tmp/demo-review &
# prints: Plan review server: http://127.0.0.1:4173/?t=<token>
cp examples/plan.html examples/plan.json /tmp/demo-review/
open "http://127.0.0.1:4173/?t=<token>"   # use the printed URL
```
