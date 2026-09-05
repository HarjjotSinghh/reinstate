package main

import (
	"fmt"
	"io"
	"strings"
)

// envPair is one KEY=VALUE a client shell needs.
type envPair struct{ Key, Value string }

// hopLabEnv is the block a client shell needs to reach a running lab and
// act as one device: REINSTATE_HOP_URL (D1: the control-plane URL is
// always configurable, see ADR 0005) plus the isolated home a `hoplab
// homes` call built.
func hopLabEnv(s LabState, h DeviceHome) []envPair {
	return []envPair{
		{"REINSTATE_HOP_URL", s.HopdBaseURL},
		{"REINSTATE_HOME", h.ReinstateHome},
		{"CLAUDE_CONFIG_DIR", h.ClaudeConfigDir},
		{"CODEX_HOME", h.CodexHome},
		{"XDG_DATA_HOME", h.XDGDataHome},
	}
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
