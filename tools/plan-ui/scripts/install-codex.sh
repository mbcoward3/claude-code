#!/usr/bin/env bash
# Install plan-ui as a Codex skill.
#
# The plugin directory doubles as an Agent-Skills skill (SKILL.md at its root,
# scripts/ alongside), so installation is one symlink into Codex's skills dir.
# Respects CODEX_HOME if set; defaults to ~/.codex.
set -euo pipefail

plugin="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills_dir="${CODEX_HOME:-$HOME/.codex}/skills"
dest="$skills_dir/plan-ui"

mkdir -p "$skills_dir"
ln -sfn "$plugin" "$dest"
echo "plan-ui installed as a Codex skill: $dest -> $plugin"
echo "Codex will trigger it when a task matches the skill description,"
echo "or invoke it explicitly with \$plan-ui."
