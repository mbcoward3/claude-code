# Update Log

Append-only, newest first. Each entry: `## [YYYY-MM-DD] <op> | <title>`
where `<op>` ∈ `ingest | query | lint | init`.

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
