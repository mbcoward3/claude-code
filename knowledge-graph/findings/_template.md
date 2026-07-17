---
type: Finding
title: <Short finding title>
description: <One sentence — what this landmark issue was.>
status: fixed            # fixed | wontfix | false-positive | accepted-risk
severity: high           # critical | high | medium | low | info
scanner: <scanner name>
project: <project name>
gitlab_issue: "<url or #id>"
tags: [finding]
timestamp: 2026-07-17T00:00:00Z
---

> Only create a standalone finding page for **landmark** issues — the canonical
> exemplar of a class, an unusually instructive fix, or a heavily-cited
> precedent. Ordinary findings live as rows in their
> [category](/categories/) page.

# What was found

<Redacted description of the issue — class and shape, not a working exploit.>

# Classification

* Category: [<category>](/categories/<slug>.md)
* Why it's landmark: <what makes this the reference example.>

# How it was resolved

* Resolution: [<resolution>](/resolutions/<slug>.md)
* Outcome: <what changed; link the decision if wontfix/accepted-risk.>

# Citations

[1] [GitLab issue](<url>)
