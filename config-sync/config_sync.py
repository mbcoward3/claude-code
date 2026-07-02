#!/usr/bin/env python3
"""config-sync: keep Claude Code and Codex config synced with a remote git repo.

Works on Ubuntu/Linux, macOS, and native Windows. Requires only Python 3.8+
and git on PATH.

Quick start (first machine):
    python3 config_sync.py init git@github.com:you/my-config.git
    python3 config_sync.py push

Every other machine:
    python3 config_sync.py init git@github.com:you/my-config.git
    python3 config_sync.py pull

Keep in sync afterwards (manually or scheduled):
    python3 config_sync.py sync
    python3 config_sync.py schedule --every 30

What syncs is controlled by manifest.conf inside the config repo, so the
manifest itself stays in sync across machines. Secrets (auth.json,
.credentials.json, keys, machine-local settings) are always excluded and can
never be pushed.
"""

import argparse
import filecmp
import fnmatch
import hashlib
import json
import os
import platform
import shutil
import socket
import subprocess
import sys
import time
from pathlib import Path

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

APP_NAME = "config-sync"

# Patterns that are NEVER synced, regardless of what the manifest says.
# Matched with fnmatch against every file name and every path component.
SECRET_EXCLUDES = [
    "auth.json",
    ".credentials.json",
    "credentials.json",
    "*.pem",
    "*.key",
    "*.p12",
    "*.pfx",
    "id_rsa*",
    "id_ed25519*",
    "*token*.json",
    "settings.local.json",  # machine-local by design
]

# Noise that should never be synced either.
JUNK_EXCLUDES = [
    ".DS_Store",
    "Thumbs.db",
    "__pycache__",
    "*.pyc",
    "node_modules",
    ".git",
    "*.log",
    "*.tmp",
    "*.swp",
]

DEFAULT_MANIFEST = """\
# config-sync manifest
# ---------------------
# One entry per line: a path relative to your home directory.
# A trailing slash means "sync the whole directory (mirrored, deletions
# propagate)". Without a trailing slash it is a single file.
#
# Optionally map to a different path inside the repo with "->":
#     .claude/settings.json -> claude/settings.json
# By default the leading dot of the first component is stripped, so
# ".claude/skills/" is stored in the repo as "claude/skills/".
#
# Lines starting with "exclude" add extra exclude patterns (fnmatch syntax):
#     exclude *.bak
#
# Secrets (auth.json, .credentials.json, *.pem, *.key, settings.local.json,
# ...) are ALWAYS excluded and cannot be re-enabled here.

# --- Claude Code -----------------------------------------------------------
.claude/settings.json
.claude/keybindings.json
.claude/CLAUDE.md
.claude/skills/
.claude/agents/
.claude/commands/
.claude/hooks/
.claude/output-styles/
# Plugin *state* (which marketplaces/plugins are enabled). The plugin repos
# themselves are git clones that Claude Code restores from this file.
.claude/plugins/config.json

# --- Codex -----------------------------------------------------------------
.codex/config.toml
.codex/AGENTS.md
.codex/prompts/

# --- Extra exclude patterns ------------------------------------------------
# exclude *.bak
"""

REPO_GITIGNORE = """\
# Belt-and-braces: config-sync already refuses to copy these, but make sure
# git never picks them up either.
auth.json
.credentials.json
credentials.json
*.pem
*.key
*.p12
*.pfx
id_rsa*
id_ed25519*
settings.local.json
.DS_Store
Thumbs.db
__pycache__/
node_modules/
*.log
"""

# ---------------------------------------------------------------------------
# Small helpers
# ---------------------------------------------------------------------------


def home() -> Path:
    override = os.environ.get("CONFIG_SYNC_HOME")
    return Path(override) if override else Path.home()


def state_file() -> Path:
    return home() / ".config-sync.json"


def load_state() -> dict:
    try:
        return json.loads(state_file().read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return {}


def save_state(state: dict) -> None:
    state_file().write_text(json.dumps(state, indent=2) + "\n", encoding="utf-8")


def repo_dir(required: bool = True) -> Path:
    override = os.environ.get("CONFIG_SYNC_REPO")
    if override:
        return Path(override)
    state = load_state()
    if "repo_dir" in state:
        return Path(state["repo_dir"])
    if required:
        die(f"not initialised - run: {prog()} init <git-url>")
    return home() / ".config-sync"


def prog() -> str:
    return f"python3 {Path(sys.argv[0]).name}" if os.name != "nt" else f"py {Path(sys.argv[0]).name}"


def log(msg: str) -> None:
    if not QUIET:
        print(msg)


def die(msg: str, code: int = 1) -> None:
    print(f"{APP_NAME}: error: {msg}", file=sys.stderr)
    sys.exit(code)


def run_git(repo: Path, *args: str, check: bool = True, capture: bool = True) -> subprocess.CompletedProcess:
    cmd = ["git", "-C", str(repo)] + list(args)
    return subprocess.run(cmd, check=check, capture_output=capture, text=True)


def git_out(repo: Path, *args: str) -> str:
    return run_git(repo, *args).stdout.strip()


def ensure_git_identity(repo: Path) -> None:
    """Make sure commits can be created even on a fresh VM with no git config."""
    for key, fallback in (("user.name", f"config-sync@{socket.gethostname()}"),
                          ("user.email", "config-sync@localhost")):
        r = run_git(repo, "config", key, check=False)
        if r.returncode != 0 or not r.stdout.strip():
            run_git(repo, "config", key, fallback)


# ---------------------------------------------------------------------------
# Manifest
# ---------------------------------------------------------------------------


class Entry:
    def __init__(self, src: str, dest: str, is_dir: bool):
        self.src = src          # relative to home
        self.dest = dest        # relative to repo
        self.is_dir = is_dir

    def __repr__(self):
        return f"Entry({self.src!r} -> {self.dest!r}, dir={self.is_dir})"


def default_dest(src: str) -> str:
    """'.claude/skills' -> 'claude/skills'."""
    parts = src.strip("/").split("/")
    if parts and parts[0].startswith("."):
        parts[0] = parts[0][1:]
    return "/".join(parts)


def parse_manifest(text: str):
    entries, extra_excludes = [], []
    for raw in text.splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        if line.startswith("exclude "):
            extra_excludes.append(line[len("exclude "):].strip())
            continue
        if "->" in line:
            src, dest = (p.strip() for p in line.split("->", 1))
        else:
            src, dest = line, default_dest(line)
        is_dir = src.endswith("/")
        entries.append(Entry(src.strip("/"), dest.strip("/"), is_dir))
    return entries, extra_excludes


def load_manifest(repo: Path):
    mf = repo / "manifest.conf"
    if not mf.is_file():
        die(f"no manifest.conf in {repo} - run: {prog()} init <git-url>")
    return parse_manifest(mf.read_text(encoding="utf-8"))


# ---------------------------------------------------------------------------
# Copying with exclusions
# ---------------------------------------------------------------------------


def is_excluded(rel_path: str, extra: list) -> bool:
    patterns = SECRET_EXCLUDES + JUNK_EXCLUDES + extra
    parts = rel_path.replace(os.sep, "/").split("/")
    return any(fnmatch.fnmatch(part, pat) for part in parts for pat in patterns)


def file_digest(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def files_differ(a: Path, b: Path) -> bool:
    if not a.exists() or not b.exists():
        return True
    if a.stat().st_size != b.stat().st_size:
        return True
    return not filecmp.cmp(a, b, shallow=False)


def copy_file(src: Path, dest: Path, changes: list) -> None:
    if files_differ(src, dest):
        dest.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(src, dest)
        changes.append(str(dest))


def walk_files(root: Path, extra_excludes: list):
    """Yield paths relative to root, honoring exclusions (prunes dirs)."""
    for dirpath, dirnames, filenames in os.walk(root):
        rel_dir = os.path.relpath(dirpath, root)
        rel_dir = "" if rel_dir == "." else rel_dir.replace(os.sep, "/")
        dirnames[:] = [d for d in dirnames
                       if not is_excluded(f"{rel_dir}/{d}" if rel_dir else d, extra_excludes)]
        for name in filenames:
            rel = f"{rel_dir}/{name}" if rel_dir else name
            if not is_excluded(rel, extra_excludes):
                yield rel


def mirror_dir(src_root: Path, dest_root: Path, extra_excludes: list,
               changes: list, backup_root=None) -> None:
    """Mirror src_root into dest_root: copy changed files, remove files that
    no longer exist in src (deletions propagate). Excluded files are never
    copied and never deleted from the destination."""
    src_files = set(walk_files(src_root, extra_excludes)) if src_root.is_dir() else set()
    dest_files = set(walk_files(dest_root, extra_excludes)) if dest_root.is_dir() else set()

    for rel in sorted(src_files):
        s, d = src_root / rel, dest_root / rel
        if files_differ(s, d):
            if backup_root is not None and d.exists():
                backup_copy(d, backup_root / rel)
            d.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(s, d)
            changes.append(str(d))

    for rel in sorted(dest_files - src_files):
        d = dest_root / rel
        if backup_root is not None:
            backup_copy(d, backup_root / rel)
        d.unlink()
        changes.append(f"{d} (removed)")

    # prune now-empty directories on the destination side
    if dest_root.is_dir():
        for dirpath, dirnames, filenames in os.walk(dest_root, topdown=False):
            if not dirnames and not filenames and Path(dirpath) != dest_root:
                try:
                    os.rmdir(dirpath)
                except OSError:
                    pass


def backup_copy(src: Path, dest: Path) -> None:
    dest.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dest)


# ---------------------------------------------------------------------------
# Sync directions
# ---------------------------------------------------------------------------


def collect_local_to_repo(repo: Path, dry: bool = False):
    """home -> repo working tree. Returns list of changed repo paths."""
    entries, extra = load_manifest(repo)
    changes = []
    for e in entries:
        src, dest = home() / e.src, repo / e.dest
        if e.is_dir:
            if src.is_dir():
                if not dry:
                    mirror_dir(src, dest, extra, changes)
                else:
                    changes.extend(diff_dir(src, dest, extra))
            # if the local dir doesn't exist, leave the repo copy alone
            # (this machine may simply not use that tool)
        else:
            if src.is_file() and not is_excluded(e.src, extra):
                if files_differ(src, dest):
                    changes.append(str(dest))
                    if not dry:
                        dest.parent.mkdir(parents=True, exist_ok=True)
                        shutil.copy2(src, dest)
    return changes


def apply_repo_to_local(repo: Path, dry: bool = False):
    """repo working tree -> home, backing up anything overwritten/removed."""
    entries, extra = load_manifest(repo)
    ts = time.strftime("%Y%m%d-%H%M%S")
    backup_root = home() / f".config-sync-backups" / ts
    changes = []
    for e in entries:
        src, dest = repo / e.dest, home() / e.src
        if e.is_dir:
            if src.is_dir():
                if not dry:
                    mirror_dir(src, dest, extra, changes, backup_root=backup_root)
                else:
                    changes.extend(diff_dir(src, dest, extra))
        else:
            if src.is_file() and not is_excluded(e.src, extra):
                if files_differ(src, dest):
                    changes.append(str(dest))
                    if not dry:
                        if dest.exists():
                            backup_copy(dest, backup_root / e.src)
                        dest.parent.mkdir(parents=True, exist_ok=True)
                        shutil.copy2(src, dest)
    if changes and not dry and backup_root.is_dir():
        log(f"  (previous versions backed up to {backup_root})")
    prune_backups(backup_root.parent)
    return changes


def diff_dir(src_root: Path, dest_root: Path, extra: list):
    out = []
    src_files = set(walk_files(src_root, extra)) if src_root.is_dir() else set()
    dest_files = set(walk_files(dest_root, extra)) if dest_root.is_dir() else set()
    for rel in sorted(src_files):
        if files_differ(src_root / rel, dest_root / rel):
            out.append(str(dest_root / rel))
    for rel in sorted(dest_files - src_files):
        out.append(f"{dest_root / rel} (would remove)")
    return out


def prune_backups(backups_dir: Path, keep: int = 10) -> None:
    if not backups_dir.is_dir():
        return
    snaps = sorted(p for p in backups_dir.iterdir() if p.is_dir())
    for old in snaps[:-keep]:
        shutil.rmtree(old, ignore_errors=True)


# ---------------------------------------------------------------------------
# Git plumbing
# ---------------------------------------------------------------------------


def git_commit_if_needed(repo: Path) -> bool:
    if not git_out(repo, "status", "--porcelain"):
        return False
    ensure_git_identity(repo)
    run_git(repo, "add", "-A")
    msg = f"sync from {socket.gethostname()} ({platform.system()}) {time.strftime('%Y-%m-%d %H:%M:%S')}"
    run_git(repo, "commit", "-m", msg)
    return True


def git_pull_rebase(repo: Path) -> None:
    if not git_out(repo, "remote"):
        return  # local-only repo, nothing to pull
    r = run_git(repo, "pull", "--rebase", "--autostash", "origin", current_branch(repo), check=False)
    if r.returncode != 0:
        # remote may be an empty repo with no branch yet - that's fine
        if "couldn't find remote ref" in (r.stderr or "").lower():
            return
        run_git(repo, "rebase", "--abort", check=False)
        die("git pull --rebase failed (conflicting edits from another machine?).\n"
            f"Resolve manually in {repo} then re-run.\n\n{r.stderr}")


def git_push(repo: Path) -> None:
    if not git_out(repo, "remote"):
        log("  (no remote configured, skipping push)")
        return
    branch = current_branch(repo)
    delay = 2
    for attempt in range(5):
        r = run_git(repo, "push", "-u", "origin", branch, check=False)
        if r.returncode == 0:
            return
        err = (r.stderr or "").lower()
        transient = any(s in err for s in ("could not resolve", "timed out", "connection", "503", "502"))
        if attempt < 4 and transient:
            log(f"  push failed (network), retrying in {delay}s...")
            time.sleep(delay)
            delay *= 2
        else:
            die(f"git push failed:\n{r.stderr}")


def current_branch(repo: Path) -> str:
    b = git_out(repo, "rev-parse", "--abbrev-ref", "HEAD")
    return b if b != "HEAD" else "main"


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------


def cmd_init(args) -> None:
    repo = Path(args.dir).expanduser() if args.dir else home() / ".config-sync"
    if repo.is_dir() and (repo / ".git").is_dir():
        log(f"using existing clone at {repo}")
    elif repo.exists() and any(repo.iterdir()):
        die(f"{repo} exists and is not a git clone")
    else:
        log(f"cloning {args.url} -> {repo}")
        r = subprocess.run(["git", "clone", args.url, str(repo)],
                           capture_output=True, text=True)
        if r.returncode != 0:
            # empty remote repos can still be cloned by modern git; if the URL
            # itself is bad, surface it
            die(f"git clone failed:\n{r.stderr}")

    ensure_git_identity(repo)

    # Fresh/empty repo: create an initial branch so we can commit.
    if not git_out(repo, "branch", "--list"):
        run_git(repo, "checkout", "-b", "main", check=False)

    seeded = False
    if not (repo / "manifest.conf").is_file():
        (repo / "manifest.conf").write_text(DEFAULT_MANIFEST, encoding="utf-8")
        seeded = True
    if not (repo / ".gitignore").is_file():
        (repo / ".gitignore").write_text(REPO_GITIGNORE, encoding="utf-8")
        seeded = True
    # Carry the tool inside the repo so new machines can bootstrap from the
    # clone alone.
    self_copy = repo / "bin" / "config_sync.py"
    me = Path(__file__).resolve()
    running_from_repo = self_copy.exists() and me == self_copy.resolve()
    if not running_from_repo and (not self_copy.exists() or files_differ(me, self_copy)):
        self_copy.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(me, self_copy)
        seeded = True

    if seeded and git_commit_if_needed(repo):
        log("committed manifest/bootstrap files")

    save_state({"repo_dir": str(repo)})
    log(f"initialised. repo: {repo}")
    log(f"next: '{prog()} push' (first machine) or '{prog()} pull' (new machine)")


def cmd_push(args) -> None:
    repo = repo_dir()
    changes = collect_local_to_repo(repo, dry=args.dry_run)
    if args.dry_run:
        report("would push", changes)
        return
    report("updated in repo", changes)
    if git_commit_if_needed(repo):
        git_pull_rebase(repo)
        git_push(repo)
        log("pushed.")
    else:
        # still push any locally-committed-but-unpushed work
        git_pull_rebase(repo)
        git_push(repo)
        log("nothing new to commit; remote is up to date.")


def cmd_pull(args) -> None:
    repo = repo_dir()
    # Don't let unpushed local edits be silently clobbered: commit them first
    # so they're preserved in git history even though pull overwrites $HOME.
    if not args.dry_run:
        git_commit_if_needed(repo)
        git_pull_rebase(repo)
    changes = apply_repo_to_local(repo, dry=args.dry_run)
    report("would apply" if args.dry_run else "applied to home", changes)


def cmd_sync(args) -> None:
    repo = repo_dir()
    pushed = collect_local_to_repo(repo)
    report("local changes captured", pushed)
    git_commit_if_needed(repo)
    git_pull_rebase(repo)
    applied = apply_repo_to_local(repo)
    report("remote changes applied", applied)
    git_push(repo)
    log("in sync.")


def cmd_status(args) -> None:
    repo = repo_dir()
    to_repo = collect_local_to_repo(repo, dry=True)
    to_home = apply_repo_to_local(repo, dry=True)
    if not to_repo and not to_home:
        print("clean: local config matches the repo working tree.")
    else:
        report("local -> repo (push would update)", to_repo, always=True)
        report("repo -> local (pull would update)", to_home, always=True)
    unpushed = git_out(repo, "log", "--oneline", "@{u}..") if has_upstream(repo) else ""
    if unpushed:
        print(f"\nunpushed commits:\n{unpushed}")


def has_upstream(repo: Path) -> bool:
    return run_git(repo, "rev-parse", "--abbrev-ref", "@{u}", check=False).returncode == 0


def cmd_schedule(args) -> None:
    repo = repo_dir()
    script = repo / "bin" / "config_sync.py"
    if not script.is_file():
        script = Path(__file__).resolve()
    minutes = args.every

    if os.name == "nt":
        py = sys.executable or "py"
        task_cmd = f'"{py}" "{script}" sync --quiet'
        r = subprocess.run(
            ["schtasks", "/Create", "/F", "/SC", "MINUTE", "/MO", str(minutes),
             "/TN", "ConfigSync", "/TR", task_cmd],
            capture_output=True, text=True)
        if r.returncode != 0:
            die(f"schtasks failed:\n{r.stderr}")
        log(f"Windows scheduled task 'ConfigSync' created (every {minutes} min).")
        log("Remove with: schtasks /Delete /TN ConfigSync /F")
        return

    py = sys.executable or "python3"
    line = f"*/{minutes} * * * * {py} {script} sync --quiet >> {home() / '.config-sync.log'} 2>&1"
    if shutil.which("crontab"):
        r = subprocess.run(["crontab", "-l"], capture_output=True, text=True)
        existing = r.stdout if r.returncode == 0 else ""
        kept = [l for l in existing.splitlines() if "config_sync.py sync" not in l]
        new_tab = "\n".join(kept + [line]) + "\n"
        subprocess.run(["crontab", "-"], input=new_tab, text=True, check=True)
        log(f"cron entry installed (every {minutes} min).")
        return

    # Fall back to a systemd user timer (common on minimal Ubuntu without cron)
    unit_dir = home() / ".config" / "systemd" / "user"
    unit_dir.mkdir(parents=True, exist_ok=True)
    (unit_dir / "config-sync.service").write_text(
        f"[Unit]\nDescription=config-sync\n\n[Service]\nType=oneshot\n"
        f"ExecStart={py} {script} sync --quiet\n", encoding="utf-8")
    (unit_dir / "config-sync.timer").write_text(
        f"[Unit]\nDescription=config-sync timer\n\n[Timer]\n"
        f"OnBootSec=2min\nOnUnitActiveSec={minutes}min\n\n"
        f"[Install]\nWantedBy=timers.target\n", encoding="utf-8")
    r = subprocess.run(["systemctl", "--user", "enable", "--now", "config-sync.timer"],
                       capture_output=True, text=True)
    if r.returncode != 0:
        die(f"no crontab and systemd user timer failed:\n{r.stderr}\n"
            f"Unit files were written to {unit_dir}; enable them manually.")
    log(f"systemd user timer enabled (every {minutes} min).")


def report(label: str, changes: list, always: bool = False) -> None:
    if not changes and not always:
        return
    if not changes:
        print(f"{label}: (none)")
        return
    log(f"{label}:")
    for c in changes:
        log(f"  {c}")


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------

QUIET = False


def main() -> None:
    global QUIET
    ap = argparse.ArgumentParser(prog=APP_NAME, description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--quiet", action="store_true", help="suppress non-error output")
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("init", help="clone/attach the config repo and seed the manifest")
    p.add_argument("url", help="git URL of your (private) config repo")
    p.add_argument("--dir", help="where to keep the clone (default: ~/.config-sync)")
    p.set_defaults(fn=cmd_init)

    for name, fn, hlp in (("push", cmd_push, "copy local config into the repo, commit, push"),
                          ("pull", cmd_pull, "fetch remote and apply the repo onto this machine")):
        p = sub.add_parser(name, help=hlp)
        p.add_argument("--dry-run", action="store_true", help="show what would change")
        p.set_defaults(fn=fn)

    p = sub.add_parser("sync", help="push local changes, then apply remote changes (both ways)")
    p.set_defaults(fn=cmd_sync)

    p = sub.add_parser("status", help="show pending differences in both directions")
    p.set_defaults(fn=cmd_status)

    p = sub.add_parser("schedule", help="run 'sync' automatically (cron/systemd/Task Scheduler)")
    p.add_argument("--every", type=int, default=30, metavar="MINUTES",
                   help="interval in minutes (default 30)")
    p.set_defaults(fn=cmd_schedule)

    args = ap.parse_args()
    QUIET = args.quiet
    if not shutil.which("git"):
        die("git not found on PATH")
    args.fn(args)


if __name__ == "__main__":
    main()
