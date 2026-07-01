Collaborate with me on a plan using plan-ui instead of writing it as prose.

Steps:
1. Run `python3 "$PLAN_UI/scripts/plan_ui.py" playbook` and follow it. (Set
   `$PLAN_UI` to this plugin's directory, or replace it with the absolute path.)
2. Write the plan as a self-contained HTML file, e.g. `.plan-ui/plan.html`.
3. Run `python3 "$PLAN_UI/scripts/plan_ui.py" open .plan-ui/plan.html` and share
   the printed URL with me.
4. Run `python3 "$PLAN_UI/scripts/plan_ui.py" poll .plan-ui/plan.html` and wait.
   It blocks silently until I annotate or comment — do not kill it. Each comment
   tells you which part of the plan it refers to.
5. Apply my feedback by editing the file (it live-reloads), then poll again with
   `--agent-reply "<what you changed>"`.
6. When I approve, run `python3 "$PLAN_UI/scripts/plan_ui.py" end .plan-ui/plan.html`.
