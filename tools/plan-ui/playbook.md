# plan-ui — plan authoring playbook

You are writing a **plan** as a self-contained HTML artifact that a human will
review and annotate in a browser. Optimize for fast, accurate review.

## Loop

1. Write the plan to an HTML file (e.g. `.plan-ui/<name>.html`).
2. `plan-ui open <file>` — serves it and opens the browser.
3. `plan-ui poll <file>` — blocks until the human annotates or messages. Never
   kill a running poll; it waits silently.
4. Apply the feedback to the file (it live-reloads).
5. `plan-ui poll <file> --agent-reply "<what you changed>"` — show your response
   and wait again.
6. `plan-ui end <file>` — when the human approves or is done.

## Content rules

- **Verify every claim against the codebase before stating it as fact.** A plan
  the human can't trust is worse than no plan.
- Lead with the decision or approach, not implementation minutiae.
- Include **risks, open questions, and failure modes** explicitly — these are
  what review is for.
- Make tradeoffs visible: show the cost of each option as clearly as the benefit.
- Break the plan into steps the human can approve or reject individually.

## Layout rules (the gate enforces these)

- **No horizontal overflow at any nesting level.** The layout gate masks the
  plan until this passes, and reports failures back to you.
- Use relative asset paths (no leading `/`) so the file stays portable.
- The artifact must render identically opened directly in a browser or through
  plan-ui — the SDK is injected at serve time, not saved into your file.

## Interactive actions

Add one-click actions the human can trigger by putting `data-plan-action` on an
element:

    <button data-plan-action="approve" data-plan-label="Looks good, proceed">
      Approve
    </button>

The action id and label arrive in your next `poll` as a prompt.

## Design

Tailwind (v4, JIT) and DaisyUI v5 are injected locally — use their utility
classes freely. No CDN is required or available; everything renders offline.
