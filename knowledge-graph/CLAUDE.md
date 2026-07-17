---
type: Schema
title: Security Remediation Knowledge Graph — Schema & Operating Manual
description: The configuration file that makes the LLM a disciplined maintainer and triage assistant for this bundle.
tags: [schema, config, security]
timestamp: 2026-07-17T00:00:00Z
---

# Security Remediation Knowledge Graph — Schema & Operating Manual

An **LLM-maintained knowledge graph of recurring security-finding patterns and
the best-practice fix we've already agreed on for each**, distilled from a large
backlog of *resolved* GitLab issues from AI security scanning. Its job: when a
future agent meets a finding that **has precedent**, let it recognize the pattern
and **implement the established fix in the new location** — instead of re-deciding
a problem the team has already solved.

It follows two sources, mirrored under [`references/`](/references/):
the **[LLM Wiki pattern](/references/karpathy-llm-wiki.md)** (three layers: raw
sources → LLM-maintained wiki → this schema) and
**[OKF v0.1](/references/okf-spec-v0.1.md)** (the file format every concept
conforms to).

You (the LLM) own the wiki layer. The human curates sources and asks questions;
you do the reading, categorizing, cross-referencing, and bookkeeping.

---

## Content hygiene

This is internal engineering knowledge — solved patterns and their fixes. Keep
the pages about *patterns*, not raw incident data:

- **Destination is the team's private repo.** This scaffold is staged in a
  public leak-analysis repo only temporarily; move it to the private repository
  before adding real content. Do not treat this location as the permanent home.
- Record the **class** of finding and the **shape** of the fix — not a working
  exploit or reproduction recipe.
- Don't paste real secrets, tokens, credentials, keys, internal hostnames/IPs,
  or customer data into a page; they add nothing to a reusable pattern. Redact
  them.
- When ingesting a source that contains the above, summarize and redact; keep
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

- **Category** — a finding *pattern* (e.g. "SSRF in outbound fetchers", "missing
  authorization check on an endpoint"). This is where the precedent lives: how to
  recognize the pattern, the **agreed best-practice fix**, common false-positive
  signals, and the findings that established the precedent. **Categories emerge
  from the data** — create one when a pattern recurs; don't force findings into a
  rigid pre-set taxonomy. Where a standard mapping is obvious, record it in
  optional `cwe:` / `owasp:` frontmatter so external scanner findings self-map,
  but the primary structure is whatever actually reflects our backlog.
- **Resolution** — the **reusable best-practice fix** for a pattern, written so an
  agent can implement it in a new location. (A resolution may also be a
  compensating control, a false-positive rationale, or an accepted-risk
  rationale — precedents to *not* fix are precedents too.) Many findings and
  categories point at the same resolution.
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
1. Read it; redact per the content-hygiene note above.
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
In short: extract signals (CWE, rule ID, message, context) → check for
**precedent** by matching to a category → if found, **implement that category's
best-practice fix in the new finding's location** → if nothing matches, it's a
new pattern: propose a category. File the outcome back so the graph compounds.

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
