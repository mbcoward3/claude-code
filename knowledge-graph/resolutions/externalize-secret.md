---
type: Resolution
title: Externalize Secret to a Secrets Manager
description: Remove a hardcoded secret from source, inject it at runtime, and rotate the exposed value.
resolution_kind: fix
tags: [resolution, secrets]
timestamp: 2026-07-17T00:00:00Z
---

> **Illustrative example** using synthetic data, to show the intended shape of a
> resolution page.

# When to use

A [hardcoded secret](/categories/hardcoded-secrets.md) is confirmed real (not a
placeholder or test fixture).

# Steps

1. **Rotate first.** Treat the committed value as compromised — issue a new
   credential and revoke the old one before anything else. It lives in git
   history regardless of what you do to the working tree.
2. Move the value into the runtime secrets mechanism (secrets manager / CI
   variable / injected env var), referenced by name, not literal.
3. Replace the literal in code/config with a lookup of that named reference.
4. Purge from history only if policy requires it; rotation is what actually
   closes the exposure.

# Verification

* Re-run the secret scanner on the branch — the finding clears.
* Confirm the app reads the secret from the injected source in a non-prod deploy.

# Notes / caveats

* Rotation is mandatory; deletion alone leaves the secret recoverable from
  history. If rotation is impossible short-term, this becomes a
  [compensating-control](/decisions/) situation instead.
