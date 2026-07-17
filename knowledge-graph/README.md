---
type: Reference
title: Security Remediation Knowledge Graph — README
description: Human entry point explaining what this bundle is and how to use it.
tags: [readme]
timestamp: 2026-07-17T00:00:00Z
---

# Security Remediation Knowledge Graph

An **LLM-maintained knowledge graph of solved security-finding patterns and the
best-practice fix for each**, distilled from a backlog of resolved GitLab issues
from AI security scanning. The goal: when a future agent meets a finding that
**has precedent**, let it recognize the pattern and **implement the established
fix in the new location** — instead of re-deciding a problem the team has already
solved.

Built as an [Open Knowledge Format v0.1](/references/okf-spec-v0.1.md) bundle
following [Karpathy's LLM Wiki pattern](/references/karpathy-llm-wiki.md).

## What this is (and isn't)

A library of **solved patterns** — recurring findings and the agreed best-practice
fix for each — kept in the team's private repo. It is *not* a live vulnerability
inventory: pages describe the class of finding and the shape of the fix, with no
real secrets, hostnames, customer data, or working exploits. See the
content-hygiene note in [`CLAUDE.md`](/CLAUDE.md).

## How to use it

1. **Read [`CLAUDE.md`](/CLAUDE.md)** — the operating manual for the maintaining
   agent.
2. **Triage a new finding** with [`triage.md`](/triage.md): extract signals →
   match a precedent category → implement its best-practice fix in the new
   location → file the outcome back.
3. **Ingest the backlog**: feed GitLab issues in and let the agent build out
   categories, resolutions, and precedents.
4. **Browse** in Obsidian (graph view shows which classes are hubs) or on your
   git host.

## Layout

| Path           | What lives there                                             |
|----------------|--------------------------------------------------------------|
| `CLAUDE.md`    | Schema + operating manual.                                   |
| `triage.md`    | Classification procedure for a new finding.                  |
| `index.md`     | Catalog of every page.                                       |
| `log.md`       | Append-only history.                                         |
| `categories/`  | One page per vulnerability class — the reusable core.        |
| `resolutions/` | Reusable fixes / controls / false-positive & risk rationale. |
| `findings/`    | Standalone pages for landmark issues.                        |
| `projects/`    | Repos/services where findings arise.                         |
| `scanners/`    | The scanning tools that produce findings.                    |
| `decisions/`   | Cross-cutting precedents & policy.                           |
| `references/`  | The OKF spec and LLM-wiki idea file.                         |

`_template.md` files show the expected shape of each concept type.

## Getting the backlog in

Export from GitLab per issue: title, description, labels, state, **closing
comment/resolution**, and close reason. Provide either API access (a token) or a
JSON/CSV export dropped where the agent can read it — that determines whether the
agent runs an ingest script or parses a file.
