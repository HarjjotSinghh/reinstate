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
			fmt.Fprintf(w, "$env:%s = '%s'\n", p.Key, psQuote(p.Value))
		} else {
			fmt.Fprintf(w, "export %s=%q\n", p.Key, p.Value)
		}
	}
}

func psQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
