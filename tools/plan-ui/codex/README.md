# plan-ui for Codex

Codex has no plugin/hook system like Claude Code, so it uses the **artifact loop**
directly. There are two ways to wire it up.

## Option A: custom prompt

Copy `plan-ui.md` into your Codex prompts directory so it's available as a
slash-command-style prompt:

```
cp codex/plan-ui.md ~/.codex/prompts/plan-ui.md
```

Then invoke it and Codex will follow the plan-ui loop.

## Option B: always-on instruction

Add the snippet below to your project `AGENTS.md` so Codex reaches for plan-ui
whenever it presents a plan:

```markdown
## Planning

When presenting a plan, approach, or design for review, use plan-ui instead of
plain prose: author the plan as an HTML file, then run
`python3 /ABS/PATH/tools/plan-ui/scripts/plan_ui.py open <file>` and follow the
`next_step` in each JSON response. Run the `playbook` subcommand first for the
authoring rules.
```

Replace `/ABS/PATH` with this plugin's location. Unlike Claude Code, Codex does
not add `bin/` to `PATH`, so call `scripts/plan_ui.py` directly (or symlink
`bin/plan-ui` onto your `PATH`).
