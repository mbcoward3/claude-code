---
type: Schema
title: Knowledge Graph — Schema & Operating Manual
description: The configuration file that makes the LLM a disciplined maintainer of this bundle.
tags: [schema, config]
timestamp: 2026-07-16T00:00:00Z
---

# Knowledge Graph — Schema & Operating Manual

This directory is an **LLM-maintained knowledge graph** for a significant
development effort. It follows two sources, both mirrored under
[`references/`](/references/):

- The **[LLM Wiki pattern](/references/karpathy-llm-wiki.md)** (Karpathy) — the
  three-layer idea: immutable raw sources, an LLM-maintained wiki, and this
  schema file.
- The **[Open Knowledge Format v0.1](/references/okf-spec-v0.1.md)** (OKF) — the
  file format every concept in this bundle conforms to.

You (the LLM) own the wiki layer entirely. The human curates sources, directs
the analysis, and asks questions. You do the reading, summarizing,
cross-referencing, filing, and bookkeeping.

---

## Bundle layout

```
knowledge-graph/
├── CLAUDE.md          # This file. The schema. Read it first.
├── index.md           # Root catalog of every page. Read second.
├── log.md             # Append-only chronological history.
├── references/        # The OKF spec + LLM-wiki idea file (immutable).
├── sources/           # Ingested raw material, one summary concept per source.
├── entities/          # People, teams, systems, services, components.
├── concepts/          # Abstract topics: architecture, requirements, risks.
└── decisions/         # Decision records (what was chosen and why).
```

Every `.md` file except `index.md` and `log.md` is an **OKF concept**: a YAML
frontmatter block followed by a markdown body.

## Frontmatter contract (OKF §4.1)

Required on every concept:

```yaml
---
type: <Concept type>          # REQUIRED. e.g. Source, Entity, Concept, Decision, Reference
title: <Human-readable name>
description: <One-sentence summary>   # used by index.md
tags: [<tag>, <tag>]
timestamp: <ISO 8601>          # last meaningful change
---
```

`type` is the only field OKF *requires*. Pick descriptive values. Types in use
here: `Source`, `Entity`, `Concept`, `Decision`, `Reference`, `Requirement`,
`Risk`. Add more as the domain needs them — no registry, no approval.

## Cross-linking (OKF §5)

- Link concepts with **bundle-relative absolute** links: `[orders](/tables/orders.md)`.
  These stay valid when files move within a subdirectory.
- A link is an untyped directed edge; the *kind* of relationship lives in the
  surrounding prose, not the link.
- Broken links are allowed — they represent not-yet-written knowledge. Prefer
  linking to a page that doesn't exist yet over not linking at all; it becomes a
  visible to-do in the graph.

---

## Operations (from the LLM Wiki pattern)

### Ingest
When the human drops a source into `sources/` (or points you at a URL/doc):
1. Read it fully.
2. Discuss the key takeaways with the human.
3. Write a summary concept in `sources/` (`type: Source`, cite the original in a
   `# Citations` section).
4. Update or create the entity/concept/decision pages it touches — a single
   source often touches 5–15 pages.
5. Update [`index.md`](/index.md).
6. Append an entry to [`log.md`](/log.md).

### Query
1. Read [`index.md`](/index.md) first to find candidate pages.
2. Read those pages, synthesize an answer **with citations** (link the pages).
3. **File good answers back** as new concept pages — a comparison, an analysis,
   a discovered connection shouldn't vanish into chat. Log it.

### Lint
Periodically health-check the graph. Look for: contradictions between pages,
stale claims newer sources have superseded, orphan pages with no inbound links,
important concepts mentioned but lacking their own page, missing cross-links,
and gaps a search could fill. Report findings and propose fixes; log the pass.

---

## Conventions

- **index.md** is content-oriented — a catalog grouped by category, each entry a
  link + one-line description (pulled from the concept's `description`). Update
  on every ingest.
- **log.md** is chronological and append-only. Start each entry with a
  consistent prefix so it stays greppable:
  `## [YYYY-MM-DD] <op> | <title>` where `<op>` ∈ `ingest | query | lint | init`.
  `grep "^## \[" log.md | tail -5` then gives the recent history.
- Favor **structural markdown** — headings, lists, tables — over prose, for both
  human reading and agent retrieval (OKF §4.2).
- The whole bundle is a git repo of markdown. Commit meaningful updates so the
  history tells the story of the effort.

## This bundle's purpose

Scaffolded 2026-07-16 to support a significant development effort starting the
following week. The directory structure is ready; the content is not yet
populated. First real work: ingest the effort's founding documents (spec, RFC,
kickoff notes) into `sources/`, stand up the core `entities/` and `concepts/`
pages, and record early architectural choices in `decisions/`. See
[`index.md`](/index.md) for current contents and the placeholder pages for the
expected shape of each type.
