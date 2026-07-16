---
type: Reference
title: Knowledge Graph — README
description: Human entry point explaining what this bundle is and how to use it.
tags: [readme]
timestamp: 2026-07-16T00:00:00Z
---

# Knowledge Graph

An **LLM-maintained knowledge graph** for a significant development effort,
built as an [Open Knowledge Format v0.1](/references/okf-spec-v0.1.md) bundle
following [Karpathy's LLM Wiki pattern](/references/karpathy-llm-wiki.md).

The knowledge here is a compounding artifact: the LLM reads sources once and
integrates them into interlinked pages, so cross-references and synthesis are
built up rather than re-derived on every question.

## How to use it

1. **Open [`CLAUDE.md`](/CLAUDE.md)** — it's the operating manual. Any LLM agent
   working in this directory should read it first.
2. **Curate sources.** Drop founding documents (spec, RFC, kickoff notes) into
   `sources/` and ask the agent to *ingest* them.
3. **Ask questions.** Query the graph; good answers get filed back as pages.
4. **Browse the graph.** Every page is plain markdown with bundle-relative
   links — open it in Obsidian (use the graph view) or read it on GitHub.

## Layout

| Path          | What lives there                                        |
|---------------|---------------------------------------------------------|
| `CLAUDE.md`   | Schema + operating manual for the maintaining agent.    |
| `index.md`    | Catalog of every page, grouped by category.             |
| `log.md`      | Append-only chronological history.                      |
| `references/` | The OKF spec and LLM-wiki idea file.                    |
| `sources/`    | One summary concept per ingested source.                |
| `entities/`   | People, teams, systems, services, components.           |
| `concepts/`   | Architecture, requirements, risks, open questions.      |
| `decisions/`  | Decision records.                                       |

`_template.md` files show the expected shape of each concept type.
