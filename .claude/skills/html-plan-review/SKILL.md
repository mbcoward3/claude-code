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
  HTML. **Read `references/REFERENCE.md` in this skill before authoring
  your first plan of the session** — it has the component vocabulary
  (stat tiles, steps, tabs, diffs, file trees, callouts…) and house
  style. No JSON escaping, ever.
- **`plan.json`** — tiny metadata only (`updatedAt` optional; the file's
  mtime is shown when omitted):

      { "title": "Add OAuth login", "round": 1 }

When the state dir is inside a git repo, append `.claude/plan-review/`
to `.git/info/exclude` on first publish so review state never shows up
as untracked noise (skip if the user wants review records committed).

## Workflow

Let `SERVER` = `python3 "<this skill dir>/scripts/plan_review_server.py"`.

**Publish round 1:** write both files, then run two **background**
tasks (never poll or sleep — the watcher wakes you on submit):

    $SERVER serve --dir <dir> --port 4173     # prints the URL; idempotent
    $SERVER wait  --dir <dir> --round 1        # exits when feedback lands

Give the user the printed URL **verbatim** — it carries a `?t=` access
token the page needs — and end your turn.

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

## Brief rounds — consensus before the plan (optional)

Publishing a plan asserts you have the context to defend it. When you
don't — requirements genuinely ambiguous, a decision genuinely open —
publish a **brief** on the same page first instead of asking in chat:
same files and workflow, but `plan.json` carries `"stage": "brief"`
(rounds share one counter: brief round 1, then plan round 2, …).

A brief leads with your current best understanding — what you'd build
absent answers — then asks 2–5 **multiple-choice** questions whose
options you can defend, each with its consequence:

    <div class="question" data-q="transport">
      <h4>Which remote transport should the plan target?</h4>
      <label data-opt="relay"><strong>Signed relay URL</strong>
        one-click for reviewers; needs a token story</label>
      <label data-opt="ssh"><strong>SSH port-forward</strong>
        zero server code; manual step each session</label>
    </div>

`data-multi` on the div allows several selections. The page always
appends a "Something else…" free-text option — never rely on your
enumeration being complete. Don't ask what you can infer.

Feedback arrives with `verdict: "answers"` plus

    "answers": { "transport": { "selected": ["relay"], "other": null } }

(`"other"` non-null = the user's own alternative; an unanswered
question means "your call"). **On answers, you decide the next round:**
publish the plan (`"stage": "plan"`) if consensus is reached — the
normal case — or one follow-up brief only if the answers opened a
genuinely new question. Never exceed two briefs; if consensus still
hasn't formed, move to chat. While any round is open, never also ask
questions in chat — one surface at a time.

## Feedback schema

    {
      "round": 1,
      "stage": "plan | brief",
      "verdict": "approve | request_changes | answers",
      "answers": { "…": { "selected": ["…"], "other": null } },
      "overallComment": "may be empty",
      "newThreads": [
        { "sectionId": "_doc", "quote": "selected text or null",
          "type": "comment | strike", "text": "…",
          "context": { "block": "li.add", "heading": "File changes" } }
      ],
      "sectionVerdicts": { "api": "needs_changes" },
      "submittedAt": "…"
    }

`quote` targets that phrase (`sectionId` `"_doc"` = the `plan.html`
body, `"_summary"` = summaryHtml metadata if you used it). Quotes are
whitespace-collapsed rendered text, so they may not match plan.html
byte-for-byte across line breaks — locate them tolerantly. `context`
(when present) names the block element the selection sits in and the
last heading above it. **`type: "strike"` means the user crossed the
text out — remove the annotated element (per `context`/`quote`) from
the plan**; `text` is their optional reason. A null `quote` targets a
whole section. `sectionVerdicts` only appears for
sectioned plans (an opt-in — see references/REFERENCE.md; free-form `plan.html`
is the default and preferred style).
