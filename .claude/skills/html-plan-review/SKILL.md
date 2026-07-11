---
name: html-plan-review
description: Present implementation plans as an interactive local HTML review page instead of a markdown wall. Use whenever presenting a plan for user approval — in plan mode before calling ExitPlanMode, or when the user asks to review a plan/spec/design. The user annotates sections, comments on selected text, sets verdicts in the browser, and the feedback returns automatically through a local server; iterate rounds until approved.
---

# html-plan-review

Replace markdown plan dumps with an interactive review page served at
`http://127.0.0.1:<port>`. The user comments on sections or selected text,
marks sections ✓/✗, and clicks **Approve plan** or **Request changes**.
Their feedback lands back in this session automatically. Revised plans
re-render in the same tab with the user's earlier comments shown as
threads carrying your replies and open/resolved states.

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
  1. For each entry in `newThreads`, add a thread to `plan.json`:
     `{id, sectionId, quote, status, messages:[{author:"user", round:N, text}]}`
     (reuse the draft `id`).
  2. For each entry in `replies`, append
     `{author:"user", round:N, text}` to that thread's `messages` and set
     its `status` to `"open"`.
  3. Revise the plan sections to address the feedback (also weigh
     `sectionVerdicts` and `overallComment`).
  4. For every open thread, append your reply
     `{author:"agent", round:N+1, text:"<what you changed / why not>"}`
     and set `status` to `"resolved"` if addressed, or leave `"open"` if
     you need the user's input or respectfully disagree.
  5. Bump `round`, set `updatedAt`, append `{round:N, verdict}` to
     `history`, and rewrite `plan.json`. The page live-refreshes.
  6. Start `wait --round N+1` in the background, briefly tell the user the
     revision is up (same URL), and end your turn.

Repeat until approved. Keep **section `id`s stable across rounds** so
threads and highlights stay anchored; if you delete a section, resolve its
threads with an explanatory reply and mention the removal in `summaryHtml`.

## plan.json schema

```json
{
  "title": "Add OAuth login",
  "round": 2,
  "updatedAt": "2026-07-11 14:03",
  "summaryHtml": "<p>One-paragraph overview. Optional.</p>",
  "sections": [
    {
      "id": "db",
      "heading": "1. Database changes",
      "bodyHtml": "<p>…</p><ul><li>…</li></ul><pre><code>…</code></pre>"
    }
  ],
  "threads": [
    {
      "id": "d1720000000001",
      "sectionId": "db",
      "quote": "drop the sessions table",
      "status": "resolved",
      "messages": [
        { "author": "user",  "round": 1, "text": "Don't drop this — analytics reads it." },
        { "author": "agent", "round": 2, "text": "Kept the table; added a deprecation note instead." }
      ]
    }
  ],
  "history": [ { "round": 1, "verdict": "request_changes" } ]
}
```

Authoring rules: 3–10 sections; `bodyHtml` uses simple tags only
(`p ul ol li pre code table tr td th strong em h3 h4`) — never scripts or
event handlers; `quote` must be an exact substring of the section's
rendered text or the highlight is skipped (the comment still shows).

## Feedback schema (what `wait` prints)

```json
{
  "round": 1,
  "verdict": "approve | request_changes",
  "overallComment": "free text, may be empty",
  "newThreads": [ { "id": "d…", "sectionId": "db", "quote": "…or null", "text": "…" } ],
  "replies": [ { "threadId": "d1720000000001", "text": "…" } ],
  "sectionVerdicts": { "db": "approved", "api": "needs_changes" },
  "submittedAt": "…"
}
```
