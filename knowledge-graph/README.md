---
type: Reference
title: Security Remediation Knowledge Graph — README
description: Human entry point explaining what this bundle is and how to use it.
tags: [readme]
timestamp: 2026-07-17T00:00:00Z
---

# Security Remediation Knowledge Graph

An **LLM-maintained knowledge graph of security findings and their resolutions**,
distilled from a backlog of GitLab issues produced by AI security scanning. The
goal: let a future agent take a *new* finding, **categorize it against what we've
already solved, and reach the right resolution quickly** — instead of re-triaging
the same vulnerability classes over and over.

Built as an [Open Knowledge Format v0.1](/references/okf-spec-v0.1.md) bundle
following [Karpathy's LLM Wiki pattern](/references/karpathy-llm-wiki.md).

## ⚠️ This is sensitive

The graph maps the org's known weaknesses; the *wontfix* set are live,
unmitigated issues. **It belongs in a private repo** and must contain **no real
secrets, hostnames, customer data, or working exploits** — only vulnerability
classes and remediation patterns. See the sensitivity rules in
[`CLAUDE.md`](/CLAUDE.md). The examples shipped here are synthetic.

## How to use it

1. **Read [`CLAUDE.md`](/CLAUDE.md)** — the operating manual for the maintaining
   agent.
2. **Triage a new finding** with [`triage.md`](/triage.md): extract signals →
   match a category → apply a known resolution → file the outcome back.
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
