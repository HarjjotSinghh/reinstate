package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// envPair is one KEY=VALUE a client shell needs.
type envPair struct{ Key, Value string }

// extraCatalogRootEnv names the RootEnv variable and the natural (Grok's
// own home-fallback subdirectory, so a future `hoplab homes` that wants to
// seed one of these agents too has somewhere sensible to write) default
// path, for every internal/agents/catalog descriptor that (a) declares a
// NewIndexSource, so it contributes to `rein sessions`, and (b) is not one
// of Claude/Codex/OpenCode, whose RootEnv this package already sets
// directly. Read from each catalog file's own Roots func -- grok.go,
// gemini.go, kimi.go (its primary root only; the legacy ~/.kimi mirror
// needs no separate isolation, since KIMI_CODE_HOME alone already wins the
// hometree.ResolveRoot check ahead of every Candidates entry), qwen.go,
// cline.go, copilot.go, cursor.go, pi.go.
var extraCatalogRootEnv = []struct{ Env, Subdir string }{
	{"GROK_HOME", ".grok"},
	{"GEMINI_CLI_HOME", ".gemini"},
	{"KIMI_CODE_HOME", ".kimi-code"},
	{"QWEN_HOME", ".qwen"},
	{"CLINE_DATA_DIR", filepath.Join(".cline", "data")},
	{"COPILOT_HOME", ".copilot"},
	{"CURSOR_CONFIG_DIR", ".cursor"},
	{"PI_CODING_AGENT_DIR", filepath.Join(".pi", "agent")},
}

// hopLabEnv is the block a client shell needs to reach a running lab and
// act as one device: REINSTATE_HOP_URL (D1: the control-plane URL is
// always configurable, see ADR 0005) plus the isolated home a `hoplab
// homes` call built.
//
// HOME and USERPROFILE are both set to h.Home, the device's whole isolated
// home directory -- not only REINSTATE_HOME/CLAUDE_CONFIG_DIR/CODEX_HOME/
// XDG_DATA_HOME. Without this, every catalog agent that falls back to a
// path under the real process home when its own RootEnv is unset (the
// eight named in extraCatalogRootEnv -- see homes.go's DeviceHome doc
// comment) reads the real host account's real session data instead,
// identically for every simulated device. This is the same contract
// scripts/tuisandbox's sandboxEnv already sets for its own single-home
// bench; setting both names (not just the one the running OS reads)
// matches that precedent and keeps the block portable if this ever runs a
// cross-platform tool that reads $HOME.
//
// extraCatalogRootEnv's eight variables are set explicitly, in addition to
// HOME/USERPROFILE, even though a Roots-func fallback under an isolated
// HOME already resolves to the same, empty, unmarked directory: an
// operator's own shell may already export one of these (a developer who
// also uses Grok, say), and RootEnv always wins over Candidates in
// hometree.ResolveRoot regardless of HOME. Explicit beats inherited.
func hopLabEnv(s LabState, h DeviceHome) []envPair {
	pairs := []envPair{
		{"REINSTATE_HOP_URL", s.HopdBaseURL},
		{"REINSTATE_HOME", h.ReinstateHome},
		{"HOME", h.Home},
		{"USERPROFILE", h.Home},
		{"CLAUDE_CONFIG_DIR", h.ClaudeConfigDir},
		{"CODEX_HOME", h.CodexHome},
		{"XDG_DATA_HOME", h.XDGDataHome},
	}
	for _, extra := range extraCatalogRootEnv {
		pairs = append(pairs, envPair{extra.Env, filepath.Join(h.Home, extra.Subdir)})
	}
	return pairs
}

// printEnv writes pairs as a shell fragment on w in the requested syntax
// ("sh" or "powershell"), matching scripts/tuisandbox's own emit format so
// `eval "$(hoplab env ...)"` and `hoplab env ... | Invoke-Expression` both
// work the same way that tool's output already does.
func printEnv(w io.Writer, shell string, pairs []envPair) {
	for _, p := range pairs {
		if shell == "powershell" {
			_, _ = fmt.Fprintf(w, "$env:%s = '%s'\n", p.Key, psQuote(p.Value))
		} else {
			_, _ = fmt.Fprintf(w, "export %s=%q\n", p.Key, p.Value)
		}
	}
}

func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

// ambientOverrideEnv lists REINSTATE_* environment variables that let an
// ordinary `rein` invocation route around whatever storage a device's own
// home actually configures: REINSTATE_BACKEND=memory makes
// internal/cli/commands_impl.go's openBackend use a local disk store
// before it even looks at cfg.Storage.Type, REINSTATE_MEMORY_BACKEND_DIR
// says where (internal/cli/commands_impl.go's memoryBackendRoot: "lets two
// homes share one store", which is exactly the problem when it is left set
// by accident rather than chosen on purpose), and the four REINSTATE_S3_*
// names are the BYO credential/endpoint fallback (same file, and
// internal/credentials.Resolve). None of them is a lab concept -- hoplab's
// own isolation is REINSTATE_HOME plus the fourteen other pairs hopLabEnv
// returns.
//
// Left set in an operator's shell from earlier, unrelated local testing
// (REINSTATE_BACKEND=memory backed by one long-lived shared directory is
// an ordinary way to run this project's own local e2e tests by hand, and
// persists across every new shell once set at the Windows user level),
// these silently redirect a hop-mode `rein login`/`init --hop`/`account
// init` away from the lab's real hopd and fakelocker entirely, onto
// whatever that shared directory already holds. See
// docs/testing/windows-acceptance-host.md's Hop lab section for the dated
// repro this produced on this exact host: a brand-new account against a
// brand-new hopd and fakelocker, `rein account status --json` still
// reporting a keyring that already held two devices, because a hop-mode
// config's storage prefix is always empty (openBackend deliberately scopes
// hop storage through the bucket the control plane names, not a shared
// prefix) and the leaked REINSTATE_MEMORY_BACKEND_DIR pointed every
// device -- and every account -- at the exact same flat "keyring.v1.json"
// file on disk, left over from unrelated earlier work.
//
// Every environment hoplab builds -- the block `hoplab env` prints and
// every subprocess pair.go launches itself (reinEnviron) -- clears all
// seven, unconditionally, regardless of what the operator's own shell
// carries.
var ambientOverrideEnv = []string{
	"REINSTATE_BACKEND",
	"REINSTATE_MEMORY_BACKEND_DIR",
	"REINSTATE_S3_ACCESS_KEY_ID",
	"REINSTATE_S3_SECRET_ACCESS_KEY",
	"REINSTATE_S3_ENDPOINT",
	"REINSTATE_S3_BUCKET",
	"REINSTATE_S3_REGION",
}

// printEnvClear writes an unset line (shell-appropriate) for each of keys,
// meant to run right after printEnv's export lines in the same block --
// see ambientOverrideEnv for what these are and why a lab environment must
// clear them explicitly rather than merely not set them itself.
func printEnvClear(w io.Writer, shell string, keys []string) {
	for _, k := range keys {
		if shell == "powershell" {
			_, _ = fmt.Fprintf(w, "Remove-Item Env:%s -ErrorAction SilentlyContinue\n", k)
		} else {
			_, _ = fmt.Fprintf(w, "unset %s\n", k)
		}
	}
}

// stripEnv drops every "KEY=value" entry in base whose key is in keys,
// regardless of value. Unlike mergeEnv's overlay (pair.go), which leaves
// an override alone when its own value is empty (so it does not clobber
// whatever the base environment already had), stripEnv unconditionally
// removes the named keys -- how hoplab clears an ambient variable it wants
// gone rather than merely absent from what it itself sets. See
// ambientOverrideEnv.
func stripEnv(base []string, keys []string) []string {
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	out := make([]string, 0, len(base))
	for _, kv := range base {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if drop[key] {
			continue
		}
		out = append(out, kv)
	}
	return out
}
