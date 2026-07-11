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
  "sections": [
    {
      "id": "db",
      "heading": "1. Database changes",
      "bodyHtml": "<p>…</p><ul><li>…</li></ul><pre><code>…</code></pre>"
    }
  ]
}
```

Authoring rules: 3–10 sections; `bodyHtml` uses simple tags only
(`p ul ol li pre code table tr td th strong em h3 h4`) — never scripts or
event handlers.

## Feedback schema (what `wait` prints)

```json
{
  "round": 1,
  "verdict": "approve | request_changes",
  "overallComment": "free text, may be empty",
  "newThreads": [
    { "id": "d…", "sectionId": "db", "quote": "exact selected text or null", "text": "…" }
  ],
  "sectionVerdicts": { "api": "needs_changes" },
  "submittedAt": "…"
}
```

`quote` non-null means the comment targets that exact phrase in the
section; null means it applies to the section as a whole. Untouched
sections carry no entry anywhere — treat them as approved.
