package main

import (
	"context"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

// scanDeviceAsRein runs exactly the source-gathering loop `rein sessions`
// runs (internal/cli/sessions.go's defaultLocalSources: every catalog
// descriptor with a NewIndexSource, called with an empty agents.Env{}, so
// each source resolves its root the same way the real RootEnv env vars and
// $HOME/%USERPROFILE% the caller exported determine, exactly as the real
// binary would after `eval "$(hoplab env ...)"`). It is the same seam a
// cold repro of the T-402 blocker used: build rein.exe, `hoplab homes`,
// eval the printed env for one device, run `rein sessions --json`. This
// helper does the same walk in-process, without a build step, so the
// regression stays fast and needs no `go build`.
func scanDeviceAsRein(t *testing.T) []sessionindex.Record {
	t.Helper()
	var all []sessionindex.Record
	for _, descriptor := range agents.Capable(agents.CapabilityIndex) {
		if descriptor.NewIndexSource == nil {
			continue
		}
		source, err := descriptor.NewIndexSource(agents.Env{})
		if err != nil || source == nil {
			continue
		}
		result, err := source.Scan(context.Background())
		if err != nil {
			t.Fatalf("%s: Scan: %v", descriptor.Key, err)
		}
		all = append(all, result.Records...)
	}
	return all
}

// setDeviceEnv exports exactly the block `hoplab env` prints for h, the way
// a caller's `eval "$(hoplab env ...)"` would, scoped to this test by
// t.Setenv (never REINSTATE_HOME/XDG_DATA_HOME/host-root env hygiene
// violations: everything here is torn down when the test ends).
func setDeviceEnv(t *testing.T, h DeviceHome) {
	t.Helper()
	for _, p := range hopLabEnv(LabState{}, h) {
		if p.Key == "REINSTATE_HOP_URL" {
			continue // not read by session scanning; LabState{} leaves it empty anyway
		}
		t.Setenv(p.Key, p.Value)
	}
}

// TestDeviceHomesDoNotLeakTheHostAccount is the regression for the T-402
// isolation blocker: with a seeded device's env exported, `rein sessions`
// (reproduced in-process by scanDeviceAsRein) must see exactly that
// device's three synthetic fixtures -- one Claude, one Codex, one
// OpenCode session, each under that device's own fixture project -- never
// the real host account's real sessions from the other eight catalog
// agents that have a NewIndexSource (Grok, Gemini, Kimi, Qwen, Cline,
// Copilot, Cursor, Pi), which is exactly what leaked when hopLabEnv left
// HOME/USERPROFILE unset: those eight fall back to a path under the real
// process home (internal/agents/scan/hometree.ResolveRoot) and reported
// whatever the real, ambient host account happened to have.
//
// A cold repro of the original bug (docs/testing/windows-acceptance-host.md,
// Hop lab section, 2026-09-05): 48 sessions for each of two "isolated"
// devices, 45 of them byte-identical between the two and none of them one
// of the 3 seeded fixtures.
func TestDeviceHomesDoNotLeakTheHostAccount(t *testing.T) {
	repoRoot, err := findRepoRoot("")
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	lab := t.TempDir()
	a := BuildDeviceHome(lab, "device-a")
	b := BuildDeviceHome(lab, "device-b")
	if err := a.Seed(repoRoot); err != nil {
		t.Fatalf("seed device-a: %v", err)
	}
	if err := b.Seed(repoRoot); err != nil {
		t.Fatalf("seed device-b: %v", err)
	}

	setDeviceEnv(t, a)
	recordsA := scanDeviceAsRein(t)
	if len(recordsA) != 3 {
		agentsSeen := map[string]int{}
		for _, r := range recordsA {
			agentsSeen[r.Agent]++
		}
		t.Fatalf("device-a: got %d sessions, want 3 (one claude, one codex, one opencode); by agent: %+v", len(recordsA), agentsSeen)
	}
	for _, r := range recordsA {
		if r.Workspace != a.Project {
			t.Fatalf("device-a session %s (%s) has workspace %q, want %q -- this is real host data, not the seeded fixture", r.ID, r.Agent, r.Workspace, a.Project)
		}
	}

	setDeviceEnv(t, b)
	recordsB := scanDeviceAsRein(t)
	if len(recordsB) != 3 {
		agentsSeen := map[string]int{}
		for _, r := range recordsB {
			agentsSeen[r.Agent]++
		}
		t.Fatalf("device-b: got %d sessions, want 3 (one claude, one codex, one opencode); by agent: %+v", len(recordsB), agentsSeen)
	}
	for _, r := range recordsB {
		if r.Workspace != b.Project {
			t.Fatalf("device-b session %s (%s) has workspace %q, want %q -- this is real host data, not the seeded fixture", r.ID, r.Agent, r.Workspace, b.Project)
		}
	}

	// The two devices must be distinguishable, not merely each individually
	// small: T-402's whole point is that pairing/revocation/lagging-device/
	// path-remap scenarios can tell device A and device B apart.
	seenA := map[string]bool{}
	for _, r := range recordsA {
		seenA[r.Key] = true
	}
	for _, r := range recordsB {
		if seenA[r.Key] {
			t.Fatalf("device-b session key %q also appeared under device-a; the two devices are not isolated", r.Key)
		}
	}
}
