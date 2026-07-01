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

**No frameworks are injected or available — no Tailwind, no CDN.** Put all
styling in one `<style>` block in your `<head>` so the file is fully
self-contained and renders identically anywhere.

Follow this design language so every plan looks crisp and consistent:

- **Fonts:** `font-family: ui-sans-serif, system-ui, -apple-system, sans-serif;`
  code: `ui-monospace, 'SF Mono', Menlo, Consolas, monospace`.
- **Page:** content in a centered column, `max-width: 820px`, generous padding
  (`48px 32px`), `line-height: 1.6`, base size `16px`.
- **Palette:** near-black text `#1c2230` on white; muted `#55606f`; borders
  `#e6e8ec`; one accent `#e0771b` used sparingly (links, highlights, key
  numbers); status colors — good `#2f8f4e`, risk `#b54708`, danger `#b42318`.
- **Spacing:** stick to multiples of 8px. Separate sections with whitespace and
  a light `border-top`, not heavy boxes.
- **Structure:** number the steps; put risks / open questions in visually
  distinct callouts (left border + tinted background, e.g.
  `border-left: 3px solid #b54708; background: #fff8f1; padding: 12px 16px;`).
- **Tables** for comparisons: minimal — `border-collapse: collapse`, a bottom
  border per row, no vertical rules, header in muted small caps.
- **Code:** dark blocks (`background: #0e1017; color: #e8ecf2; border-radius: 8px;
  padding: 14px 16px; overflow-x: auto;`).
- Prefer clean typography and whitespace over decoration. No images or icon
  fonts; if you need a small diagram, draw it with inline SVG.
