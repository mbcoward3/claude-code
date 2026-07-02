#!/usr/bin/env bash
# Install plan-ui as a Codex skill.
#
# The plugin directory doubles as an Agent-Skills skill (SKILL.md at its root,
# scripts/ and agents/openai.yaml alongside), so installation is one symlink
# into the user skills directory — ~/.agents/skills per the Codex docs
# (https://developers.openai.com/codex/skills). Override with SKILLS_DIR for a
# repo-scoped install (.agents/skills) or a nonstandard location.
set -euo pipefail

plugin="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skills_dir="${SKILLS_DIR:-$HOME/.agents/skills}"
dest="$skills_dir/plan-ui"

mkdir -p "$skills_dir"
ln -sfn "$plugin" "$dest"
echo "plan-ui installed as a Codex skill: $dest -> $plugin"
echo "Codex will trigger it when a task matches the skill description,"
echo "or invoke it explicitly with \$plan-ui."
