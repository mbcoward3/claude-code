# config-sync (skill)

A Claude Code skill that syncs your **Claude Code** and **Codex** config
(settings, skills, agents, commands, hooks, plugin state, prompts) with a
private git repo — manually, whenever you run it. No scripts, no schedulers:
the skill's instructions tell Claude what to copy, what to exclude, and how
to commit/push, and Claude executes it with git on whatever OS it's on
(Ubuntu, Windows, macOS).

## One-time setup

1. Create an empty **private** repo on GitHub (e.g. `my-agent-config`).
2. On your main machine, install the skill:

   ```bash
   mkdir -p ~/.claude/skills/config-sync
   cp SKILL.md ~/.claude/skills/config-sync/SKILL.md
   ```

3. In Claude Code, run `/config-sync` and give it the repo URL when asked.
   It clones the repo to `~/.config-sync` and does the first push.

## Every other machine

```bash
git clone git@github.com:YOU/my-agent-config.git ~/.config-sync
mkdir -p ~/.claude/skills
cp -r ~/.config-sync/claude/skills/config-sync ~/.claude/skills/
```

Then run `/config-sync pull` in Claude Code to install everything else.
Because the skill lives in `~/.claude/skills/` — which is itself synced — it
propagates to every machine and stays up to date.

## Usage

| Command | Effect |
|---|---|
| `/config-sync` | Two-way: push local changes, apply remote changes |
| `/config-sync push` | Local → repo only |
| `/config-sync pull` | Repo → local only (backs up anything it overwrites) |

Secrets (`auth.json`, `.credentials.json`, keys, `settings.local.json`) are
hard-excluded by the skill's instructions and never leave the machine.
Anything a pull overwrites is backed up to `~/.config-sync-backups/` first,
and every config change from every machine is in the repo's git history.
