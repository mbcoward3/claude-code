# config-sync

Keep your **Claude Code** and **Codex** configuration (settings, skills,
agents, commands, hooks, plugin state, prompts) synced across machines
through a remote git repo.

- **One script, no dependencies** — a single Python 3.8+ stdlib file
  ([`config_sync.py`](config_sync.py)). Works on Ubuntu/Linux, macOS, and
  native Windows. Only `git` and `python3` need to be on PATH.
- **Copy-based, not symlinks** — Claude Code rewrites `settings.json`
  atomically (write-then-rename), which silently replaces symlinks with plain
  files, so symlink-style dotfiles break. config-sync copies files both ways
  and uses git as the source of truth.
- **Secrets can never sync** — `auth.json`, `.credentials.json`, `*.pem`,
  `*.key`, `settings.local.json`, etc. are hard-excluded in the script *and*
  in the repo's `.gitignore`.
- **Self-propagating** — the script copies itself into your config repo
  (`bin/config_sync.py`), so a new VM only needs to clone the repo.

> **Use a private repo.** Your settings and CLAUDE.md often contain paths,
> hostnames, and preferences you don't want public.

## Setup

### 0. Create the config repo (once)

Create an empty **private** repo on GitHub (e.g. `my-agent-config`). Don't add
any files; `init` seeds it.

### 1. First machine

```bash
# Ubuntu / macOS
python3 config_sync.py init git@github.com:YOU/my-agent-config.git
python3 config_sync.py push
```

```powershell
# Windows (install python via: winget install Python.Python.3.12)
py config_sync.py init https://github.com/YOU/my-agent-config.git
py config_sync.py push
```

`init` clones the repo to `~/.config-sync`, seeds `manifest.conf` +
`.gitignore`, and embeds the script at `bin/config_sync.py`. `push` copies
your local config in, commits, and pushes.

### 2. Every other machine (Ubuntu VM, Windows VM, ...)

You don't even need this file — just the repo:

```bash
git clone git@github.com:YOU/my-agent-config.git ~/.config-sync
python3 ~/.config-sync/bin/config_sync.py init git@github.com:YOU/my-agent-config.git
python3 ~/.config-sync/bin/config_sync.py pull
```

Windows equivalent:

```powershell
git clone https://github.com/YOU/my-agent-config.git $HOME\.config-sync
py $HOME\.config-sync\bin\config_sync.py init https://github.com/YOU/my-agent-config.git
py $HOME\.config-sync\bin\config_sync.py pull
```

### 3. Stay in sync

```bash
python3 ~/.config-sync/bin/config_sync.py sync      # push local edits, apply remote edits
python3 ~/.config-sync/bin/config_sync.py status    # what would change, both directions
```

Or schedule it (every 30 min by default):

```bash
python3 ~/.config-sync/bin/config_sync.py schedule --every 30
```

- **Ubuntu**: installs a crontab entry, or a systemd user timer if cron is
  absent. Log at `~/.config-sync.log`.
- **Windows**: creates a Task Scheduler task named `ConfigSync`
  (remove with `schtasks /Delete /TN ConfigSync /F`).

## Commands

| Command | What it does |
|---|---|
| `init <git-url> [--dir PATH]` | Clone/attach the config repo, seed manifest, remember its location |
| `push [--dry-run]` | Copy local config → repo, commit, rebase on remote, push |
| `pull [--dry-run]` | Commit any drift, fetch remote, apply repo → local (with backups) |
| `sync` | `push` + `pull` in one shot — normal day-to-day command |
| `status` | Show pending differences in both directions and unpushed commits |
| `schedule [--every MIN]` | Run `sync --quiet` automatically via cron/systemd/Task Scheduler |

## What syncs (the manifest)

`manifest.conf` lives **inside the config repo**, so edits to it sync too.
Defaults:

```
.claude/settings.json          .claude/keybindings.json    .claude/CLAUDE.md
.claude/skills/                .claude/agents/             .claude/commands/
.claude/hooks/                 .claude/output-styles/      .claude/plugins/config.json
.codex/config.toml             .codex/AGENTS.md            .codex/prompts/
```

- A trailing `/` = whole directory, **mirrored** (deletions propagate).
- `plugins/config.json` carries which marketplaces/plugins are enabled;
  Claude Code re-clones the plugin repos themselves, so those aren't synced.
- Add your own lines (e.g. `.gitconfig`) or extra `exclude *.bak` patterns.
- Repo layout mirrors your home dir with the leading dot stripped:
  `.claude/skills/` → `claude/skills/`. Override with
  `src -> dest` syntax.

Always excluded, not overridable: `auth.json`, `.credentials.json`,
`credentials.json`, `*.pem`, `*.key`, `*.p12`, `*.pfx`, `id_rsa*`,
`id_ed25519*`, `*token*.json`, `settings.local.json` (machine-local by
design), plus junk (`.DS_Store`, `__pycache__`, `node_modules`, logs).

## Safety & conflict behavior

- Anything `pull`/`sync` overwrites or deletes locally is first copied to
  `~/.config-sync-backups/<timestamp>/` (last 10 snapshots kept).
- Concurrent edits on different machines merge via `git pull --rebase
  --autostash`. If two machines edited the *same line* of the same file, the
  rebase aborts cleanly and tells you to resolve it in `~/.config-sync` —
  nothing on disk is half-applied.
- Machines that don't have a tool installed (e.g. no `~/.codex`) simply skip
  those entries; nothing gets deleted from the repo.
- Effective semantics are **last-writer-wins per file**, with full history in
  git — `git -C ~/.config-sync log -p` shows every config change from every
  machine.
