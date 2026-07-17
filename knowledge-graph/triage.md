---
type: Playbook
title: Triage — Classify a New Security Finding
description: Step-by-step procedure for an agent to categorize a new finding against prior knowledge and reach a resolution.
tags: [triage, playbook, classification]
timestamp: 2026-07-17T00:00:00Z
---

# Triage: does this finding have precedent — and can we reuse the fix?

This is the front door when a new scanner finding arrives. Goal: check whether
the finding matches a **pattern we've already solved**, and if so, **implement
that established best-practice fix in the new location** with minimal re-analysis.
Read [`CLAUDE.md`](/CLAUDE.md) for the model.

## 1. Extract signals

Pull whatever the finding gives you:

* **CWE ID / rule ID** — strongest signal; often maps a category directly.
* **Scanner + rule name** — see [`scanners/`](/scanners/) for known tools.
* **Message text** — keywords to match against category recognition signals.
* **Location** — file/path/service; which [project](/projects/) it's in.
* **Severity** as reported.

## 2. Match to a category

In priority order:

1. **By CWE/OWASP** — scan category frontmatter (`cwe:`, `owasp:`) for a match.
2. **By recognition signals** — each category page has a "How to recognize it"
   section; compare against the finding.
3. **By keyword** — search category titles/bodies; fall back to full-text search
   over the bundle if one is set up.

Read [`index.md`](/index.md) first to see the candidate set.

## 3. If a category matches

1. Read the category page.
2. Check its **member findings** for a near-duplicate — same root cause, same
   project, or same code path. A duplicate usually inherits the same resolution.
3. Check for a **decision precedent** (linked from the category) — especially
   *accepted-risk* / *false-positive* rulings, so you don't re-litigate a call
   the team already made.
4. **Implement the category's [best-practice fix](/resolutions/) in the new
   finding's location.** Confirm it still fits (not superseded by newer guidance —
   check the timestamp) and adapt it to the new codebase/context.
5. Record the outcome (step 5 below).

## 4. If nothing matches

No precedent yet — this pattern hasn't been solved before. That's expected; the
taxonomy is emergent. This is where a new best practice gets established.

1. Confirm it isn't a near-miss of an existing category that should just be
   broadened.
2. Create a new [category](/categories/) page (copy `_template.md`). Fill in
   recognition signals and, if obvious, the `cwe:`/`owasp:` cross-map.
3. Once the team settles on the fix, capture it as a [resolution](/resolutions/)
   so it becomes the precedent.
4. Record the outcome and **log it** — the next similar finding now self-serves.

## 5. Record the outcome (so the graph compounds)

* Add the finding as a **row** in the category's finding table, or a standalone
  [`findings/`](/findings/) page if it's landmark.
* Set `status`: `fixed | wontfix | false-positive | accepted-risk`, and cite the
  GitLab issue.
* Update the category's `status_summary`.
* Append a `## [YYYY-MM-DD] triage | <finding>` entry to [`log.md`](/log.md).

## Resolution decision — quick reference

| Situation | Resolution kind | Where it's recorded |
|-----------|-----------------|---------------------|
| Real issue, code/config change fixes it | `fix` | resolution page + finding row |
| Real issue, can't fix now, mitigate around it | `compensating-control` | resolution + [decision](/decisions/) |
| Scanner is wrong / not exploitable in context | `false-positive` | resolution + precedent so it's auto-dismissed next time |
| Real but accepted (low risk / cost) | `accepted-risk` | [decision](/decisions/) with rationale + owner + review date |
