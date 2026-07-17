---
okf_version: "0.1"
---

# Security Remediation Knowledge Graph — Root Index

Catalog of every page. A graph of security **findings** from AI scanning and the
**resolutions** we've already applied, so future agents can categorize new
findings fast. Read [`CLAUDE.md`](/CLAUDE.md) for how it works,
[`triage.md`](/triage.md) to classify a new finding, and [`log.md`](/log.md) for
history.

> Status: **scaffolded with a synthetic worked example.** Real content is added
> in the private destination repo (see the sensitivity rules in `CLAUDE.md`). The
> `hardcoded-secrets` category and `externalize-secret` resolution are
> illustrative only.

# Playbooks

* [Triage — classify a new finding](/triage.md) - the front door for a new scanner finding.

# Categories

One page per vulnerability class — the reusable core of the graph.

* [Hardcoded Secrets](/categories/hardcoded-secrets.md) - credentials committed into source instead of injected at runtime. _(example)_
* [Category (template)](/categories/_template.md) - shape of a category page.

# Resolutions

Reusable remediation patterns (fix / compensating-control / false-positive / accepted-risk).

* [Externalize Secret to a Secrets Manager](/resolutions/externalize-secret.md) - remove, inject at runtime, rotate. _(example)_
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
