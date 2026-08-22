// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package handoffui

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/tuitest"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// TestMain pins the rendering environment. lipgloss resolves its colour profile
// from stdout exactly once, on the first render, so forcing NO_COLOR before any
// frame is drawn makes a golden frame identical whether the suite runs under
// `go test` (piped stdout) or as a bare test binary in a colour terminal.
func TestMain(m *testing.M) {
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		fmt.Fprintln(os.Stderr, "set NO_COLOR:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// config describes one studio under test. The zero value is a 110x30 studio
// over three destinations, opening on the balanced policy, with every plan
// succeeding.
type config struct {
	width  int
	height int
	// destinations is nil for the default trio. An empty but non-nil slice
	// builds a studio with nowhere to send, which is a real state: a source
	// agent can be the only agent installed.
	destinations []string
	policy       string
	reference    string
	sourceAgent  string
	clipboard    tui.ClipboardFunc
	// fail maps previewKey(destination, policy) to a plan failure.
	fail map[string]error
	// components overrides the fidelity components in every plan.
	components []capsule.Component
	// noPlanner builds the studio without a planner at all.
	noPlanner bool
}

// failing builds a fail map for one destination and policy pair.
func failing(destination, policy string, err error) map[string]error {
	return map[string]error{previewKey(destination, policy): err}
}

// newStudio builds a studio over a fake planner and fills in the defaults.
func newStudio(t *testing.T, cfg *config) (*Model, *fakePlanner) {
	t.Helper()
	if cfg.width == 0 {
		cfg.width = 110
	}
	if cfg.height == 0 {
		cfg.height = 30
	}
	if cfg.destinations == nil {
		cfg.destinations = []string{destCodex, destGemini, destClaude}
	}
	if cfg.policy == "" {
		cfg.policy = string(handoff.PolicyBalanced)
	}
	if cfg.reference == "" {
		cfg.reference = fixtureReference
	}
	if cfg.sourceAgent == "" {
		cfg.sourceAgent = "claude"
	}
	capability := ui.Capability{
		Mode:    ui.ModeFull,
		Color:   ui.ColorNone,
		Unicode: true,
		Width:   cfg.width,
		Height:  cfg.height,
	}
	fake := newFakePlanner()
	fake.fail = cfg.fail
	fake.components = cfg.components

	var planner *Planner
	if !cfg.noPlanner {
		planner = NewPlanner(fake.plan)
	}
	model := New(Options{
		Theme:        ui.NewTheme(capability),
		Capability:   capability,
		Planner:      planner,
		Clipboard:    cfg.clipboard,
		Reference:    cfg.reference,
		SourceAgent:  cfg.sourceAgent,
		Destinations: cfg.destinations,
		Policy:       cfg.policy,
	})
	return model, fake
}

// start builds a studio and drives it through the deterministic harness, which
// runs Init and every command it schedules synchronously. By the first frame,
// the opening selection has already been measured.
func start(t *testing.T, cfg config) (*tuitest.Driver, *Model, *fakePlanner) {
	t.Helper()
	model, fake := newStudio(t, &cfg)
	return tuitest.New(t, model, cfg.width, cfg.height), model, fake
}

// pendingStudio returns the studio in the state a user sees before the first
// plan has landed: built and sized, but with Init's command not yet drained.
// tuitest.New drains it synchronously, which is exactly what this state needs
// to avoid.
func pendingStudio(t *testing.T, cfg config) *Model {
	t.Helper()
	model, _ := newStudio(t, &cfg)
	model.Update(tea.WindowSizeMsg{Width: cfg.width, Height: cfg.height})
	return model
}

func frameOf(m *Model) string { return tuitest.Normalize(m.View()) }

// assertZeroIntent checks a cancelled studio decided nothing at all. Intent
// carries a slice, so it cannot simply be compared against its zero value.
func assertZeroIntent(t *testing.T, intent tui.Intent) {
	t.Helper()
	if intent.Chosen() {
		t.Fatalf("intent = %+v, want no action", intent)
	}
	if intent.Action != tui.ActionNone || intent.Reference != "" ||
		intent.Destination != "" || intent.Policy != "" || len(intent.AcknowledgedWarnings) != 0 {
		t.Fatalf("intent = %+v, want the zero intent", intent)
	}
}

// assertFrameWidth is the layout contract: no rendered line may be wider than
// the terminal, measured in display cells rather than bytes or runes.
func assertFrameWidth(t *testing.T, frame string, width int) {
	t.Helper()
	for index, line := range strings.Split(frame, "\n") {
		if got := ui.Width(line); got > width {
			t.Errorf("line %d is %d cells wide, terminal is %d\n%q", index+1, got, width, line)
		}
	}
}

// TestPolicyCyclingWrapsInBothDirections walks the policy spectrum. The order
// is part of the surface: left and right move from least carried to most, and
// the ends join so a user never has to reverse to reach the third option.
func TestPolicyCyclingWrapsInBothDirections(t *testing.T) {
	if strings.Join(Policies, ",") != "checkpoint,balanced,full" {
		t.Fatalf("Policies = %v, want checkpoint, balanced, full in that order", Policies)
	}

	t.Run("right", func(t *testing.T) {
		driver, model, _ := start(t, config{policy: string(handoff.PolicyCheckpoint)})
		if got := model.Policy(); got != "checkpoint" {
			t.Fatalf("opening policy = %q, want checkpoint", got)
		}
		for step, want := range []string{"balanced", "full", "checkpoint", "balanced"} {
			driver.Key("right")
			if got := model.Policy(); got != want {
				t.Fatalf("right step %d selected %q, want %q", step+1, got, want)
			}
			wantCommand := "rein handoff " + fixtureReference + " --to " + destCodex + " --policy " + want
			if got := model.EquivalentCommand(); got != wantCommand {
				t.Fatalf("right step %d command = %q, want %q", step+1, got, wantCommand)
			}
			if frame := driver.View(); !strings.Contains(frame, wantCommand) {
				t.Fatalf("right step %d frame does not show %q\n%s", step+1, wantCommand, frame)
			}
		}
	})

	t.Run("left", func(t *testing.T) {
		driver, model, _ := start(t, config{policy: string(handoff.PolicyCheckpoint)})
		for step, want := range []string{"full", "balanced", "checkpoint", "full"} {
			driver.Key("left")
			if got := model.Policy(); got != want {
				t.Fatalf("left step %d selected %q, want %q", step+1, got, want)
			}
			if got := model.EquivalentCommand(); !strings.HasSuffix(got, " --policy "+want) {
				t.Fatalf("left step %d command = %q, want it to end in --policy %s", step+1, got, want)
			}
		}
	})

	t.Run("every policy is measured once and then cached", func(t *testing.T) {
		driver, _, fake := start(t, config{policy: string(handoff.PolicyCheckpoint)})
		driver.Keys("right", "right", "right", "left", "left", "left", "right")
		for _, policy := range Policies {
			if got := fake.callCount(destCodex, policy); got != 1 {
				t.Errorf("%s was planned %d times, want once", policy, got)
			}
		}
	})

	t.Run("an unknown opening policy falls back to balanced", func(t *testing.T) {
		_, model, _ := start(t, config{policy: "teleport-everything"})
		if got := model.Policy(); got != string(handoff.PolicyBalanced) {
			t.Fatalf("policy = %q, want balanced", got)
		}
	})
}

// TestDestinationCyclingWrapsAndSurvivesDegenerateLists covers the chooser and
// both list shapes that have no next entry to move to.
func TestDestinationCyclingWrapsAndSurvivesDegenerateLists(t *testing.T) {
	t.Run("tab", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		if got := model.Destination(); got != destCodex {
			t.Fatalf("opening destination = %q, want %q", got, destCodex)
		}
		for step, want := range []string{destGemini, destClaude, destCodex, destGemini} {
			driver.Key("tab")
			if got := model.Destination(); got != want {
				t.Fatalf("tab step %d selected %q, want %q", step+1, got, want)
			}
			if got := model.EquivalentCommand(); !strings.Contains(got, " --to "+want+" ") {
				t.Fatalf("tab step %d command = %q, want --to %s", step+1, got, want)
			}
		}
	})

	t.Run("down", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		for step, want := range []string{destGemini, destClaude, destCodex} {
			driver.Key("down")
			if got := model.Destination(); got != want {
				t.Fatalf("down step %d selected %q, want %q", step+1, got, want)
			}
		}
	})

	t.Run("up wraps backwards", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		for step, want := range []string{destClaude, destGemini, destCodex, destClaude} {
			driver.Key("up")
			if got := model.Destination(); got != want {
				t.Fatalf("up step %d selected %q, want %q", step+1, got, want)
			}
		}
	})

	t.Run("a single destination stays put", func(t *testing.T) {
		driver, model, _ := start(t, config{destinations: []string{destGemini}})
		driver.Keys("tab", "tab", "up", "down")
		if got := model.Destination(); got != destGemini {
			t.Fatalf("destination = %q, want %q", got, destGemini)
		}
		if frame := driver.View(); !strings.Contains(frame, "gemini") {
			t.Errorf("frame does not name the only destination\n%s", frame)
		}
	})

	t.Run("no destination at all", func(t *testing.T) {
		driver, model, fake := start(t, config{destinations: []string{}})
		if got := model.Destination(); got != "" {
			t.Fatalf("destination = %q, want empty", got)
		}
		driver.Keys("tab", "up", "down", "left", "right")
		if got := model.Destination(); got != "" {
			t.Fatalf("cycling an empty list selected %q", got)
		}
		if got := fake.totalCalls(); got != 0 {
			t.Fatalf("planned %d times with nowhere to send", got)
		}

		driver.Key("enter")
		if model.Intent().Chosen() {
			t.Fatal("enter confirmed a handoff with no destination")
		}
		if model.status == "" {
			t.Fatal("enter with no destination must explain itself")
		}
		// A refused send never leaves an export pending, whatever refused it.
		driver.Key("e")
		if model.Intent().Chosen() {
			t.Fatal("e confirmed an export with no destination")
		}
		if model.ExportRequested() {
			t.Fatal("a refused export left the export flag set")
		}
		frame := driver.View()
		if !strings.Contains(frame, "no destination agent available") {
			t.Errorf("frame does not say there is nowhere to send\n%s", frame)
		}
		if !strings.Contains(frame, model.status) {
			t.Errorf("status %q is not visible in the frame\n%s", model.status, frame)
		}
	})
}

// TestEquivalentCommandIsExact pins the flag form the studio teaches. A command
// a user copies must be the one that reproduces what they are looking at.
func TestEquivalentCommandIsExact(t *testing.T) {
	tests := []struct {
		name string
		cfg  config
		keys []string
		want string
	}{
		{
			name: "the opening selection",
			want: "rein handoff claude:5f1c0b2a3e7d --to codex --policy balanced",
		},
		{
			name: "one policy to the right",
			keys: []string{"right"},
			want: "rein handoff claude:5f1c0b2a3e7d --to codex --policy full",
		},
		{
			name: "one policy to the left",
			keys: []string{"left"},
			want: "rein handoff claude:5f1c0b2a3e7d --to codex --policy checkpoint",
		},
		{
			name: "the next destination",
			keys: []string{"tab"},
			want: "rein handoff claude:5f1c0b2a3e7d --to gemini --policy balanced",
		},
		{
			name: "destination and policy together",
			keys: []string{"tab", "tab", "left"},
			want: "rein handoff claude:5f1c0b2a3e7d --to claude --policy checkpoint",
		},
		{
			name: "a different source session",
			cfg:  config{reference: "codex:9b7d4118"},
			keys: []string{"right"},
			want: "rein handoff codex:9b7d4118 --to codex --policy full",
		},
		{
			// `a` first: an export is still a send, and a send with
			// unacknowledged warnings is refused, which clears the flag.
			name: "export adds --no-launch",
			keys: []string{"a", "e"},
			want: "rein handoff claude:5f1c0b2a3e7d --to codex --policy balanced --no-launch",
		},
		{
			name: "export keeps the current selection",
			keys: []string{"tab", "right", "a", "e"},
			want: "rein handoff claude:5f1c0b2a3e7d --to gemini --policy full --no-launch",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, model, _ := start(t, test.cfg)
			driver.Keys(test.keys...)
			// An acknowledged selection also carries one --allow-warning per
			// current warning, which is the whole point: the command shown is
			// exactly the command that would reproduce this handoff.
			want := test.want
			if model.Acknowledged() {
				for _, warning := range model.Warnings() {
					want += " --allow-warning " + warning
				}
			}
			if got := model.EquivalentCommand(); got != want {
				t.Fatalf("EquivalentCommand() = %q, want %q", got, want)
			}
			if model.quitting {
				return
			}
			if frame := driver.View(); !strings.Contains(frame, test.want) {
				t.Errorf("frame does not show the equivalent command\n%s", frame)
			}
		})
	}
}

// TestEnterConfirmsAndCancelKeysDoNot is the contract between the studio and its
// caller: the surface decides, the caller acts.
func TestEnterConfirmsAndCancelKeysDoNot(t *testing.T) {
	t.Run("enter confirms the current selection", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		driver.Keys("tab", "right")

		wantDestination, wantPolicy := model.Destination(), model.Policy()
		// Warnings on the current selection must be accepted first; the flag
		// path demands one --allow-warning per warning and the studio now
		// mirrors that instead of silently sending an unacknowledged handoff.
		if len(model.Warnings()) > 0 {
			driver.Key("a")
		}
		driver.Key("enter")

		intent := model.Intent()
		if !intent.Chosen() {
			t.Fatal("enter did not confirm")
		}
		want := tui.Intent{
			Action:      tui.ActionHandoff,
			Reference:   fixtureReference,
			Destination: wantDestination,
			Policy:      wantPolicy,
		}
		if intent.Action != want.Action || intent.Reference != want.Reference ||
			intent.Destination != want.Destination || intent.Policy != want.Policy {
			t.Fatalf("intent = %+v, want %+v", intent, want)
		}
		if len(intent.AcknowledgedWarnings) != len(model.Warnings()) {
			t.Fatalf("intent acknowledged %v; it must carry exactly the current warning IDs",
				intent.AcknowledgedWarnings)
		}
		if intent.Destination != destGemini || intent.Policy != string(handoff.PolicyFull) {
			t.Fatalf("intent carried %s/%s, want gemini/full", intent.Destination, intent.Policy)
		}
		if model.ExportRequested() {
			t.Error("enter must not request an export")
		}
		if model.Err() != nil {
			t.Errorf("Err() = %v, want none", model.Err())
		}
		if frame := model.View(); frame != "" {
			t.Errorf("a quitting studio still painted a frame:\n%s", frame)
		}
	})

	t.Run("e confirms an export", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		if len(model.Warnings()) > 0 {
			driver.Key("a")
		}
		driver.Key("e")
		if !model.Intent().Chosen() {
			t.Fatal("e did not confirm")
		}
		if !model.ExportRequested() {
			t.Fatal("e did not request an export")
		}
		if model.Intent().Action != tui.ActionHandoff {
			t.Fatalf("intent action = %q, want handoff", model.Intent().Action)
		}
	})

	for _, key := range []string{"esc", "ctrl+c", "q"} {
		t.Run(key+" cancels", func(t *testing.T) {
			driver, model, _ := start(t, config{})
			driver.Keys("tab", "right")
			driver.Key(key)
			assertZeroIntent(t, model.Intent())
			if model.ExportRequested() {
				t.Error("cancelling must not leave an export pending")
			}
		})
	}
}

// TestEnterRefusesAPlanThatCouldNotBeBuilt is the load-bearing safety rule of
// this surface. A handoff whose plan failed carries no briefing, so confirming
// it would drop the user into a brand new destination session that knows
// nothing about the work being handed over.
func TestEnterRefusesAPlanThatCouldNotBeBuilt(t *testing.T) {
	failure := errors.New("read transcript: permission denied")
	driver, model, _ := start(t, config{
		policy: string(handoff.PolicyBalanced),
		fail:   failing(destCodex, string(handoff.PolicyBalanced), failure),
	})

	preview, ready := model.current()
	if !ready || preview.Err == nil {
		t.Fatalf("the fixture did not fail the opening selection: %+v", preview)
	}

	driver.Key("enter")

	if model.Intent().Chosen() {
		t.Fatal("enter confirmed a handoff whose plan could not be built")
	}
	assertZeroIntent(t, model.Intent())
	if model.quitting {
		t.Fatal("a refused send must leave the studio open")
	}
	if !strings.Contains(model.status, "cannot be planned") {
		t.Fatalf("status = %q, want it to say the handoff cannot be planned", model.status)
	}
	if !strings.Contains(model.status, "permission denied") {
		t.Fatalf("status = %q, want it to name the failure", model.status)
	}
	frame := driver.View()
	if !strings.Contains(frame, "this handoff cannot be planned") {
		t.Errorf("frame does not report the failed plan\n%s", frame)
	}

	// A different policy plans cleanly, and the same key now confirms.
	driver.Key("right")
	if got := model.Policy(); got != string(handoff.PolicyFull) {
		t.Fatalf("policy = %q, want full", got)
	}
	if preview, ready := model.current(); !ready || preview.Err != nil {
		t.Fatalf("the full policy should plan cleanly: %+v", preview)
	}
	if model.status != "" {
		t.Errorf("status = %q, want the refusal cleared once the selection moved", model.status)
	}

	if len(model.Warnings()) > 0 {
		driver.Key("a")
	}
	driver.Key("enter")
	intent := model.Intent()
	if !intent.Chosen() {
		t.Fatal("enter did not confirm a policy that planned cleanly")
	}
	if intent.Policy != string(handoff.PolicyFull) || intent.Destination != destCodex {
		t.Fatalf("intent = %+v, want codex on the full policy", intent)
	}
}

// TestExportRefusesAPlanThatCouldNotBeBuilt covers the same refusal on the
// export key, and the flag it must not leave behind: a later successful send
// that silently became --no-launch would never reach the destination agent.
func TestExportRefusesAPlanThatCouldNotBeBuilt(t *testing.T) {
	failure := errors.New("destination codex is not installed")
	driver, model, _ := start(t, config{
		policy: string(handoff.PolicyBalanced),
		fail:   failing(destCodex, string(handoff.PolicyBalanced), failure),
	})

	driver.Key("e")

	if model.Intent().Chosen() {
		t.Fatal("e confirmed an export whose plan could not be built")
	}
	if model.ExportRequested() {
		t.Fatal("a refused export left the export flag set")
	}
	if !strings.Contains(model.status, "cannot be planned") {
		t.Fatalf("status = %q, want it to say the handoff cannot be planned", model.status)
	}
	if got := model.EquivalentCommand(); strings.Contains(got, "--no-launch") {
		t.Fatalf("command = %q, want no --no-launch after a refused export", got)
	}

	// The next successful send must be a real send, not a silent export.
	driver.Key("right")
	if len(model.Warnings()) > 0 {
		driver.Key("a")
	}
	driver.Key("enter")
	if !model.Intent().Chosen() {
		t.Fatal("enter did not confirm after the refused export")
	}
	if model.ExportRequested() {
		t.Fatal("the refused export flag survived into a later send")
	}
	if got := model.EquivalentCommand(); strings.Contains(got, "--no-launch") {
		t.Fatalf("command = %q, want no --no-launch", got)
	}
}

// TestPendingFrameMeasuresRatherThanShowingZeroes guards the one reading a user
// must never take from this screen. Zeroes on an unmeasured projection read as
// "this handoff carries nothing", which is a different fact entirely.
func TestPendingFrameMeasuresRatherThanShowingZeroes(t *testing.T) {
	for _, size := range []struct{ width, height int }{{110, 30}, {70, 24}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			model := pendingStudio(t, config{width: size.width, height: size.height})
			if _, ready := model.current(); ready {
				t.Fatal("the fixture measured the projection before the frame was drawn")
			}
			frame := frameOf(model)

			if !strings.Contains(frame, "measuring") {
				t.Errorf("a pending frame does not say it is measuring\n%s", frame)
			}
			for _, forbidden := range []string{"0 B", "0 events", "~0 tokens"} {
				if strings.Contains(frame, forbidden) {
					t.Errorf("a pending frame shows %q, which reads as a measured zero\n%s", forbidden, frame)
				}
			}
			if strings.Contains(frame, "carried across") {
				t.Errorf("a pending frame claims to know what is carried across\n%s", frame)
			}
			assertFrameWidth(t, frame, size.width)
		})
	}

	t.Run("a studio with no planner", func(t *testing.T) {
		driver, _, _ := start(t, config{noPlanner: true})
		frame := driver.View()
		if !strings.Contains(frame, "measuring") {
			t.Errorf("a studio without a planner must not claim a measurement\n%s", frame)
		}
		if strings.Contains(frame, "0 B") {
			t.Errorf("a studio without a planner shows a measured zero\n%s", frame)
		}
	})
}

// TestEveryFrameSaysThisIsNotANativeResume is a correctness requirement, not a
// cosmetic one. A handoff starts a NEW destination session from a briefing;
// believing it resumes the original is the most damaging misunderstanding a
// user of this command can carry away from it.
func TestEveryFrameSaysThisIsNotANativeResume(t *testing.T) {
	const line = "not a native resume"

	t.Run("pending", func(t *testing.T) {
		frame := frameOf(pendingStudio(t, config{}))
		if !strings.Contains(frame, line) {
			t.Fatalf("a pending frame omits %q\n%s", line, frame)
		}
	})

	t.Run("ready", func(t *testing.T) {
		driver, _, _ := start(t, config{})
		for _, keys := range [][]string{nil, {"right"}, {"left"}, {"tab"}, {"tab", "right"}} {
			driver.Keys(keys...)
			frame := driver.View()
			if !strings.Contains(frame, line) {
				t.Fatalf("a ready frame after %v omits %q\n%s", keys, line, frame)
			}
			if !strings.Contains(frame, "starts a NEW") {
				t.Fatalf("a ready frame after %v does not say a new session is started\n%s", keys, frame)
			}
		}
	})

	t.Run("plan error", func(t *testing.T) {
		driver, _, _ := start(t, config{
			fail: failing(destCodex, string(handoff.PolicyBalanced), errors.New("no such transcript")),
		})
		frame := driver.View()
		if !strings.Contains(frame, line) {
			t.Fatalf("an error frame omits %q\n%s", line, frame)
		}
		// A failed plan is not still being measured; promising a number that
		// will never arrive contradicts the refusal directly above it.
		if strings.Contains(frame, "measuring") {
			t.Errorf("an error frame still claims to be measuring\n%s", frame)
		}
		driver.Key("enter")
		if frame := driver.View(); !strings.Contains(frame, line) {
			t.Fatalf("a refused frame omits %q\n%s", line, frame)
		}
	})

	t.Run("it names the destination it would open", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		driver.Key("tab")
		want := "starts a NEW " + model.Destination() + " session"
		if frame := driver.View(); !strings.Contains(frame, want) {
			t.Fatalf("frame does not contain %q\n%s", want, frame)
		}
	})
}

// TestRedactionsShowCountsAndCategoriesNeverValues is the privacy contract of
// the preview. Reporting that a secret was hidden is the point; reprinting it
// on the way past would undo the redaction that hid it.
func TestRedactionsShowCountsAndCategoriesNeverValues(t *testing.T) {
	driver, model, _ := start(t, config{})
	preview, ready := model.current()
	if !ready {
		t.Fatal("the opening selection was not measured")
	}
	if preview.RedactionTotal() != 3 {
		t.Fatalf("the fixture hid %d values, want 3", preview.RedactionTotal())
	}

	frame := driver.View()
	if !strings.Contains(frame, "3 values hidden by redaction") {
		t.Errorf("frame does not report the redaction count\n%s", frame)
	}
	for _, category := range []string{"api_key", "aws_access_key_id"} {
		if !strings.Contains(frame, category) {
			t.Errorf("frame does not name the %q category\n%s", category, frame)
		}
	}
	if strings.Contains(frame, secretValue) {
		t.Fatalf("the planted secret reached the frame\n%s", frame)
	}
	if strings.Contains(frame, "sk-live") {
		t.Fatalf("a secret-shaped value reached the frame\n%s", frame)
	}
	if strings.Contains(frame, "DEPLOY_TOKEN") {
		t.Fatalf("event text reached the frame; the studio reports counts, not content\n%s", frame)
	}

	// The warning count is reported the same way: a number, not a payload.
	if !strings.Contains(frame, "1 warning to acknowledge") {
		t.Errorf("frame does not report the warning count\n%s", frame)
	}

	t.Run("a policy with more categories reports all of them", func(t *testing.T) {
		driver.Key("right")
		frame := driver.View()
		if !strings.Contains(frame, "7 values hidden by redaction") {
			t.Errorf("frame does not report the full policy's redaction count\n%s", frame)
		}
		if !strings.Contains(frame, "api_key, aws_access_key_id, bearer_token") {
			t.Errorf("frame does not list the categories in order\n%s", frame)
		}
		if strings.Contains(frame, secretValue) {
			t.Fatalf("the planted secret reached the frame\n%s", frame)
		}
		if !strings.Contains(frame, "2 warnings to acknowledge") {
			t.Errorf("frame does not report the full policy's warning count\n%s", frame)
		}
	})

	t.Run("no redactions means no redaction line", func(t *testing.T) {
		driver, _, _ := start(t, config{policy: string(handoff.PolicyCheckpoint)})
		frame := driver.View()
		if strings.Contains(frame, "hidden by redaction") {
			t.Errorf("a plan with no redactions reported some\n%s", frame)
		}
	})
}

// TestPreviewChangesWithTheSelection proves the frame is a measurement of the
// current pair rather than a static description of the command.
func TestPreviewChangesWithTheSelection(t *testing.T) {
	driver, model, _ := start(t, config{policy: string(handoff.PolicyCheckpoint)})

	frames := map[string]string{}
	for _, policy := range []string{"checkpoint", "balanced", "full"} {
		if got := model.Policy(); got != policy {
			t.Fatalf("policy = %q, want %q", got, policy)
		}
		frames[policy] = driver.View()
		driver.Key("right")
	}

	if !strings.Contains(frames["checkpoint"], "812 B") {
		t.Errorf("the checkpoint frame does not show its measured size\n%s", frames["checkpoint"])
	}
	if !strings.Contains(frames["balanced"], "11.5 KB") || !strings.Contains(frames["balanced"], "12 events") {
		t.Errorf("the balanced frame does not show its measurement\n%s", frames["balanced"])
	}
	if !strings.Contains(frames["full"], "47.0 KB") || !strings.Contains(frames["full"], "37 events") {
		t.Errorf("the full frame does not show its measurement\n%s", frames["full"])
	}
	if !strings.Contains(frames["full"], "~12.4k tokens") {
		t.Errorf("the full frame does not show its token estimate\n%s", frames["full"])
	}
	if frames["checkpoint"] == frames["balanced"] || frames["balanced"] == frames["full"] {
		t.Fatal("two policies rendered identical frames")
	}

	// The same policy against a different destination is a different plan.
	driver.Keys("right", "tab")
	if got := model.Destination(); got != destGemini {
		t.Fatalf("destination = %q, want gemini", got)
	}
	if got := driver.View(); got == frames["checkpoint"] {
		t.Fatal("changing the destination did not change the frame")
	}

	t.Run("a checkpoint policy says what it leaves behind", func(t *testing.T) {
		driver, _, _ := start(t, config{policy: string(handoff.PolicyCheckpoint)})
		frame := driver.View()
		if !strings.Contains(frame, "left behind") {
			t.Errorf("frame does not name what stays behind\n%s", frame)
		}
		if !strings.Contains(frame, "task boundary only") {
			t.Errorf("frame does not explain the policy\n%s", frame)
		}
	})
}

// TestClipboardCopiesTheEquivalentCommand covers the copy key in all three
// shapes a terminal can offer.
func TestClipboardCopiesTheEquivalentCommand(t *testing.T) {
	for _, key := range []string{"c", "y"} {
		t.Run(key+" copies", func(t *testing.T) {
			var copied []string
			driver, model, _ := start(t, config{
				clipboard: func(text string) error {
					copied = append(copied, text)
					return nil
				},
			})
			driver.Keys("tab", "right")
			want := model.EquivalentCommand()

			driver.Key(key)

			if len(copied) != 1 {
				t.Fatalf("clipboard called %d times, want once", len(copied))
			}
			if copied[0] != want {
				t.Fatalf("copied %q, want %q", copied[0], want)
			}
			if model.status != "copied "+want {
				t.Fatalf("status = %q, want %q", model.status, "copied "+want)
			}
			if !strings.Contains(driver.View(), model.status) {
				t.Errorf("status %q is not visible in the frame", model.status)
			}
			if model.quitting || model.Intent().Chosen() {
				t.Fatal("copying must not confirm or quit")
			}
		})
	}

	t.Run("a failing clipboard reports why", func(t *testing.T) {
		driver, model, _ := start(t, config{
			clipboard: func(string) error { return errors.New("no terminal") },
		})
		driver.Key("c")
		if !strings.Contains(model.status, "could not copy") {
			t.Fatalf("status = %q, want a copy failure", model.status)
		}
		if !strings.Contains(model.status, "no terminal") {
			t.Fatalf("status = %q, want it to name the failure", model.status)
		}
	})

	t.Run("a clipboard message from elsewhere still shows", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		driver.Send(tui.ClipboardMsg{Text: "rein handoff x --to codex --policy full"})
		if model.status != "copied rein handoff x --to codex --policy full" {
			t.Fatalf("status = %q, want the copied command", model.status)
		}
		if !strings.Contains(driver.View(), model.status) {
			t.Errorf("status %q is not visible in the frame", model.status)
		}
	})

	t.Run("without a clipboard the command is still shown", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		want := model.EquivalentCommand()
		driver.Key("y")
		if model.status != want {
			t.Fatalf("status = %q, want the command %q", model.status, want)
		}
		if model.quitting {
			t.Fatal("copying without a clipboard must not quit")
		}
	})
}

// TestUnhandledMessagesAreInert guards the update loop against reacting to
// messages it does not own.
func TestUnhandledMessagesAreInert(t *testing.T) {
	driver, _, fake := start(t, config{})
	before := driver.View()
	calls := fake.totalCalls()

	driver.Send(struct{ unknown int }{})
	driver.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("no")})
	driver.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	driver.Send(PreviewedMsg{Destination: destOpenCode, Policy: "full"})

	if got := driver.View(); got != before {
		t.Errorf("an unhandled message changed the frame\n%s", got)
	}
	if got := fake.totalCalls(); got != calls {
		t.Errorf("an unhandled message planned %d extra times", got-calls)
	}
}

// TestUntrustedPlanTextCannotRepaintTheTerminal is a security invariant.
// Component names, reasons and plan errors are built from vendor session files
// and vendor tooling output, so an escape sequence surviving into a frame would
// let a session file drive the user's terminal.
func TestUntrustedPlanTextCannotRepaintTheTerminal(t *testing.T) {
	hostile := []capsule.Component{
		{
			Name:        "\x1b[31muser_messages\x1b[0m\nrepaint attempt",
			Portability: capsule.PortabilityExact,
			Count:       4,
			Bytes:       2048,
		},
		{
			Name:        "attachments",
			Portability: capsule.PortabilityReferenced,
			Count:       1,
			Reason:      "\x1b[2Jleft\tin\nplace",
		},
	}
	hostileError := errors.New("read \x1b[31mtranscript\x1b[0m:\npermission denied")

	sizes := []struct{ width, height int }{{110, 30}, {70, 24}, {40, 12}}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			t.Run("hostile components", func(t *testing.T) {
				driver, _, _ := start(t, config{
					width: size.width, height: size.height, components: hostile,
				})
				for _, keys := range [][]string{nil, {"right"}, {"tab"}, {"left"}} {
					driver.Keys(keys...)
					frame := driver.View()
					if strings.ContainsRune(frame, 0x1b) {
						t.Fatalf("frame contains a raw escape byte after %v:\n%q", keys, frame)
					}
					if strings.Contains(frame, "\n\n\n") {
						t.Errorf("a component name broke the layout after %v:\n%s", keys, frame)
					}
					assertFrameWidth(t, frame, size.width)
				}
				// The text stays readable, only inert.
				if frame := driver.View(); !strings.Contains(frame, "[31muser_messages") {
					t.Errorf("the hostile name should still be readable with the escape removed:\n%s", frame)
				}
			})

			t.Run("a hostile plan error", func(t *testing.T) {
				driver, model, _ := start(t, config{
					width: size.width, height: size.height,
					fail: failing(destCodex, string(handoff.PolicyBalanced), hostileError),
				})
				frame := driver.View()
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("the error frame contains a raw escape byte:\n%q", frame)
				}
				assertFrameWidth(t, frame, size.width)

				// The refusal status repeats the message; it must be inert too.
				driver.Key("enter")
				if strings.ContainsRune(model.status, 0x1b) {
					t.Fatalf("the refusal status carries a raw escape byte: %q", model.status)
				}
				frame = driver.View()
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("the refused frame contains a raw escape byte:\n%q", frame)
				}
				assertFrameWidth(t, frame, size.width)
			})
		})
	}
}

// TestFrameWidthNeverExceedsTerminal is the layout regression net: every state
// of the surface, at every supported size, measured in display cells.
func TestFrameWidthNeverExceedsTerminal(t *testing.T) {
	longReason := []capsule.Component{
		{
			Name:        "files_touched_per_transcript",
			Portability: capsule.PortabilityReferenced,
			Count:       128,
			Reason:      "content_left_in_place_because_the_destination_cannot_address_it",
		},
		{
			Name:        "user_messages",
			Portability: capsule.PortabilityExact,
			Count:       12,
			Bytes:       6144,
		},
	}
	sizes := []struct{ width, height int }{
		{40, 12},
		{52, 14},
		{70, 24},
		{80, 24},
		{110, 30},
		{120, 40},
		{200, 50},
	}
	states := []struct {
		name string
		cfg  config
		keys []string
	}{
		{name: "ready"},
		{name: "checkpoint", cfg: config{policy: string(handoff.PolicyCheckpoint)}},
		{name: "full", keys: []string{"right"}},
		{name: "another destination", keys: []string{"tab", "tab"}},
		{name: "status", keys: []string{"c"}},
		{name: "no planner", cfg: config{noPlanner: true}},
		{name: "no destination", cfg: config{destinations: []string{}}},
		{name: "long component reasons", cfg: config{components: longReason}},
		{name: "no components", cfg: config{components: []capsule.Component{}}},
		{
			name: "plan error",
			cfg: config{fail: failing(destCodex, string(handoff.PolicyBalanced),
				errors.New("read transcript at /very/long/path/to/a/vendor/session/file.jsonl: permission denied"))},
			keys: []string{"enter"},
		},
	}
	for _, size := range sizes {
		for _, state := range states {
			t.Run(fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name), func(t *testing.T) {
				cfg := state.cfg
				cfg.width, cfg.height = size.width, size.height
				driver, _, _ := start(t, cfg)
				driver.Keys(state.keys...)
				assertFrameWidth(t, driver.View(), size.width)
				assertFrameWidth(t, frameOf(pendingStudio(t, cfg)), size.width)
			})
		}
	}
}

// TestGoldenFrames pins what a user actually sees. Regenerate with
// `go test ./internal/tui/handoffui/ -update-golden` and review the diff: an
// unexplained change to one of these is a regression in the surface.
func TestGoldenFrames(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config
		keys   []string
		golden string
	}{
		{
			name:   "ready 110x30",
			cfg:    config{width: 110, height: 30},
			golden: "handoff_ready_110x30",
		},
		{
			name: "plan error 110x30",
			cfg: config{width: 110, height: 30,
				fail: failing(destCodex, string(handoff.PolicyBalanced),
					errors.New("read transcript: permission denied"))},
			keys:   []string{"enter"},
			golden: "handoff_plan_error_110x30",
		},
		{
			name:   "narrow 70x24",
			cfg:    config{width: 70, height: 24},
			golden: "handoff_narrow_70x24",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, _, _ := start(t, test.cfg)
			driver.Keys(test.keys...)
			frame := driver.View()

			assertFrameWidth(t, frame, test.cfg.width)
			if strings.ContainsRune(frame, 0x1b) {
				t.Fatal("a golden frame must contain no escape sequences")
			}
			tuitest.AssertGolden(t, test.golden, frame)
		})
	}

	t.Run("pending 110x30", func(t *testing.T) {
		frame := frameOf(pendingStudio(t, config{width: 110, height: 30}))
		assertFrameWidth(t, frame, 110)
		if strings.ContainsRune(frame, 0x1b) {
			t.Fatal("a golden frame must contain no escape sequences")
		}
		tuitest.AssertGolden(t, "handoff_pending_110x30", frame)
	})
}

// TestSendRequiresAcknowledgingWarnings guards a defect that made the studio
// structurally incapable of completing a handoff.
//
// The pipeline requires one acknowledgement per current warning, and
// baseline.unavailable is always present on a handoff because the workspace is
// bound read-only with no baseline. The studio confirmed without collecting
// any, so the caller re-entered `rein handoff` with none and every studio
// launch died with a safety refusal.
func TestSendRequiresAcknowledgingWarnings(t *testing.T) {
	t.Run("enter refuses while warnings are unacknowledged", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		if len(model.Warnings()) == 0 {
			t.Skip("fixture selection carries no warnings")
		}
		driver.Key("enter")

		if model.Intent().Chosen() {
			t.Fatal("enter confirmed a handoff whose warnings were never acknowledged")
		}
		if model.quitting {
			t.Fatal("the studio quit instead of staying open to explain")
		}
		if !strings.Contains(model.status, "acknowledge") {
			t.Fatalf("status = %q, want it to say what is missing", model.status)
		}
		if frame := model.View(); !strings.Contains(frame, "acknowledge") {
			t.Fatalf("the frame does not tell the reader what to press:\n%s", frame)
		}
	})

	t.Run("a then enter forwards exactly the current warning IDs", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		warnings := model.Warnings()
		if len(warnings) == 0 {
			t.Skip("fixture selection carries no warnings")
		}
		driver.Keys("a", "enter")

		intent := model.Intent()
		if !intent.Chosen() {
			t.Fatal("enter did not confirm after acknowledgement")
		}
		if len(intent.AcknowledgedWarnings) != len(warnings) {
			t.Fatalf("acknowledged %v, want %v", intent.AcknowledgedWarnings, warnings)
		}
		for index, warning := range warnings {
			if intent.AcknowledgedWarnings[index] != warning {
				t.Fatalf("acknowledged %v, want %v", intent.AcknowledgedWarnings, warnings)
			}
		}
	})

	t.Run("the warning identifiers are on screen, not just a count", func(t *testing.T) {
		_, model, _ := start(t, config{})
		warnings := model.Warnings()
		if len(warnings) == 0 {
			t.Skip("fixture selection carries no warnings")
		}
		frame := model.View()
		for _, warning := range warnings {
			if !strings.Contains(frame, warning) {
				t.Fatalf("warning %q is not shown; accepting a warning you cannot name is not consent:\n%s",
					warning, frame)
			}
		}
	})

	t.Run("changing the selection clears the acknowledgement", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		if len(model.Warnings()) == 0 {
			t.Skip("fixture selection carries no warnings")
		}
		driver.Key("a")
		if !model.Acknowledged() {
			t.Fatal("a did not acknowledge")
		}
		// A different policy produces a different warning set, so carrying the
		// acceptance across would acknowledge warnings the user never saw.
		driver.Key("right")
		if model.Acknowledged() {
			t.Fatal("the acknowledgement survived a policy change")
		}
		driver.Key("a")
		driver.Key("tab")
		if model.Acknowledged() {
			t.Fatal("the acknowledgement survived a destination change")
		}
	})
}
