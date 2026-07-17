---
okf_version: "0.1"
---

# Security Remediation Knowledge Graph — Root Index

Catalog of every page. A graph of **solved finding patterns** from AI scanning
and the **best-practice fix** for each, so a future agent meeting a finding with
precedent can recognize it and implement the known fix in the new location. Read
[`CLAUDE.md`](/CLAUDE.md) for how it works, [`triage.md`](/triage.md) to check a
new finding for precedent, and [`log.md`](/log.md) for history.

> Status: **scaffolded, not yet populated.** Real content is added in the private
> destination repo (see the content-hygiene note in `CLAUDE.md`). The `_template.md`
> files show the expected shape of each concept type.

# Playbooks

* [Triage — classify a new finding](/triage.md) - the front door for a new scanner finding.

# Categories

One page per vulnerability class — the reusable core of the graph. _Empty — a
category is created when a finding pattern recurs._

* [Category (template)](/categories/_template.md) - shape of a category page.

# Resolutions

Reusable remediation patterns (fix / compensating-control / false-positive / accepted-risk). _Empty._

* [Resolution (template)](/resolutions/_template.md) - shape of a resolution page.

# Findings

Standalone pages for landmark issues only. _Empty — promote exemplars here._

* [Finding (template)](/findings/_template.md) - shape of a landmark finding page.

# Projects

Repos/services where findings arise. _Empty._

* [Project (template)](/projects/_template.md) - shape of a project page.

# Scanners

The scanning tools that produce findings. _Empty._

* [Scanner (template)](/scanners/_template.md) - shape of a scanner page.

# Decisions

Cross-cutting precedents & policy, especially wontfix / accepted-risk rationale.

* [Decision (template)](/decisions/_template.md) - shape of a decision record.

# References

* [Open Knowledge Format (OKF) v0.1](/references/okf-spec-v0.1.md) - the file format every concept conforms to.
* [LLM Wiki — Andrej Karpathy](/references/karpathy-llm-wiki.md) - the wiki-maintenance pattern this bundle instantiates.
