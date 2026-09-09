package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/fsx"
)

// AgentRootsPath is where `rein daemon install` persists the agent-root
// environment it resolved: the per-agent RootEnv variables (CLAUDE_CONFIG_DIR,
// CODEX_HOME, XDG_DATA_HOME, and the rest the catalog declares) that were set
// in the installing process's own environment. A scheduled or supervised
// daemon run applies this file at startup instead of trusting whatever its
// launch context happens to carry (#424): Windows Task Scheduler has no
// per-action environment block (unlike launchd's EnvironmentVariables or
// systemd's Environment=), so this file is the daemon's own equivalent, and
// every platform reads the same file so behavior does not fork by OS.
func AgentRootsPath(home string) string { return filepath.Join(Dir(home), "agent-roots.json") }

// AgentRoots is the recorded agent-root environment: variable name to value,
// for every variable that had a non-blank value. A variable absent from the
// map was unset (or blank, which every reader in this codebase treats the
// same) when it was recorded.
type AgentRoots map[string]string

// agentRootsVersion is AgentRoots' on-disk envelope format.
const agentRootsVersion = 1

type agentRootsFile struct {
	Version int        `json:"version"`
	Roots   AgentRoots `json:"roots"`
}

// ErrNoAgentRoots reports that no install under this fix has recorded an
// agent-root baseline for this home: an install from before #424 shipped, or
// a `rein daemon run` with no install at all. Callers must treat this as
// "nothing to compare or pin", never as a recorded empty set.
var ErrNoAgentRoots = errors.New("no recorded agent-root environment")

// WriteAgentRoots persists roots for home, atomically and owner-only, next
// to the daemon's other per-home state (status.json, daemon.log).
func WriteAgentRoots(home string, roots AgentRoots) error {
	if roots == nil {
		roots = AgentRoots{}
	}
	if err := os.MkdirAll(Dir(home), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(agentRootsFile{Version: agentRootsVersion, Roots: roots}, "", "  ")
	if err != nil {
		return err
	}
	path := AgentRootsPath(home)
	tmp, err := os.CreateTemp(Dir(home), ".agent-roots-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = fsx.ProtectOwnerOnly(tmpPath, false)
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// ReadAgentRoots loads what WriteAgentRoots persisted for home. A home that
// was never installed under this fix (or never installed at all) reports
// ErrNoAgentRoots, not an empty AgentRoots.
func ReadAgentRoots(home string) (AgentRoots, error) {
	raw, err := os.ReadFile(AgentRootsPath(home))
	if os.IsNotExist(err) {
		return nil, ErrNoAgentRoots
	}
	if err != nil {
		return nil, err
	}
	var f agentRootsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", AgentRootsPath(home), err)
	}
	if f.Roots == nil {
		f.Roots = AgentRoots{}
	}
	return f.Roots, nil
}

// RemoveAgentRoots deletes the recorded baseline; a missing file is not an
// error. `rein daemon uninstall` calls this so a later install starts clean.
func RemoveAgentRoots(home string) error {
	if err := os.Remove(AgentRootsPath(home)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ResolveAgentRoots reads names from getenv (typically os.Getenv) and
// returns only the ones with a non-blank value, trimmed. This mirrors every
// other agent-root read in the codebase (adapter/claude, adapter/codex,
// adapter/opencode, preflight: `strings.TrimSpace(os.Getenv(...)) != ""`),
// so a variable that is unset and one that is merely blank compare equal
// here exactly as they do everywhere else a root variable is read — neither
// looks like a recorded override.
func ResolveAgentRoots(names []string, getenv func(string) string) AgentRoots {
	roots := AgentRoots{}
	for _, name := range names {
		if value := strings.TrimSpace(getenv(name)); value != "" {
			roots[name] = value
		}
	}
	return roots
}

// AgentRootsDiff is one variable whose recorded and current value disagree.
// An empty Recorded or Current means that side was unset (or blank).
type AgentRootsDiff struct {
	Name     string
	Recorded string
	Current  string
}

// CompareAgentRoots reports every variable named in recorded or current
// whose value disagrees, sorted by name. An empty result means the
// environment a process would resolve right now matches what was recorded.
func CompareAgentRoots(recorded, current AgentRoots) []AgentRootsDiff {
	names := make(map[string]struct{}, len(recorded)+len(current))
	for name := range recorded {
		names[name] = struct{}{}
	}
	for name := range current {
		names[name] = struct{}{}
	}
	diffs := make([]AgentRootsDiff, 0, len(names))
	for name := range names {
		if rv, cv := recorded[name], current[name]; rv != cv {
			diffs = append(diffs, AgentRootsDiff{Name: name, Recorded: rv, Current: cv})
		}
	}
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Name < diffs[j].Name })
	return diffs
}

// ApplyAgentRoots pins the process environment for every name in names to
// exactly what roots recorded: setenv for a name present in roots, unsetenv
// for a name absent from it. names should be the full catalog of agent-root
// variables (every RootEnv name the caller's agent catalog declares), not
// just the ones present in roots — that is what makes this a *pin* rather
// than a partial overlay: a variable nobody customized at install time
// (commonly CODEX_HOME, say) is pinned to unset exactly as firmly as one
// that was customized is pinned to its recorded value, regardless of
// whatever the process's own launch environment happens to carry for it.
//
// This matters specifically for `rein daemon run --allow-root-change`
// (reinstate#424): the flag only lifts the startup refusal, it does not
// adopt drift, so every catalog variable not in roots must end up unset
// here even if the calling process currently has it set to something the
// install-time environment never saw — otherwise the daemon would silently
// watch whatever root its drifted launch context happened to carry, the
// exact class of bug #424 was filed to close, just triggered by a variable
// going from unset to set instead of the reverse.
func ApplyAgentRoots(roots AgentRoots, names []string, setenv func(string, string) error, unsetenv func(string) error) error {
	sorted := append([]string(nil), names...)
	sort.Strings(sorted)
	for _, name := range sorted {
		if value, ok := roots[name]; ok {
			if err := setenv(name, value); err != nil {
				return fmt.Errorf("set %s: %w", name, err)
			}
			continue
		}
		if err := unsetenv(name); err != nil {
			return fmt.Errorf("unset %s: %w", name, err)
		}
	}
	return nil
}
