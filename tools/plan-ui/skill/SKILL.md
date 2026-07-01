---
name: plan-ui
description: >-
  Collaborate with the user on a plan in a rich, annotatable browser UI instead
  of plain-text plan mode. Use when presenting a plan, approach, or design the
  user will want to review and sign off on: author it as an HTML file, the user
  annotates and comments inline, and you iterate until they approve.
---

# plan-ui

Present a plan as a browser artifact the user can annotate, instead of prose.

## When to use

When you're about to propose a plan, approach, or design the user will review
before you implement — especially one they'll want to approve or change
step-by-step. For a quick plan they didn't ask to review, plain text is fine.

## Loop

1. Write the plan to an HTML file (`.plan-ui/<name>.html`). Run `plan-ui playbook`
   first and follow its authoring rules.
2. `plan-ui open <file>` — serves it and opens the browser.
3. `plan-ui poll <file>` — waits for feedback. It blocks silently; never kill it.
   Each comment tells you which part of the plan it refers to.
4. Edit the file (it live-reloads), then `plan-ui poll <file> --agent-reply
   "<what you changed>"` to reply and wait again.
5. `plan-ui end <file>` when the user approves.

Every command prints a `next_step` — follow it. If `plan-ui` isn't on `PATH`,
run the bundled `bootstrap.sh` once to install it.
