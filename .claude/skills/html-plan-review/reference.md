# plan.html authoring reference

Read once per session, before writing your first plan. `plan.html` is a
raw HTML fragment (no `<html>`/`<head>`); the review page supplies all
styling and interactivity. See `examples/plan.html` for the house style
end-to-end.

## Hard rules

- Never include `<script>`, `<style>`, or event-handler attributes.
- Base tags: `p ul ol li pre code kbd table thead tbody tr td th strong
  em h3 h4 details summary a`.
- Multi-line, indented, human-diffable HTML — you will revise this file
  with targeted Edit calls, so keep lines short and stable.

## House style

Lead with a 1–2 sentence framing paragraph, then a stat-tile row giving
the plan's shape at a glance. Use a steps timeline for the approach,
tables for anything enumerable, tabs for genuine alternatives, callouts
sparingly (one risk, one win), `details` for depth that would bloat the
page. `h3` for top-level headings, `h4` inside components.

## Components

**Stat tiles** — the plan's shape at a glance:

    <div class="stats">
      <div class="stat"><div class="v">14</div><div class="l">files touched</div></div>
      <div class="stat"><div class="v">~2 wk</div><div class="l">to full rollout</div></div>
    </div>

**Steps timeline** — phases/approach; `<strong>` first child is the title line:

    <ol class="steps">
      <li><strong>Schema</strong>Add an identities table…</li>
      <li><strong>Rollout</strong>Canary to 5%…</li>
    </ol>

**Tables** — auto-styled (header band, row hover). Migrations, file
lists, API matrices.

**Tabs** — alternatives and trade-offs (pros/cons columns nest well):

    <div class="tabs">
      <section data-tab="Option A — cookies">…</section>
      <section data-tab="Option B — JWT">…</section>
    </div>

**Callouts** — `<div class="callout info|warn|risk|success">…</div>`
(icon added automatically).

**File tree** — `add` = new file, `mod` = modified:

    <ul class="file-tree">
      <li class="dir">src/auth/</li>
      <ul><li class="add">oauth.ts</li><li class="mod">session.ts</li></ul>
    </ul>

**Diff block** — `+`/`-`/`@@` lines colored automatically:

    <pre class="diff"><code>@@ config/auth.ts @@
    -  strategy: 'password',
    +  strategy: 'oauth',</code></pre>

**Pills** — `<span class="pill info|ok|warn|danger">label</span>` for
inline status (flag names, reversibility).

**Columns** — `<div class="cols"><div class="col">…</div>…</div>`.

## Sectioned plans (opt-in, rarely needed)

Free-form `plan.html` is the default. Only when the user explicitly
wants per-section flag/comment controls, put the body in `plan.json`
instead as `sections`:

    "sections": [
      { "id": "db", "heading": "1. Database changes", "bodyHtml": "<p>…</p>" }
    ]

Sections render as headings with hover tools (💬 / ⚑), not boxes.
`plan.json` may also carry `summaryHtml` (renders above everything;
annotations on it report `sectionId: "_summary"`). If `plan.html`
exists it renders as the `_doc` block alongside any sections.
