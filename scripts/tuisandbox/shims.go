// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog" // registers the shipped descriptors
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// Why the bench needs fake vendor binaries at all:
//
// preflight's agent probe does three things per row — resolve the vendor
// executable, confirm the agent's private root layout, and run the executable
// with --version. All three must succeed or `agent.executable` / `agent.version`
// block the row, and a blocked row can never demonstrate READY. Installing the
// real Claude Code or Codex CLI is not an option for a hermetic bench, so the
// sandbox ships shims that satisfy exactly the contract the probe checks and
// nothing more.
//
// Four properties are load-bearing and easy to get wrong:
//
//  1. The PATH entry must be ABSOLUTE. internal/executabletrust drops relative
//     PATH entries outright.
//  2. The PATH entry must sit OUTSIDE the trust boundary, which is the
//     outermost `.git`-marked ancestor of the workspace being launched into
//     (internal/executabletrust/resolve.go trustBoundary). Every demo workspace
//     is a Git repository, so <root>/bin — a sibling of <root>/home — is safe,
//     and a -root that itself lives inside a checkout is not. main.go refuses
//     that case rather than producing nine mysteriously blocked rows.
//  3. Mode must be 0755. A readable but non-executable file still resolves and
//     then fails the version probe, which is the most confusing failure of the
//     set: "native agent version probe failed" with the binary sitting right
//     there.
//  4. --version must print exactly one line on stdout and NOTHING on stderr.
//     internal/agents keeps the two streams apart (VersionOutput) precisely so
//     a warning on stderr cannot be mistaken for version output; any stderr
//     byte turns into `agent.version` "native agent version is unrecognized".
//
// The version string itself must land inside the catalog's inclusive
// fail-closed range. Those ranges are read from the catalog at generate time
// (see shimVersion) so a catalog bump cannot silently turn the whole bench red.

// preferredClaudeVersion / preferredCodexVersion are the versions the bench
// prints when they are still inside the catalog range. They are pinned rather
// than derived so the printed legend and the docs stay readable; shimVersion
// falls back to the catalog minimum the moment the range moves past them.
const (
	preferredClaudeVersion = "2.1.228"
	preferredCodexVersion  = "0.140.0"
)

// shimVersion returns a version string for agent that is guaranteed to be
// inside the catalog's [Min, Max] range. Deriving it means a catalog bump
// either keeps working (preferred version still in range) or self-heals to the
// new minimum, instead of turning every row of the bench into an
// `agent.version` block that looks like a product regression.
func shimVersion(agent, preferred string) (string, error) {
	descriptor, ok := agents.Get(agent)
	if !ok || descriptor.Version == nil {
		return "", fmt.Errorf("no version range registered for agent %q", agent)
	}
	min, max := descriptor.Version.Min, descriptor.Version.Max
	if compareTriple(preferred, min) >= 0 && compareTriple(preferred, max) <= 0 {
		return preferred, nil
	}
	return min, nil
}

// compareTriple orders two dotted numeric versions. Missing or non-numeric
// components sort as zero, which is only ever reached if the catalog stops
// using plain X.Y.Z — in which case falling back to Min is still correct.
func compareTriple(left, right string) int {
	l, r := splitTriple(left), splitTriple(right)
	for i := 0; i < 3; i++ {
		switch {
		case l[i] < r[i]:
			return -1
		case l[i] > r[i]:
			return 1
		}
	}
	return 0
}

func splitTriple(value string) [3]int {
	var out [3]int
	for i, part := range strings.SplitN(strings.TrimSpace(value), ".", 3) {
		if i > 2 {
			break
		}
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return out
		}
		out[i] = n
	}
	return out
}

// shimSet is the versions actually written into the sandbox, so the legend can
// quote them without recomputing.
type shimSet struct {
	binDir      string
	staleDir    string
	claude      string
	codex       string
	staleClaude string
}

// staleClaudeVersion is deliberately outside every plausible catalog range. It
// backs the -stale-claude lever: prepending <root>/bin-stale turns every Claude
// row into `agent.version` "outside the verified range", which is the refusal
// the fail-closed range exists for. It cannot be a per-session lever, because
// the executable name is fixed per agent by the catalog — version state is
// necessarily global to the sandbox.
const staleClaudeVersion = "2.1.300"

// writeShims materializes <root>/bin (in-range) and <root>/bin-stale (claude
// only, out of range). Both are always written; -stale-claude only changes
// which one printEnv puts first on PATH.
func writeShims(root string) (shimSet, error) {
	set := shimSet{
		binDir:      filepath.Join(root, "bin"),
		staleDir:    filepath.Join(root, "bin-stale"),
		staleClaude: staleClaudeVersion,
	}
	var err error
	if set.claude, err = shimVersion(sessionindex.AgentClaude, preferredClaudeVersion); err != nil {
		return shimSet{}, err
	}
	if set.codex, err = shimVersion(sessionindex.AgentCodex, preferredCodexVersion); err != nil {
		return shimSet{}, err
	}
	for _, dir := range []string{set.binDir, set.staleDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return shimSet{}, err
		}
	}
	if runtime.GOOS == "windows" {
		if err := writeWindowsShims(set); err != nil {
			return shimSet{}, err
		}
		return set, nil
	}
	if err := writeFile(filepath.Join(set.binDir, "claude"), claudeShim(set.claude), 0o755); err != nil {
		return shimSet{}, err
	}
	if err := writeFile(filepath.Join(set.staleDir, "claude"), claudeShim(set.staleClaude), 0o755); err != nil {
		return shimSet{}, err
	}
	if err := writeFile(filepath.Join(set.binDir, "codex"), codexShim(set.codex), 0o755); err != nil {
		return shimSet{}, err
	}
	// The unresolved-handoff demo: same version contract, but it never writes a
	// rollout file, so handoff lineage records launched=true with
	// destination.state="unresolved".
	return set, writeFile(filepath.Join(set.staleDir, "codex-silent"), silentCodexShim(set.codex), 0o755)
}

func writeFile(path, body string, mode os.FileMode) error {
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		return err
	}
	// WriteFile honours umask, which on many hosts strips group/other execute.
	// The probe only needs owner-execute, but 0755 is what the shipped fixtures
	// use and what the comment above promises, so set it explicitly.
	return os.Chmod(path, mode)
}

// claudeShim answers --version and otherwise echoes what it was asked to do.
// It never rewrites itself: the launch boundary SHA-256s the executable and
// aborts if size, mode, mtime, or content moved between plan and exec.
func claudeShim(version string) string {
	return `#!/bin/sh
# Synthetic Claude Code stand-in for the Reinstate TUI bench. Not a real agent.
if [ "$1" = "--version" ]; then
  # Exactly one line on stdout, nothing on stderr: any stderr byte makes the
  # version probe report "unrecognized" and block the row.
  printf '%s\n' '` + version + ` (Claude Code)'
  exit 0
fi
printf 'reinstate sandbox: claude %s\n' "$*"
printf 'cwd: %s\n' "$(pwd)"
printf 'press enter to exit '
read _ 2>/dev/null || true
exit 0
`
}

// codexShim answers --version, prints resume/fork, and — for anything else —
// treats argv[1] as a structured-handoff bootstrap prompt and writes a rollout
// file so the handoff can reconcile.
//
// Reconciliation needs three things simultaneously, all produced below:
//
//  1. the file under $CODEX_HOME/sessions/ named rollout-<stamp>-<uuid>.jsonl
//     with a lower-hex 8-4-4-4-12 uuid;
//  2. payload.cwd equal (after Clean) to the child's own working directory —
//     the child is started in plan.Dir, so `pwd -P` is exactly right;
//  3. the first user message byte-identical to argv[1], because the reconciler
//     SHA-256s it. The sed/awk dance escapes backslash then double quote and
//     re-emits every line with a literal \n, which reproduces the trailing
//     newline the bootstrap ends with. event_msg/user_message is preferred
//     because the reader consults it before response_item.
//
// The uuid is generated per invocation. A hardcoded one still reconciles (the
// mtime filter hides older files and the index dedupes by key) but quietly
// accumulates N rollout files all claiming one session id.
func codexShim(version string) string {
	return `#!/bin/sh
# Synthetic Codex CLI stand-in for the Reinstate TUI bench. Not a real agent.
case "$1" in
  --version) printf '%s\n' 'codex-cli ` + version + `'; exit 0 ;;
  resume|fork)
    printf 'reinstate sandbox: codex %s %s\n' "$1" "$2"
    printf 'cwd: %s\n' "$(pwd)"
    printf 'press enter to exit '
    read _ 2>/dev/null || true
    exit 0 ;;
esac

# Structured handoff: argv[1] IS the whole bootstrap prompt.
root="${CODEX_HOME:-$HOME/.codex}"
sessions="$root/sessions"
mkdir -p "$sessions" || exit 1
uuid=$(uuidgen 2>/dev/null | tr 'ABCDEF' 'abcdef')
if [ -z "$uuid" ]; then
  h=$(od -An -N16 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n')
  if [ -n "$h" ]; then
    uuid="$(printf %s "$h" | cut -c1-8)-$(printf %s "$h" | cut -c9-12)-4$(printf %s "$h" | cut -c14-16)-8$(printf %s "$h" | cut -c18-20)-$(printf %s "$h" | cut -c21-32)"
  fi
fi
[ -n "$uuid" ] || uuid="5f0a1c00-0000-4000-8000-00000000f001"
stamp=$(date -u '+%Y-%m-%dT%H-%M-%S')
iso=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
cwd=$(pwd -P)
esc=$(printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | awk '{printf "%s\\n", $0}')
file="$sessions/rollout-$stamp-$uuid.jsonl"
{
  printf '{"type":"session_meta","payload":{"id":"%s","timestamp":"%s","cwd":"%s","title":"Handed off from Claude Code","git":{}}}\n' "$uuid" "$iso" "$cwd"
  printf '{"type":"event_msg","payload":{"type":"user_message","message":"%s"}}\n' "$esc"
} > "$file"
printf 'reinstate sandbox: codex started a new session from a structured handoff\n'
printf 'rollout: %s\n' "$file"
exit 0
`
}

// silentCodexShim launches and exits cleanly without writing a rollout. Copy it
// over <root>/bin/codex to demo the other handoff outcome: launched=true with
// destination.state="unresolved", exit 0.
func silentCodexShim(version string) string {
	return `#!/bin/sh
# Synthetic Codex CLI stand-in that deliberately leaves no rollout behind.
case "$1" in
  --version) printf '%s\n' 'codex-cli ` + version + `'; exit 0 ;;
esac
printf 'reinstate sandbox: codex ran but recorded nothing\n'
exit 0
`
}

// writeWindowsShims is DESIGN, NOT VERIFIED. No researcher had a Windows host,
// and the repo's own proven CI fixture only ever handles --version. Two things
// are known to matter: `echo … (Claude Code)` must not sit inside a
// parenthesised if-block (the closing paren terminates the block, hence the
// goto form), and the file needs CRLF line endings and plain ASCII. The Windows
// handoff path is additionally untested — Reinstate substitutes a single-line
// short bootstrap there rather than hand CR/LF to a batch file, so the bytes the
// reconciler hashes differ from the Unix case.
func writeWindowsShims(set shimSet) error {
	claude := func(version string) string {
		return crlf(`@echo off
if "%~1"=="--version" goto version
echo reinstate sandbox: claude %*
exit /b 0
:version
echo ` + version + ` (Claude Code)
exit /b 0
`)
	}
	codex := crlf(`@echo off
if "%~1"=="--version" goto version
echo reinstate sandbox: codex %*
exit /b 0
:version
echo codex-cli ` + set.codex + `
exit /b 0
`)
	if err := writeFile(filepath.Join(set.binDir, "claude.cmd"), claude(set.claude), 0o755); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(set.staleDir, "claude.cmd"), claude(set.staleClaude), 0o755); err != nil {
		return err
	}
	return writeFile(filepath.Join(set.binDir, "codex.cmd"), codex, 0o755)
}

func crlf(body string) string {
	return strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
}
