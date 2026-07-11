---
name: html-plan-review
description: Present implementation plans as an interactive local HTML review page instead of a markdown wall. Use whenever presenting a plan for user approval — in plan mode before calling ExitPlanMode, or when the user asks to review a plan/spec/design. The user annotates only what they want changed in the browser (comments, flags), and the feedback returns automatically through a local server; iterate rounds until approved.
---

# html-plan-review

Replace markdown plan dumps with an interactive review page served at
`http://127.0.0.1:<port>`. The review model is **critique-only**: the user
touches nothing they're happy with. They comment on sections or selected
text, flag sections as "needs changes", then click **Approve plan** or
**Request changes**. Their feedback lands back in this session
automatically. While you revise, the page shows a waiting spinner; when
you publish the next round it replaces the page with **only the current
latest version** (revision counter + "awaiting your review" banner — never
inline change-tracking or thread history).

Requires `python3` on PATH (stdlib only). The user's browser must be able
to reach the machine running Claude Code (local CLI / desktop sessions).
If either is unavailable, fall back to a normal markdown plan.

## Workflow

Let `SKILL_DIR` be this skill's directory and `SERVER` be
`python3 "$SKILL_DIR/scripts/plan_review_server.py"`.

### Round 1 — publish the plan

1. Pick a state dir: `.claude/plan-review/<kebab-case-slug>/` in the
   project (create it). Suggest gitignoring `.claude/plan-review/` if the
   user hasn't decided to keep reviews as decision records.
2. Write `<dir>/plan.json` following the schema below.
3. Start the server as a **background** task (it blocks; it is idempotent —
   if one is already serving this dir it prints the existing URL and exits):

       $SERVER serve --dir <dir> --port 4173

   Capture the printed URL.
4. Start the feedback watcher as a **background** task:

       $SERVER wait --dir <dir> --round 1

5. Tell the user the URL and end your turn. Do NOT poll or sleep — the
   watcher exits (waking you) the instant feedback is submitted.

### On feedback (the watcher's stdout is the feedback JSON)

- **`verdict: "approve"`** — the plan is approved. Treat any attached
  comments as non-blocking implementation notes. Stop the server
  (`$SERVER stop --dir <dir>`). If you are in plan mode, now call
  ExitPlanMode noting the plan was approved in the review page.
- **`verdict: "request_changes"`** — revise and publish round N+1:
  1. Address every comment (`newThreads`), flagged section
     (`sectionVerdicts`), and the `overallComment`. Anything the user did
     NOT touch is implicitly approved — do not rework it.
  2. Rewrite `plan.json` as the **complete current plan only**: bump
     `round`, set `updatedAt`, revise the sections. Do NOT embed the
     user's comments, your responses, changelogs, or any what-changed
     annotations — the page must show a clean latest version. If you
     disagree with a comment or need input, raise it in chat, not in the
     plan page.
  3. Start `wait --round N+1` in the background, briefly tell the user
     the revision is up (same URL, page refreshes itself), and end your
     turn. You may summarize what you changed in chat.

Repeat until approved. Keep section `id`s stable across rounds so the
user's mental map (and any unsent draft anchors) survive revisions.

## plan.json schema

```json
{
  "title": "Add OAuth login",
  "round": 1,
  "updatedAt": "2026-07-11 14:03",
  "summaryHtml": "<p>One-paragraph overview. Optional.</p>",
  "bodyHtml": "<p>Optional free-form document — use INSTEAD of sections when the plan doesn't decompose naturally.</p>",
  "sections": [
    {
      "id": "db",
      "heading": "1. Database changes",
      "bodyHtml": "<p>…</p><ul><li>…</li></ul><pre><code>…</code></pre>"
    }
  ]
}
```

**Default to a free-form `bodyHtml` document** — one flowing page with
`h3` headings and rich components, like a well-designed product doc.
Every part of it is annotatable by selection. Only reach for `sections`
when the user explicitly wants per-section comment/flag controls; they
render as headings with hover tools, not boxes. See
`examples/plan.json` for the house style.

Authoring rules: never include scripts or event handlers
in `bodyHtml`. Base tags: `p ul ol li pre code kbd table tr td th thead
tbody strong em h3 h4 details summary`. The page also enhances these
**rich components — use them**, they are the point of this skill:

- **Stat tiles** — lead with the plan's shape at a glance:

      <div class="stats">
        <div class="stat"><div class="v">14</div><div class="l">files touched</div></div>
        <div class="stat"><div class="v">~2 wk</div><div class="l">to full rollout</div></div>
      </div>

- **Steps** — `<ol class="steps"><li><strong>Title</strong>detail…</li>…</ol>`
  renders a numbered timeline; ideal for the approach/phases.
- **Tables** — plain `<table>`, auto-styled (header band, row hover).
  Use for migrations, file-change lists, API matrices.
- **Tabs** — for alternatives, trade-offs, per-platform variants:

      <div class="tabs">
        <section data-tab="Option A — cookies">…</section>
        <section data-tab="Option B — JWT">…</section>
      </div>

- **Callouts** — `<div class="callout info|warn|risk|success">…</div>`
  for risks, caveats, and wins (icon added automatically).
- **File tree** — `<ul class="file-tree"><li class="dir">src/auth/</li>
  <ul><li class="add">oauth.ts</li><li class="mod">session.ts</li></ul></ul>`
  (`add` = new file, `mod` = modified).
- **Diff blocks** — `<pre class="diff"><code>@@ file @@\n-old\n+new</code></pre>`
  gets +/- line coloring automatically.
- **Pills** — `<span class="pill info|ok|warn|danger">label</span>` for
  inline status like flag names or reversibility.
- **Collapsibles** — `<details><summary>Label</summary>…</details>` for
  long detail that would bloat the page.
- **Columns** — `<div class="cols"><div class="col">…</div>…</div>` for
  side-by-side comparison (pros/cons inside tabs work well).

Every part of the plan is annotatable: the user can select any text —
in the summary, a table cell, a tab panel — and comment on it or cross
it out.

## Feedback schema (what `wait` prints)

```json
{
  "round": 1,
  "verdict": "approve | request_changes",
  "overallComment": "free text, may be empty",
  "newThreads": [
    { "id": "d…", "sectionId": "db", "quote": "exact selected text or null",
      "type": "comment | strike", "text": "…" }
  ],
  "sectionVerdicts": { "api": "needs_changes" },
  "submittedAt": "…"
}
```

`quote` non-null means the annotation targets that exact phrase; null
means it applies to the section as a whole. `sectionId` is `"_summary"`
for the plan summary and `"_doc"` for a free-form `bodyHtml` document.
`type: "strike"` means the user **crossed the quoted text out — remove
or eliminate that element from the plan** (the `text`, if present, is
their reason). Anything untouched carries no entry anywhere — treat it
as approved. (In the UI, staged annotations live in a drawer opened
from the footer bar, not inline — the document stays clean apart from
the highlights themselves.)
