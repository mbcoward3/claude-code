---
name: plan-ui
description: >-
  Collaborate with the user on a plan in a rich, annotatable browser UI instead
  of plain-text plan mode. Use when presenting a plan, approach, or design the
  user will want to review and sign off on: author it as an HTML file, the user
  annotates and comments inline, and you iterate until they approve.
---

# plan-ui

When you're about to present a plan, approach, or design the user will review
before you implement, collaborate on it with plan-ui instead of prose — author
it as a rich HTML artifact the user can annotate inline.

Run `plan-ui playbook` and follow it — the playbook has the full loop and the
authoring rules. (`plan-ui` is on `PATH` while this plugin is enabled.)

Note: in Claude Code, when you finish native plan mode the plugin also opens a
plan-ui review automatically via the ExitPlanMode hook — no command needed. Use
the `plan-ui` commands above when you want the richer HTML artifact instead of
the plain plan-mode text.
