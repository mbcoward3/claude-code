---
type: Schema
title: Security Remediation Knowledge Graph — Schema & Operating Manual
description: The configuration file that makes the LLM a disciplined maintainer and triage assistant for this bundle.
tags: [schema, config, security]
timestamp: 2026-07-17T00:00:00Z
---

# Security Remediation Knowledge Graph — Schema & Operating Manual

An **LLM-maintained knowledge graph of security findings and how we resolved
them**, distilled from a large backlog of GitLab issues produced by AI security
scanning. Its job is to let a future agent take a *new* finding, **categorize it
against what we already know, and reach the right resolution quickly** — instead
of re-triaging the same vulnerability classes from scratch every time.

It follows two sources, mirrored under [`references/`](/references/):
the **[LLM Wiki pattern](/references/karpathy-llm-wiki.md)** (three layers: raw
sources → LLM-maintained wiki → this schema) and
**[OKF v0.1](/references/okf-spec-v0.1.md)** (the file format every concept
conforms to).

You (the LLM) own the wiki layer. The human curates sources and asks questions;
you do the reading, categorizing, cross-referencing, and bookkeeping.

---

## ⚠️ Sensitivity rules — read before writing anything

This graph is a map of the organization's known weaknesses. The
**"decided against resolving" (wontfix) set are live, unmitigated issues.**
Treat the whole bundle as sensitive.

- **Destination is a private repo.** This scaffold is staged in a public
  leak-analysis repo only temporarily; it must be moved to a dedicated private
  repository before real finding detail is added. Do not treat this location as
  the permanent home.
- **Never commit** real secrets, tokens, credentials, private keys, internal
  hostnames/IPs, customer data, or working exploit steps. Redact them. Describe
  the *class* of problem and the *shape* of the fix, not a reproduction recipe.
- Prefer generic, redacted descriptions over pasting real scanner output
  verbatim.
- If a source you're ingesting contains the above, summarize and redact; keep
  the raw material out of the bundle.

---

## Bundle layout

```
knowledge-graph/
├── CLAUDE.md          # This file — the schema. Read first.
├── index.md           # Catalog of every page. Read second.
├── triage.md          # The front door for classifying a NEW finding.
├── log.md             # Append-only chronological history.
├── references/        # OKF spec + LLM-wiki idea file (immutable).
├── categories/        # One page per vulnerability class. The reusable core.
├── resolutions/       # Reusable remediation patterns (fix / control / FP / risk).
├── findings/          # Landmark individual issues worth a deep-dive page.
├── projects/          # The repos/services where findings arise.
├── scanners/          # The scanning tools that produce findings.
└── decisions/         # Cross-cutting precedents & policy (esp. wontfix rationale).
```

## The domain model (how the graph is shaped)

The three most important node types and how they relate:

- **Category** — a vulnerability *class* (e.g. "hardcoded secrets", "SSRF in
  outbound fetchers"). This is where reusable knowledge lives: how to recognize
  it, common false-positive signals, the canonical resolution(s), and a rolled-up
  list of member findings. **Categories emerge from the data** — create one when
  a pattern recurs; don't force findings into a rigid pre-set taxonomy. Where a
  standard mapping is obvious, record it in optional `cwe:` / `owasp:`
  frontmatter so external scanner findings self-map, but the primary structure is
  whatever actually reflects our backlog.
- **Resolution** — a *reusable* way we've addressed a class: a fix pattern, a
  compensating control, a false-positive rationale, or an accepted-risk
  rationale. Many findings and categories point at the same resolution.
- **Finding** — an individual GitLab issue. **Hybrid granularity:** most findings
  live as *rows in their category page*, not as their own file. Only **landmark**
  findings — the canonical exemplar of a class, an unusually instructive fix, or
  a heavily-cited precedent — get a standalone page under `findings/`.

`projects/` and `scanners/` are supporting entities so findings can be traced to
where they arose and which tool flagged them (useful for tracking scanner
false-positive rates).

## Frontmatter contract (OKF §4.1)

Required on every concept: a non-empty `type`. Recommended: `title`,
`description`, `tags`, `timestamp`. Domain-specific optional fields by type:

```yaml
# Category
type: Category
cwe: [CWE-798]                 # optional cross-map
owasp: "A07:2021"              # optional
status_summary: "12 fixed, 3 wontfix, 4 false-positive"

# Resolution
type: Resolution
resolution_kind: fix           # fix | compensating-control | false-positive | accepted-risk

# Finding (landmark)
type: Finding
status: fixed                  # fixed | wontfix | false-positive | accepted-risk
severity: high                 # critical | high | medium | low | info
scanner: <name>
project: <name>
gitlab_issue: "<url or #id>"   # citation to the original
```

Types in use: `Category`, `Resolution`, `Finding`, `Project`, `Scanner`,
`Decision`, `Reference`, `Playbook`, `Schema`.

## Cross-linking (OKF §5)

Use **bundle-relative absolute** links: `[<category>](/categories/<slug>.md)`.
A finding links to its category and its resolution; a category links to its
resolutions, landmark findings, and any decision precedent. Broken links are
allowed and represent not-yet-written knowledge.

---

## Operations

### Ingest (backlog → graph)
Given a GitLab issue (or a batch export):
1. Read it; redact anything sensitive per the rules above.
2. Determine its vulnerability class. Match to an existing
   [category](/categories/) or create a new one if the pattern is new.
3. Capture how it was resolved. Reuse an existing [resolution](/resolutions/) or
   write a new reusable one. Record the close reason (fixed / wontfix /
   false-positive / accepted-risk).
4. Record the finding: a **row** in the category's finding table, or — if
   landmark — a standalone [`findings/`](/findings/) page.
5. Update the category's `status_summary`, touch related projects/scanners.
6. Update [`index.md`](/index.md); append to [`log.md`](/log.md).

### Triage (the key query — a NEW finding comes in)
This is what the graph exists for. Full procedure in [`triage.md`](/triage.md).
In short: extract signals (CWE, rule ID, message, context) → match to a category
→ apply its canonical resolution and check prior precedents → if nothing matches,
propose a new category. File the outcome back so the graph compounds.

### Lint
Periodically health-check: categories with no resolution, resolutions no finding
uses, contradictory precedents, stale fixes newer guidance supersedes, orphan
pages, near-duplicate categories that should merge, and scanners with high
false-positive rates worth flagging. Report and propose fixes; log the pass.

## Conventions

- **index.md** — catalog grouped by category, each entry a link + one-line
  description. Update on every ingest.
- **log.md** — append-only, newest first. Prefix each entry
  `## [YYYY-MM-DD] <op> | <title>`, `<op>` ∈ `ingest | triage | lint | init`, so
  `grep "^## \[" log.md | tail -5` shows recent history.
- Favor **structural markdown** — tables, lists, headings — over prose.
- Commit meaningful updates; the git history is the story of the effort.
