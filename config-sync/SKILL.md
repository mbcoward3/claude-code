---
name: config-sync
description: Sync Claude Code and Codex configuration (settings, skills, agents, commands, plugins, prompts) with the user's private git config repo. Use when the user runs /config-sync or asks to sync, push, pull, or back up their Claude/Codex config or settings.
---

# config-sync

Sync the user's Claude Code and Codex configuration with their private git
config repo, using a clone at `~/.config-sync`. Manually triggered only —
never set up cron jobs, scheduled tasks, or hooks for this.

**Direction** (from the argument the user passed):
- `push` — local → repo only
- `pull` — repo → local only
- no argument — both: capture local changes, merge remote, apply, push back

## Repo location

The clone lives at `~/.config-sync`. If it doesn't exist, ask the user for
their config repo's git URL, then `git clone <url> ~/.config-sync`. If the
clone is empty (brand-new repo), treat this as the first push: create the
layout below from local files, commit, and push.

## What syncs

| Local (in `~`) | In repo |
|---|---|
| `.claude/settings.json` | `claude/settings.json` |
| `.claude/keybindings.json` | `claude/keybindings.json` |
| `.claude/CLAUDE.md` | `claude/CLAUDE.md` |
| `.claude/skills/` | `claude/skills/` |
| `.claude/agents/` | `claude/agents/` |
| `.claude/commands/` | `claude/commands/` |
| `.claude/hooks/` | `claude/hooks/` |
| `.claude/output-styles/` | `claude/output-styles/` |
| `.claude/plugins/config.json` | `claude/plugins/config.json` |
| `.codex/config.toml` | `codex/config.toml` |
| `.codex/AGENTS.md` | `codex/AGENTS.md` |
| `.codex/prompts/` | `codex/prompts/` |

Directories are mirrored: a file deleted on one side gets deleted on the
other when syncing that direction. Paths that don't exist locally (e.g. no
`~/.codex` on this machine) are skipped — never delete the repo copy because
a tool isn't installed here.

## Never sync these

Hard rule, even if they appear inside a synced directory, even if asked:
`auth.json`, `.credentials.json`, `credentials.json`, `settings.local.json`,
`*.pem`, `*.key`, `*.p12`, `id_rsa*`, `id_ed25519*`, anything that looks like
an API key or token, plus junk (`.DS_Store`, `__pycache__`, `node_modules`,
`*.log`). If one of these is already tracked in the repo, warn the user
instead of updating it.

## Procedure

1. `git -C ~/.config-sync pull --rebase --autostash`. If it conflicts, abort
   the rebase, show the conflicting files, and ask the user which side wins —
   do not guess.
2. **Pull direction:** copy repo → local for every mapping above. Before
   overwriting or deleting a local file, copy the old version to
   `~/.config-sync-backups/<YYYYMMDD-HHMMSS>/` (mirror the relative path).
   Tell the user where the backup is if anything was overwritten.
3. **Push direction:** copy local → repo working tree for every mapping,
   applying the exclusion list. Then, if `git status --porcelain` shows
   changes: `git add -A`, commit as
   `sync from <hostname> <YYYY-MM-DD HH:MM>`, and `git push`. Retry a failed
   push once after `git pull --rebase`.
4. Finish with a short summary: files updated in each direction (or "already
   in sync"), and the commit hash if one was made.

Use whatever shell fits the platform (bash on Linux/macOS, PowerShell on
Windows) — paths above are home-relative on both.

## First time on a new machine

If `~/.config-sync` doesn't exist yet, after cloning just run the pull
direction. That installs the skills directory too — including this skill —
so future syncs are `/config-sync`.
