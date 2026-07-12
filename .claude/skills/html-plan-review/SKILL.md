---
name: html-plan-review
description: Present implementation plans as an interactive local HTML review page instead of a markdown wall. Use whenever presenting a plan for user approval — in plan mode before calling ExitPlanMode, or when the user asks to review a plan/spec/design. The user annotates only what they want changed in the browser (comments, cross-outs, flags), and the feedback returns automatically through a local server; iterate rounds until approved.
---

# html-plan-review

Serve the plan as a rich document at `http://127.0.0.1:<port>`. The
review model is **critique-only**: the user touches nothing they're
happy with. They comment on or cross out any selected text, then click
**Approve plan** or **Request changes**; feedback lands back in this
session automatically. Revisions replace the page with only the clean
latest version — never inline change-tracking.

Requires `python3` on PATH (stdlib only) and a browser that can reach
this machine (local CLI / desktop sessions). Otherwise fall back to a
normal markdown plan.

## Files you write (state dir: `.claude/plan-review/<kebab-slug>/`)

- **`plan.html`** — the plan body: raw, multi-line, nicely indented
  HTML. **Read `reference.md` in this skill directory before authoring
  your first plan of the session** — it has the component vocabulary
  (stat tiles, steps, tabs, diffs, file trees, callouts…) and house
  style. No JSON escaping, ever.
- **`plan.json`** — tiny metadata only:

      { "title": "Add OAuth login", "round": 1, "updatedAt": "2026-07-11 14:03" }

## Workflow

Let `SERVER` = `python3 "<this skill dir>/scripts/plan_review_server.py"`.

**Publish round 1:** write both files, then run two **background**
tasks (never poll or sleep — the watcher wakes you on submit):

    $SERVER serve --dir <dir> --port 4173     # prints the URL; idempotent
    $SERVER wait  --dir <dir> --round 1        # exits when feedback lands

Tell the user the URL and end your turn.

**On feedback** (the watcher's stdout, schema below):

- `verdict: "approve"` — done. Attached comments are non-blocking
  notes. `$SERVER stop --dir <dir>`; if in plan mode, call ExitPlanMode
  noting browser approval.
- `verdict: "request_changes"` — revise **with targeted Edit calls on
  `plan.html`** — never regenerate the whole file for localized
  feedback. Address every annotation and the `overallComment`; anything
  untouched is implicitly approved, so don't rework it. If you disagree
  with an annotation, raise it in chat, not in the page. Then bump
  `round`/`updatedAt` in `plan.json` (the page live-refreshes), start
  `wait --round N+1` in the background, tell the user briefly, and end
  your turn.

## Feedback schema

    {
      "round": 1,
      "verdict": "approve | request_changes",
      "overallComment": "may be empty",
      "newThreads": [
        { "sectionId": "_doc", "quote": "exact selected text or null",
          "type": "comment | strike", "text": "…" }
      ],
      "sectionVerdicts": { "api": "needs_changes" },
      "submittedAt": "…"
    }

`quote` targets that exact phrase (`sectionId` `"_doc"` = the
`plan.html` body, `"_summary"` = summaryHtml metadata if you used it).
**`type: "strike"` means the user crossed the text out — remove that
element from the plan**; `text` is their optional reason. A null
`quote` targets a whole section. `sectionVerdicts` only appears for
sectioned plans (an opt-in — see reference.md; free-form `plan.html`
is the default and preferred style).
