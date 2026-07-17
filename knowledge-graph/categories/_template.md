---
type: Category
title: <Vulnerability class name>
description: <One sentence — the class of issue this page covers.>
tags: [category]
cwe: [<CWE-XXX>]           # optional cross-map; omit if not obvious
owasp: "<AXX:2021>"        # optional
status_summary: "<n fixed, n wontfix, n false-positive>"
timestamp: 2026-07-17T00:00:00Z
---

# What it is

<The vulnerability class in a few sentences: root cause and why it matters.>

# How to recognize it

Signals a triaging agent can match against a new finding:

* **CWE / rule IDs**: <e.g. CWE-798, scanner rule ids>
* **Message keywords**: <phrases scanners use>
* **Code/config shape**: <what the vulnerable pattern looks like>

# Common false-positive signals

* <Context in which this finding is usually NOT exploitable — so it can be
  dismissed quickly. Link the precedent: [decision](/decisions/<slug>.md).>

# Canonical resolutions

* [<Resolution>](/resolutions/<slug>.md) — <when to use it.>

# Landmark findings

* [<Finding>](/findings/<slug>.md) — <why it's the exemplar.>

# Member findings

| GitLab issue | Project | Severity | Status | Resolution |
|--------------|---------|----------|--------|------------|
| #<id>        | <proj>  | <sev>    | fixed  | [<res>](/resolutions/<slug>.md) |
