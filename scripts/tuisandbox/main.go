// Command tuisandbox builds an isolated, synthetic agent home so Reinstate's
// interactive surfaces can be exercised by hand without ever touching the
// developer's real ~/.claude, ~/.codex, or ~/.grok.
//
// It is a demo bench, not a unit test: it plants one session per readiness
// state, with the real mechanism behind each state, so a human can walk the
// switcher, the acknowledgement checklist, resume, fork, and handoff end to
// end and see every branch of the environment model fire.
//
//	go run ./scripts/tuisandbox -root /tmp/rein-tui
//	eval "$(go run ./scripts/tuisandbox -root /tmp/rein-tui)"
//	./bin/rein
//
// Everything it writes is synthetic. Nothing is read from the host except the
// versions of git and the Go toolchain.
//
// # The readiness model this bench demonstrates
//
// A row's on-screen state is decided in two steps:
//
//  1. internal/tui/readiness/prober.go short-circuits to CANNOT RESUME when the
//     record itself is read-only (ReadOnlyReason set, or CanResume false),
//     WITHOUT running preflight at all. That is the B1 row.
//  2. Otherwise the preflight decision maps 1:1 to the label:
//     zero warnings and zero blocks -> READY TO RESUME
//     one or more warnings, no blocks -> NEEDS ACKNOWLEDGEMENT
//     one or more blocks -> CANNOT RESUME
//
// So there are exactly three levers: make the record read-only, add a warning
// check, or add a blocking check. Every row below picks one deliberately.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func main() {
	root := flag.String("root", "", "sandbox root directory (required, must be outside any Git repository)")
	shell := flag.String("shell", defaultShell(), "environment syntax: sh or powershell")
	scripted := flag.Bool("scripted", false,
		"also export REINSTATE_ALLOW_NON_TTY_LAUNCH=1 so launches work without a TTY (off by default: a hand-driven demo should behave like production)")
	stale := flag.Bool("stale-claude", false,
		"put the out-of-range Claude shim first on PATH, so every Claude row blocks on agent.version")
	flag.Parse()
	if strings.TrimSpace(*root) == "" {
		_, _ = fmt.Fprintln(os.Stderr, "tuisandbox: -root is required")
		os.Exit(2)
	}
	if err := run(*root, *shell, *scripted, *stale); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "tuisandbox:", err)
		os.Exit(1)
	}
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "sh"
}

// state is the readiness verdict a row is BUILT to produce. It is an intent,
// not an observation: the point of the bench is that running rein against it
// reproduces exactly this.
type state int

const (
	stateReady state = iota
	stateWarn
	stateBlocked
)

func (s state) String() string {
	switch s {
	case stateReady:
		return "READY"
	case stateWarn:
		return "WARN"
	default:
		return "BLOCKED"
	}
}

// session is one synthetic conversation planted in the sandbox, together with
// the levers that give it its intended readiness state.
type session struct {
	label   string // R1, W2, B3 … used only in the printed legend
	agent   string
	id      string
	project string // also the workspace directory name: one directory per row
	branch  string // the branch the SESSION FILE records
	title   string
	age     time.Duration
	want    state
	why     string // one-line explanation for the legend

	// Levers. Each one maps to exactly one check in the environment report.
	liveBranch   string // check out this instead of branch      -> git.branch warning
	detach       bool   // detach HEAD after committing          -> git.branch warning (permanent)
	untracked    string // leave this file untracked             -> git.working_tree warning
	nvmrc        string // committed .nvmrc content              -> runtime.node.declaration warning
	gomod        string // committed go.mod content              -> runtime.go.declaration warning
	codexHead    string // codex session_meta git.commit_hash    -> git.head warning (codex only)
	repoURL      string // codex session_meta git.repository_url -> git.repository BLOCK (exit 7)
	missingWS    bool   // never create the workspace            -> workspace.available BLOCK (exit 5)
	seedBaseline bool   // store a prelaunch baseline            -> clears baseline.unavailable
}

func (s session) reference() string { return s.agent + ":" + s.id }

// sessions is the bench. Nine rows: two READY, four NEEDS ACKNOWLEDGEMENT, three
// CANNOT RESUME. Ages span the same time buckets the switcher groups and colours
// by, from "4m" through "70d", so grouping and truncation stay exercised too.
//
// Every project name is distinct, which means every row gets its OWN workspace
// directory. That is not cosmetic: the dirty-tree and branch-drift levers are
// properties of a directory, so two rows sharing one would leak warnings into
// the row that is supposed to be clean.
var sessions = []session{
	// ---------------------------------------------------------------- READY --
	{
		label: "R1", agent: sessionindex.AgentClaude,
		id:      "5f0a1c00-0000-4000-8000-000000000001",
		project: "auth-refactor", branch: "feat/auth-refactor",
		title: "Fix the auth refactor so tokens survive a restart",
		age:   4 * time.Minute, want: stateReady, seedBaseline: true,
		// Nothing is wrong, and — critically — a baseline exists. Clean tree,
		// branch matches, no runtime declaration, no capability file, no remote,
		// workspace == repo root, and the claude shim prints an in-range
		// version. This is the row the READY path is demonstrated on.
		why: "seeded baseline, clean repo, branch matches",
	},
	{
		label: "R2", agent: sessionindex.AgentCodex,
		id:      "5f0a1c00-0000-4000-8000-000000000002",
		project: "keyring-store", branch: "feat/keyring-store",
		title: "Wire the keyring store behind the credential ref",
		age:   55 * time.Minute, want: stateReady, seedBaseline: true,
		// Same construction on the Codex side, to prove READY is not a
		// Claude-specific accident. Note the session_meta git block carries a
		// branch and NOTHING ELSE: a commit_hash or repository_url here would
		// immediately cost it the READY state.
		why: "seeded baseline, clean repo, branch matches (codex side)",
	},

	// --------------------------------------------- NEEDS ACKNOWLEDGEMENT --
	{
		label: "W1", agent: sessionindex.AgentClaude,
		id:      "5f0a1c00-0000-4000-8000-000000000003",
		project: "website", branch: "fix/og-images",
		title: "Open graph image sizes are wrong on the pricing page",
		age:   3 * time.Hour, want: stateWarn,
		liveBranch: "main", untracked: "scratch.txt", nvmrc: "18.20.4\n",
		// Four independent warnings at once, which is what makes the
		// acknowledgement checklist worth opening: baseline.unavailable (no
		// baseline seeded), git.branch (recorded fix/og-images, checked out
		// main), git.working_tree (an untracked file — untracked counts as
		// dirty), runtime.node.declaration (a pinned Node the host will not
		// have). Each must be acknowledged by exact ID.
		why: "no baseline + branch drift + untracked file + .nvmrc (4 warnings)",
	},
	{
		label: "W2", agent: sessionindex.AgentCodex,
		id:      "5f0a1c00-0000-4000-8000-000000000008",
		project: "search-index", branch: "main",
		title: "Try a smaller embedding model for the search index",
		age:   70 * 24 * time.Hour, want: stateWarn,
		codexHead: "1111111111111111111111111111111111111111",
		// A DIFFERENT warning mix, using the two levers only Codex has. The
		// recorded commit_hash is 40 lower-hex that is not the live HEAD, so
		// git.head warns — the Claude reader records no HEAD at all, so that
		// warning is unreachable from a Claude row. The branch deliberately
		// MATCHES, to show the checks are independent.
		//
		// Trap: go.mod's `go` directive is a MINIMUM constraint. `go 1.19.0`
		// is satisfied by any modern toolchain and produces no warning at all.
		// It has to be a version from the future.
		gomod: "module reinstate-demo\n\ngo 1.99.0\n",
		why:   "no baseline + recorded HEAD mismatch + future go directive (3 warnings)",
	},
	{
		label: "W3", agent: sessionindex.AgentClaude,
		id:      "5f0a1c00-0000-4000-8000-000000000007",
		project: "deps-bump", branch: "chore/deps",
		title: "Bump the pinned linter and regenerate the lockfile",
		age:   9 * 24 * time.Hour, want: stateWarn,
		// Exactly one warning: baseline.unavailable, and nothing else. This is
		// the best beat in the bench. Acknowledge it once, let the launch exit
		// cleanly, and the CLI persists the baseline itself — the row is READY
		// on the next visit, and a following `rein fork` needs no
		// acknowledgement at all. It is the only row that heals itself.
		why: "no baseline only (1 warning; flips to READY after one acknowledged resume)",
	},
	{
		label: "W4", agent: sessionindex.AgentClaude,
		id:      "5f0a1c00-0000-4000-8000-000000000009",
		project: "checkout-flow", branch: "fix/checkout",
		title: "Checkout throws when the cart has a single item",
		age:   6 * time.Hour, want: stateWarn,
		detach: true, seedBaseline: true,
		// The counterpart to W3: a row that stays amber FOREVER. Its baseline is
		// already seeded, so baseline.unavailable is gone and only git.branch
		// remains — and a detached HEAD cannot be baselined away, because
		// BaselineFromReport records no branch for a detached workspace and the
		// expectation falls straight back to the vendor-recorded value. Without
		// this row the bench has no amber left once W3 has been demonstrated.
		why: "seeded baseline + detached HEAD (1 warning that re-baselining can never clear)",
	},

	// -------------------------------------------------------- CANNOT RESUME --
	{
		label: "B1", agent: sessionindex.AgentGrok,
		id:      "5f0a1c00-0000-4000-8000-00000000000b",
		project: "payment-adapter", branch: "main",
		title: "Trace the flaky checkout test through the payment adapter",
		age:   30 * time.Hour, want: stateBlocked,
		// Blocked WITHOUT preflight. The Grok source hardcodes CanResume:false
		// and a read-only reason, and the readiness prober short-circuits on
		// that before any probe runs. Its workspace is a perfectly healthy Git
		// repository precisely to make the point: nothing about the machine is
		// consulted.
		//
		// Known TUI defect this row exposes: the switcher's "read-only: reason"
		// preview line sits behind an `else if m.readiness == nil` guard, and
		// the prober is never nil for interactive rein — so this row reads
		// CANNOT RESUME with no reason anywhere on screen. `rein inspect` shows
		// it. That is a TUI fix, not a generator one.
		why: "read-only agent record; preflight never runs",
	},
	{
		label: "B2", agent: sessionindex.AgentClaude,
		id:      "5f0a1c00-0000-4000-8000-000000000004",
		project: "gone-missing", branch: "main",
		title: "Probe the vendor session paths on Windows",
		age:   28 * time.Hour, want: stateBlocked, missingWS: true,
		// The session records a workspace the generator deliberately never
		// creates: workspace.available is StatusMissing / SeverityBlock, exit 5.
		// agent.executable blocks too, as a side effect — the vendor binary is
		// resolved relative to the workspace, and there is no workspace.
		//
		// Do not assert exit 5 unconditionally in any self-check: the two
		// StatusError variants of the same check ("could not be inspected",
		// "changed while it was inspected") block with exit 1 instead.
		why: "recorded workspace does not exist (block, exit 5)",
	},
	{
		label: "B3", agent: sessionindex.AgentCodex,
		id:      "5f0a1c00-0000-4000-8000-000000000006",
		project: "repo-drift", branch: "main",
		title: "認証リファクタを修正する", // keeps the CJK column-width case in the bench
		age:   4 * 24 * time.Hour, want: stateBlocked,
		repoURL: "https://github.com/example/other.git",
		// Purely metadata-driven, and the most legible blocked row here: a clean
		// repo on the right branch, with a session that claims to have come from
		// a different repository. Repository identity is the one recorded field
		// Reinstate refuses to launch through — git.repository is
		// StatusChanged / SeverityBlock, exit 7. The workspace has no remote, so
		// its live identity is derived from the root commit (roots-sha256:…) and
		// cannot possibly match the recorded URL.
		why: "session records a foreign repository_url (block, exit 7)",
	},
}

func run(rootFlag, shell string, scripted, stale bool) error {
	root, err := filepath.Abs(rootFlag)
	if err != nil {
		return err
	}
	// Refuse a root inside a checkout. This is not fussiness: executabletrust
	// treats the OUTERMOST .git-marked ancestor of the workspace as a trust
	// boundary and drops every PATH entry beneath it, so <root>/bin/claude
	// would be unresolvable and agent.executable would block all nine rows for
	// a reason nothing on screen explains.
	if marker, inside := gitAncestor(root); inside {
		_, _ = fmt.Fprintf(os.Stderr,
			"tuisandbox: -root %s is inside the Git repository at %s\n"+
				"  The sandbox PATH entry would fall inside that repository's executable trust\n"+
				"  boundary and every session would block on agent.executable.\n"+
				"  Use a path outside any checkout, for example %s\n",
			root, marker, suggestedRoot())
		os.Exit(2)
	}
	if !gitAvailable() {
		return fmt.Errorf("git is required: every readiness state above blocked needs a real repository")
	}

	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	// Resolve symlinks now that the root exists, so the paths printed here are
	// the same ones the child processes and the workspace probe see. On macOS
	// /tmp is a symlink to /private/tmp, and having the two forms disagree makes
	// every reported path confusing to compare by hand.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	home := filepath.Join(root, "home")
	reinstateHome := filepath.Join(root, "reinstate")
	for _, dir := range []string{home, reinstateHome} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}

	shims, err := writeShims(root)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var toSeed []string
	for _, s := range sessions {
		workspace := filepath.Join(home, "Projects", s.project)
		if err := buildWorkspace(workspace, s); err != nil {
			return err
		}
		when := now.Add(-s.age)
		switch s.agent {
		case sessionindex.AgentClaude:
			err = writeClaude(home, workspace, s, when)
		case sessionindex.AgentCodex:
			err = writeCodex(home, workspace, s, when)
		case sessionindex.AgentGrok:
			err = writeGrok(home, workspace, s, when)
		default:
			err = fmt.Errorf("unknown agent %q", s.agent)
		}
		if err != nil {
			return err
		}
		if s.seedBaseline {
			toSeed = append(toSeed, s.reference())
		}
	}

	// Point the GENERATOR's own process at the sandbox before any index or
	// preflight work. preflight.DefaultService captures CLAUDE_CONFIG_DIR,
	// CODEX_HOME and the user home at construction time, and the version probe
	// resolves the shims off PATH — doing this later would snapshot the
	// developer's real vendor trees into the bench's baselines.
	for _, pair := range sandboxEnv(root, home, reinstateHome) {
		if err := os.Setenv(pair[0], pair[1]); err != nil {
			return err
		}
	}
	if err := os.Setenv("PATH", pathPrefix(root, stale)+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
		return err
	}

	if len(toSeed) > 0 {
		if err := seedBaselines(context.Background(), reinstateHome, toSeed); err != nil {
			return err
		}
	}

	printEnv(shell, root, home, reinstateHome, scripted, stale)
	printLegend(root, home, shims, scripted, stale)
	return nil
}

// gitAncestor walks upward looking for a .git marker, mirroring what
// executabletrust.trustBoundary does. It reports the first marked ancestor;
// executabletrust keeps walking to find the outermost one, but for a refusal
// check any hit is disqualifying.
func gitAncestor(path string) (string, bool) {
	current := filepath.Clean(path)
	for {
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func suggestedRoot() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "rein-tui")
	}
	return "/tmp/rein-tui"
}

// sandboxEnv is the isolation contract, and every entry is load-bearing.
// HOME/USERPROFILE redirect agent discovery AND home-scope capability
// discovery, which reads <home>/.claude/settings.json and <home>/.claude.json —
// drop them and the bench reports on the developer's real configuration.
// GROK_HOME is deliberately absent: the Grok source falls back to $HOME/.grok.
func sandboxEnv(root, home, reinstateHome string) [][2]string {
	return [][2]string{
		{"HOME", home},
		{"USERPROFILE", home},
		{"REINSTATE_HOME", reinstateHome},
		{"CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude")},
		{"CODEX_HOME", filepath.Join(home, ".codex")},
	}
}

func pathPrefix(root string, stale bool) string {
	if stale {
		// bin-stale first, bin second: the stale claude wins, and codex still
		// resolves from bin.
		return filepath.Join(root, "bin-stale") + string(os.PathListSeparator) + filepath.Join(root, "bin")
	}
	return filepath.Join(root, "bin")
}

// printEnv writes the shell fragment to STDOUT, which is the only thing on
// stdout, so `eval "$(...)"` works.
//
// PATH is the piece the previous generator was missing, and without it nothing
// else matters: the eval'd shell never sees <root>/bin, so agent.executable
// blocks every row with exit 5. It must be PREPENDED, never replaced, for two
// reasons — git itself is resolved through executabletrust, so it needs to
// stay on an absolute PATH entry outside the trust boundary or every git.*
// check degrades; and the launched shim inherits the full parent environment
// and needs date/sed/awk/mkdir for the structured-handoff path.
func printEnv(shell, root, home, reinstateHome string, scripted, stale bool) {
	pairs := sandboxEnv(root, home, reinstateHome)
	for _, pair := range pairs {
		emit(shell, pair[0], pair[1])
	}
	prefix := pathPrefix(root, stale)
	if shell == "powershell" {
		fmt.Printf("$env:PATH = '%s' + [IO.Path]::PathSeparator + $env:PATH\n", psQuote(prefix))
	} else {
		fmt.Printf("export PATH=%q:\"$PATH\"\n", prefix)
	}
	if scripted {
		// Off by default on purpose: a hand-driven demo should hit the same
		// TTY requirement production does. RequireInteractiveTerminal demands
		// that BOTH stdin and stdout be terminals, so any scripted smoke test
		// needs this.
		emit(shell, "REINSTATE_ALLOW_NON_TTY_LAUNCH", "1")
	}
}

func emit(shell, key, value string) {
	if shell == "powershell" {
		fmt.Printf("$env:%s = '%s'\n", key, psQuote(value))
		return
	}
	fmt.Printf("export %s=%q\n", key, value)
}

// psQuote escapes for a PowerShell single-quoted string, which takes no
// backslash escapes at all — which is what makes it the right quoting for
// Windows paths. Only the quote itself needs doubling.
func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// printLegend writes to STDERR so it survives `eval "$(...)"` and tells the
// driver which row is which, why, and what to type next.
func printLegend(root, home string, shims shimSet, scripted, stale bool) {
	w := os.Stderr
	line(w, "\nReinstate TUI bench at %s\n", root)
	line(w, "  vendor shims: %s  (claude %s, codex %s)\n", shims.binDir, shims.claude, shims.codex)
	if stale {
		line(w, "  -stale-claude ACTIVE: %s is first on PATH (claude %s, out of range)\n",
			shims.staleDir, shims.staleClaude)
	}
	if scripted {
		lineln(w, "  -scripted ACTIVE: REINSTATE_ALLOW_NON_TTY_LAUNCH=1 is exported")
	}

	lineln(w, "\n  STATE     REFERENCE                                 WHY")
	for _, s := range sessions {
		line(w, "  %-9s %-41s %s\n", s.want, s.reference(), s.why)
	}

	r1, w1, w3, b1 := byLabel("R1"), byLabel("W1"), byLabel("W3"), byLabel("B1")
	lineln(w, "\n  Try it:")
	lineln(w, "    ./bin/rein                                  # the switcher: 3 glyphs, 9 rows")
	lineln(w, "    ./bin/rein sessions --json | jq .")
	line(w, "    ./bin/rein resume %s --dry-run --json | jq .environment.decision\n", r1)
	line(w, "    ./bin/rein inspect %s --json | jq .session.read_only_reason\n", b1)
	line(w, "    ./bin/rein resume %s --dry-run --json \\\n", w1)
	lineln(w, "        | jq '[.environment.checks[]|select(.severity==\"warning\")|.id]'")
	lineln(w, "\n  The self-healing row — acknowledge once, then watch it turn green:")
	line(w, "    ./bin/rein resume %s --allow-environment-warning baseline.unavailable\n", w3)
	line(w, "    ./bin/rein resume %s --dry-run --json | jq .environment.decision\n", w3)
	lineln(w, "\n  Handoff (run it from the workspace: rein refuses a cwd in a different repo,")
	lineln(w, "  and baseline.unavailable is ALWAYS a warning on a handoff — it cannot be cleared):")
	line(w, "    cd %q\n", filepath.Join(home, "Projects", "auth-refactor"))
	line(w, "    ./bin/rein handoff %s --to codex \\\n", r1)
	lineln(w, "        --allow-warning baseline.unavailable \\")
	lineln(w, "        --allow-warning handoff.capability.attachment.support")
	lineln(w, "    ./bin/rein handoff list --json | jq '.[-1].destination'")
	if !scripted {
		lineln(w, "\n  Non-interactive launches need -scripted (exports REINSTATE_ALLOW_NON_TTY_LAUNCH=1).")
	}
	lineln(w)
}

// line and lineln write one legend line. The legend is advisory text on
// stderr; if that write fails there is nothing sensible to do about it and
// nothing depends on it having succeeded, so the error is dropped here, once,
// rather than at every call site.
func line(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func lineln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}

func byLabel(label string) string {
	for _, s := range sessions {
		if s.label == label {
			return s.reference()
		}
	}
	return ""
}
