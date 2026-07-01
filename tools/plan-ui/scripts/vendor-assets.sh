#!/usr/bin/env bash
# Refresh the vendored browser assets (Tailwind v4 JIT + DaisyUI v5).
# Maintenance-time only — the plugin never fetches these at runtime.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
web="$here/../web"

curl -fsSL "https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4/dist/index.global.js" -o "$web/tailwind.js"
curl -fsSL "https://cdn.jsdelivr.net/npm/daisyui@5/daisyui.css" -o "$web/daisyui.css"

echo "vendored:"
grep -oE 'nr="[0-9.]+"' "$web/tailwind.js" | head -1 | sed 's/nr=/  tailwind /'
head -c 120 "$web/daisyui.css" | grep -oE 'daisyUI [0-9.]+' | sed 's/^/  /'
