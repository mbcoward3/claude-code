# Update Log

Append-only, newest first. Each entry: `## [YYYY-MM-DD] <op> | <title>`
where `<op>` ∈ `ingest | triage | lint | init`.

## [2026-07-17] init | Removed synthetic example pages

* **Update**: Dropped the illustrative `hardcoded-secrets` category and
  `externalize-secret` resolution; the `_template.md` files carry the shape.
  Categories and resolutions now start empty, to be built from the real backlog.

## [2026-07-17] init | Retargeted bundle to security-remediation domain

* **Update**: Reworked the bundle for its real purpose — a knowledge graph of AI
  security-scanning findings and their resolutions, to speed triage of new
  findings.
* **Creation**: Domain structure — `categories/`, `resolutions/`, `findings/`,
  `projects/`, `scanners/`; retooled `decisions/`. Removed the generic
  `concepts/`, `entities/`, `sources/` scaffolding.
* **Creation**: [`triage.md`](/triage.md) — the classification front door.
* **Creation**: Synthetic worked example — the
  [Hardcoded Secrets](/categories/hardcoded-secrets.md) category and
  [Externalize Secret](/resolutions/externalize-secret.md) resolution.
* **Update**: [`CLAUDE.md`](/CLAUDE.md) with sensitivity rules, hybrid
  granularity, emergent taxonomy, and the ingest/triage/lint operations.
* **Decision**: Destined for a **private** repo; no real secrets/hostnames/
  exploits committed here.
* **Next**: Provide GitLab export or API access; ingest the backlog.

## [2026-07-16] init | Bundle scaffolded

* **Initialization**: Created the OKF v0.1 bundle structure — `references/`,
  `sources/`, `entities/`, `concepts/`, `decisions/`.
* **Creation**: Mirrored the [OKF spec](/references/okf-spec-v0.1.md) and the
  [LLM Wiki idea file](/references/karpathy-llm-wiki.md) as reference concepts.
* **Creation**: Wrote the schema/operating manual [`CLAUDE.md`](/CLAUDE.md) and
  the root [`index.md`](/index.md).
* **Creation**: Added `_template.md` placeholders showing the expected shape of
  each concept type.
* **Next**: Ingest the development effort's founding documents into `sources/`.
