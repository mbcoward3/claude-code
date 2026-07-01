---
name: plan-ui
description: >-
  Collaborate with the user on a plan in a rich, annotatable browser UI instead
  of plain-text plan mode. Use when presenting a technical or product plan,
  approach, design, or proposal for review: you author the plan as an HTML
  artifact, the user annotates specific elements or text inline and sends
  feedback, and you iterate through a poll loop until they approve. Prefer this
  over emitting a plan as prose whenever the user will want to review, comment
  on, or sign off on the plan.
---

# plan-ui

Turn a plan into a **reviewable, annotatable HTML artifact** the user opens in a
browser. They mark up specific steps or lines, send structured feedback, and you
refine — a richer replacement for plain plan mode.

## When to use this

Reach for plan-ui instead of writing a plan as prose when **any** of these hold:

- You are proposing an approach, design, migration, or multi-step plan the user
  will want to review before you implement.
- The plan benefits from structure the user can react to part-by-part (steps,
  tradeoffs, risks, comparisons, diagrams).
- The user is likely to approve/reject or request targeted changes.

For a throwaway one-liner or a plan the user did not ask to review, plain text is
fine — don't over-reach.

## The loop

1. **Author** the plan as a self-contained HTML file, e.g. `.plan-ui/<name>.html`.
   Run `plan-ui playbook` first and follow it — verify every claim against the
   codebase, surface risks and open questions, and avoid horizontal overflow.
2. **Open** it:

   ```
   plan-ui open .plan-ui/<name>.html
   ```

   This serves the plan, opens the browser, and returns JSON with a `next_step`.

3. **Poll** for feedback:

   ```
   plan-ui poll .plan-ui/<name>.html
   ```

   This blocks silently until the user annotates or messages. **Never kill a
   running poll** — it is waiting, not stuck. It returns the user's `prompts`
   (each with the DOM `target` it refers to) and any `layout_warnings`.

4. **Apply** the feedback by editing the HTML file. It live-reloads in the
   browser.

5. **Reply and wait again**, showing the user what you changed:

   ```
   plan-ui poll .plan-ui/<name>.html --agent-reply "Tightened step 1 and added the rollback risk you flagged."
   ```

6. **End** when the user approves or is done:

   ```
   plan-ui end .plan-ui/<name>.html
   ```

## Reading feedback

Each prompt in a poll result looks like:

```json
{ "text": "make this concrete", "action": "comment",
  "target": { "selector": "li:nth-of-type(2)", "quoted_text": "do the thing" } }
```

- `action` is `comment`, `approve`, `request-changes`, or a custom id from a
  `data-plan-action` button you added.
- `target` tells you **exactly which part** of the plan the comment is about —
  use it to make a precise edit, not a broad rewrite.

## The layout gate

plan-ui masks the plan in the browser until an automated layout audit passes
(no horizontal overflow, no clipped text). If it fails, the warnings come back
to you on `poll` as `layout_warnings` — fix them before expecting user feedback.

## Interactive actions

Give the user one-click actions by adding `data-plan-action` to an element; the
action id and label arrive in your next poll:

```html
<button data-plan-action="approve" data-plan-label="Looks good, proceed">
  Approve plan
</button>
```

## Design

Tailwind (v4, JIT) and DaisyUI v5 are served locally by plan-ui — use their
utility classes freely. No CDN is needed or available; the plan renders offline
and identically whether opened directly or through plan-ui.

## Installing plan-ui

If `plan-ui` is not on `PATH`, run the bundled bootstrap, which fetches the
binary for this platform from the public package registry and caches it under
`~/.plan-ui/bin`:

```
bash "$(dirname "$0")/bootstrap.sh" open .plan-ui/<name>.html
```

Once installed it stays cached; subsequent calls are instant.
