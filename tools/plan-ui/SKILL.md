---
name: plan-ui
description: >-
  Collaborate with the user on a plan in a rich, annotatable browser UI instead
  of plain-text plan mode. Use whenever presenting a plan, approach, or design
  the user will want to review and sign off on: author it as an HTML file, the
  user annotates and comments inline in the browser, and you iterate until they
  approve.
license: MIT
compatibility: Requires Python 3.8+ and a local browser to open the review UI. No network access or third-party packages needed.
metadata:
  author: mbcoward3
  version: "0.2.0"
---

# plan-ui

Present plans as annotatable HTML artifacts the user reviews in a browser,
instead of prose. The user marks up exact steps or lines; you get structured
feedback telling you precisely what to change.

## Running the CLI

If this skill is installed as a Claude Code plugin, `plan-ui` is on `PATH`.
Otherwise (e.g. as a Codex skill), invoke it directly from this skill's
directory:

```
python3 <this-skill-directory>/scripts/plan_ui.py <command> …
```

Every command below works either way. All commands print JSON with a
`next_step` field — follow it.

## Workflow

1. **Author the plan** as a self-contained HTML file, e.g.
   `.plan-ui/<name>.html`. First read
   [references/playbook.md](references/playbook.md) and follow its content,
   layout, and design rules.

2. **Open it** for review (serves the file, opens the browser):

   ```
   plan-ui open .plan-ui/<name>.html
   ```

3. **Wait for feedback**:

   ```
   plan-ui poll .plan-ui/<name>.html
   ```

   The poll blocks silently until the user acts — **never kill it**. It returns
   `prompts` (each with a `target` locating the exact element or quoted text it
   refers to), plus any `layout_warnings` from the automated layout gate.

4. **Apply the feedback** by editing the HTML file — the browser live-reloads.
   Use each prompt's `target` to make precise edits, not broad rewrites. Fix
   any `layout_warnings` immediately; the plan stays masked until the layout
   audit passes.

5. **Reply and wait again**:

   ```
   plan-ui poll .plan-ui/<name>.html --agent-reply "<what you changed>"
   ```

6. **Finish** when a prompt arrives with `"action": "approve"`, or the user
   says they're done:

   ```
   plan-ui end .plan-ui/<name>.html
   ```

## Reference

- [references/playbook.md](references/playbook.md) — plan authoring rules:
  content standards (verify claims, surface risks and open questions), layout
  rules the gate enforces, interactive `data-plan-action` buttons, and the
  design language for crisp, framework-free styling.
