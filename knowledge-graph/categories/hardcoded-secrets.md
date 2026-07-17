---
type: Category
title: Hardcoded Secrets
description: Credentials, tokens, or keys committed directly into source or config instead of being injected at runtime.
tags: [category, secrets, credentials]
cwe: [CWE-798, CWE-259]
owasp: "A07:2021"
status_summary: "example page — replace counts with real backlog data"
timestamp: 2026-07-17T00:00:00Z
---

> **Illustrative example** using synthetic data, to show the intended shape of a
> category page. Replace with real backlog content in the private repo.

# What it is

A secret (API key, password, private key, connection string) is embedded
literally in code, config, or CI files. It leaks to anyone with repo read access
and persists in git history even after removal, so rotation — not just deletion —
is required.

# How to recognize it

* **CWE / rule IDs**: CWE-798, CWE-259; secret-scanning rules (e.g. generic
  high-entropy string, known token prefixes like `AKIA…`, `ghp_…`).
* **Message keywords**: "hardcoded", "secret", "credential", "private key",
  "high entropy string".
* **Code/config shape**: a literal string assigned to a name like `password`,
  `api_key`, `token`, `secret` in source or `.env`/`.yml` checked into the repo.

# Common false-positive signals

* The value is an obvious placeholder (`changeme`, `xxxx`, `example`) or a
  documented test fixture. Dismiss as false-positive; see the precedent under
  [decisions](/decisions/) once one exists.

# Canonical resolutions

* [Externalize secret to a secrets manager](/resolutions/externalize-secret.md)
  — the standard fix: remove from source, inject at runtime, **rotate** the
  exposed value.

# Landmark findings

* _(none yet — promote the most instructive real issue here)_

# Member findings

| GitLab issue | Project     | Severity | Status | Resolution |
|--------------|-------------|----------|--------|------------|
| #EXAMPLE-1   | example-svc | high     | fixed  | [Externalize secret](/resolutions/externalize-secret.md) |
