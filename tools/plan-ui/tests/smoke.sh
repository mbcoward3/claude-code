#!/usr/bin/env bash
# End-to-end smoke test for both plan-ui modes. Stdlib Python + curl only.
# Runs against an isolated HOME so it never touches your real ~/.plan-ui.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
plugin="$(dirname "$here")"
export HOME="$(mktemp -d)"
work="$(mktemp -d)"
trap 'python3 "$plugin/scripts/plan_ui.py" stop >/dev/null 2>&1 || true; rm -rf "$HOME" "$work"' EXIT

pu() { python3 "$plugin/scripts/plan_ui.py" "$@"; }
jget() { python3 -c "import sys,json;d=json.load(sys.stdin);print(eval(sys.argv[1]))" "$1"; }
fail() { echo "FAIL: $1" >&2; exit 1; }

# --- artifact mode -----------------------------------------------------------

cat > "$work/plan.html" <<'EOF'
<!doctype html><html><head><meta charset="utf-8"><title>Plan</title></head>
<body><h1>Test Plan</h1><p>Step 1: do the thing.</p></body></html>
EOF

out=$(pu open "$work/plan.html" --no-open)
key=$(echo "$out" | jget "d['session']['key']")
[ "$(echo "$out" | jget "d['session']['status']")" = "open" ] || fail "open status"
port=$(python3 -c "import json,os;print(json.load(open(os.path.expanduser('~/.plan-ui/server.json')))['port'])")
base="http://127.0.0.1:$port"

curl -sf "$base/s/$key" | grep -q "__PLAN_UI__ = { key: \"$key\" }" || fail "SDK not injected"
curl -sf "$base/assets/sdk.js" >/dev/null || fail "assets not served"

curl -sf -X POST "$base/api/$key/feedback" -H 'Content-Type: application/json' \
  -d '{"prompts":[{"text":"tighten step 1","action":"comment","target":{"selector":"p"}}]}' >/dev/null
text=$(pu poll "$work/plan.html" --timeout-ms 3000 | jget "d['prompts'][0]['text']")
[ "$text" = "tighten step 1" ] || fail "poll did not deliver feedback"

curl -sf -X POST "$base/api/$key/gate" -H 'Content-Type: application/json' \
  -d '{"warnings":[{"type":"overflow","selector":"html","detail":"too wide"}]}' >/dev/null
w=$(pu poll "$work/plan.html" --timeout-ms 3000 | jget "d['layout_warnings'][0]['type']")
[ "$w" = "overflow" ] || fail "gate warnings not delivered to poll"

[ "$(pu end "$work/plan.html" | jget "d['session']['status']")" = "ended" ] || fail "end"
echo "artifact loop: OK"

# --- approve action ------------------------------------------------------------

curl -sf -X POST "$base/api/$key/feedback" -H 'Content-Type: application/json' \
  -d '{"prompts":[{"text":"Approved.","action":"approve"}]}' >/dev/null
out=$(pu open "$work/plan.html" --no-open)  # reopen so poll works after end test ordering
act=$(pu poll "$work/plan.html" --timeout-ms 3000 | jget "d['prompts'][0]['action']")
[ "$act" = "approve" ] || fail "approve action not delivered"
echo "approve action: OK"

echo "ALL SMOKE TESTS PASSED"
