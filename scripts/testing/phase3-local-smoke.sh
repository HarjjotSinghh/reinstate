#!/usr/bin/env bash
# Local Phase 3 smoke using ./bin/rein only (never global install).
set -euo pipefail
ROOT="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
REIN="$ROOT/bin/rein"
test -x "$REIN" || { echo "run: make build" >&2; exit 2; }
EVID="${PHASE3_SMOKE_ROOT:-$HOME/.reinstate-phase3-local-smoke}"
rm -rf "$EVID"
mkdir -p "$EVID"/{home,project,claude/projects,codex/sessions,stubs}

cat >"$EVID/stubs/codex" <<'SH'
#!/bin/sh
[ "$1" = "--version" ] && { echo "codex-cli 0.147.0"; exit 0; }
echo "stub codex ok $*"; exit 0
SH
cat >"$EVID/stubs/claude" <<'SH'
#!/bin/sh
[ "$1" = "--version" ] && { echo "2.1.225 (Claude Code)"; exit 0; }
echo "stub claude ok $*"; exit 0
SH
chmod +x "$EVID/stubs/codex" "$EVID/stubs/claude"

cd "$EVID/project"
git init -b main >/dev/null
git config user.email smoke@test
git config user.name smoke
git config core.autocrlf false
printf 'x\n' > README.md
git add README.md && git commit -m i >/dev/null
printf '2.1.225\n' > "$EVID/claude/version"

CSID=11111111-0000-4000-8000-000000001111
XSID=22222222-0000-4000-8000-000000002222
PROJ="$(pwd -P)"
ENC="${PROJ//\//-}"
mkdir -p "$EVID/claude/projects/$ENC"
python3 - "$EVID" "$PROJ" "$ENC" "$CSID" "$XSID" <<'PY'
import json, pathlib, sys
evid, proj, enc, csid, xsid = sys.argv[1:6]
cdir = pathlib.Path(evid)/"claude"/"projects"/enc
(cdir/f"{csid}.jsonl").write_text("\n".join([
  json.dumps({"type":"user","sessionId":csid,"cwd":proj,"gitBranch":"main","timestamp":"2026-08-11T10:00:00Z","message":{"role":"user","content":"smoke"}}),
  json.dumps({"type":"assistant","sessionId":csid,"timestamp":"2026-08-11T10:00:01Z","message":{"role":"assistant","content":"ok"}}),
])+"\n")
sdir = pathlib.Path(evid)/"codex"/"sessions"
(sdir/f"rollout-{xsid}.jsonl").write_text("\n".join([
  json.dumps({"type":"session_meta","payload":{"id":xsid,"session_id":xsid,"cwd":proj,"git":{"branch":"main"}}}),
  json.dumps({"type":"message","role":"user","content":"smoke"}),
])+"\n")
PY

export REINSTATE_HOME="$EVID/home" CLAUDE_CONFIG_DIR="$EVID/claude" CODEX_HOME="$EVID/codex"
# Stubs exit 0 without a TTY; production native launches still require a real terminal.
export REINSTATE_ALLOW_NON_TTY_LAUNCH=1
export PATH="$EVID/stubs:$(dirname "$REIN"):/usr/bin:/bin:/opt/homebrew/bin${PATH:+:$PATH}"

"$REIN" resume "codex:$XSID" --allow-environment-warning baseline.unavailable >/dev/null
"$REIN" resume "claude:$CSID" --allow-environment-warning baseline.unavailable >/dev/null
dec_c=$("$REIN" inspect "claude:$CSID" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["environment"]["decision"])')
dec_x=$("$REIN" inspect "codex:$XSID" --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["environment"]["decision"])')
test "$dec_c" = "ready"
test "$dec_x" = "ready"
echo "phase3-local-smoke PASS (claude=$dec_c codex=$dec_x rein=$("$REIN" version --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])'))"

# --- extended assertions (before repo-replacement mutates the fixture) ---
# invalid warning ID
set +e
"$REIN" resume "claude:$CSID" --allow-environment-warning not.a.real.id >/dev/null 2>&1
code=$?
set -e
test "$code" -eq 2

# hard blocker without vendor on PATH
set +e
PATH="$(dirname "$REIN"):/usr/bin:/bin" "$REIN" resume "claude:$CSID" --dry-run >/dev/null 2>&1
code=$?
set -e
test "$code" -eq 5

# dirty tree requires ack
printf 'dirty\n' >> README.md
set +e
"$REIN" resume "claude:$CSID" >/dev/null 2>&1
code=$?
set -e
test "$code" -eq 7
"$REIN" resume "claude:$CSID" --allow-environment-warning git.working_tree >/dev/null
# Restore a clean tree and re-baseline so later ready-state rows are not poisoned.
git checkout -- README.md >/dev/null
"$REIN" resume "claude:$CSID" --allow-environment-warning git.working_tree >/dev/null

# privacy: human inspect should not emit absolute home paths
out=$("$REIN" inspect "claude:$CSID" 2>/dev/null || true)
if printf '%s' "$out" | grep -E '/Users/|C:\\\\Users\\\\' >/dev/null; then
  echo "privacy FAIL: absolute user path in human inspect" >&2
  exit 1
fi

# fork dry-run while environment is still ready
"$REIN" fork "claude:$CSID" --dry-run --json >"$EVID/fork-c.json"
"$REIN" fork "codex:$XSID" --dry-run --json >"$EVID/fork-x.json"
python3 - "$EVID" <<'PY'
import json, pathlib, sys
evid = pathlib.Path(sys.argv[1])
for name in ("fork-c.json", "fork-x.json"):
    d = json.loads((evid / name).read_text())
    assert d.get("operation") == "fork", d
    assert d["environment"]["decision"] == "ready", d["environment"]["decision"]
print("fork dry-run PASS")
PY

# alias parity: rein vs reinstate (same binary family)
"$REIN" sessions --limit 10 --json >"$EVID/sessions-rein.json"
"$(dirname "$REIN")/reinstate" sessions --limit 10 --json >"$EVID/sessions-reinstate.json"
python3 - "$EVID" <<'PY'
import json, pathlib, sys
evid = pathlib.Path(sys.argv[1])
a = json.loads((evid / "sessions-rein.json").read_text())
b = json.loads((evid / "sessions-reinstate.json").read_text())
sa = sorted(s["id"] for s in a["sessions"])
sb = sorted(s["id"] for s in b["sessions"])
assert sa == sb and len(sa) >= 2, (sa, sb)
print("alias parity PASS", len(sa))
PY

# missing workspace: rewrite codex cwd, expect exit 5, then restore for later rows
CODEX_ROLL="$EVID/codex/sessions/rollout-$XSID.jsonl"
cp "$CODEX_ROLL" "$EVID/codex-rollout.bak"
python3 - "$CODEX_ROLL" <<'PY'
import json, pathlib, sys
p = pathlib.Path(sys.argv[1])
lines = []
for line in p.read_text().splitlines():
    o = json.loads(line)
    if o.get("type") == "session_meta":
        o["payload"]["cwd"] = "/tmp/reinstate-missing-workspace-path-does-not-exist"
    lines.append(json.dumps(o))
p.write_text("\n".join(lines) + "\n")
PY
set +e
"$REIN" resume "codex:$XSID" --dry-run >/dev/null 2>&1
code=$?
set -e
test "$code" -eq 5
mv "$EVID/codex-rollout.bak" "$CODEX_ROLL"
echo "missing-workspace block PASS (exit=5)"

echo "phase3-local-smoke extended PASS"

# --- repo replacement after baseline must block (last: mutates project git) ---
# Keep same cwd path but re-init a different repository identity.
rm -rf .git
git init -b main >/dev/null
git config user.email smoke@test
git config user.name smoke
git config core.autocrlf false
printf 'replaced\n' > README.md
git add README.md && git commit -m replaced >/dev/null
set +e
"$REIN" resume "codex:$XSID" --dry-run >/dev/null 2>&1
code=$?
set -e
test "$code" -eq 7
echo "phase3-local-smoke repo-replacement block PASS"
echo "phase3-local-smoke ALL PASS"
