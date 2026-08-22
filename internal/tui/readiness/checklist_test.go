// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package readiness

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/exitcode"
	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/tuitest"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
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

// reference is the session every checklist in this file is acknowledging.
const reference = "claude:5f1c0b2a"

// The fixture check identifiers. None of them appear in the verifier's own
// ordering table, so a canonical report simply sorts them by identifier — which
// is also the order the checklist renders them in.
const (
	warnNodeVersion = "env.node.version"
	warnTreeDrift   = "env.working_tree.drift"
	infoBaseline    = "session.baseline"
	blockExecutable = "vendor.executable"
)

// checkSpec is the shorthand a test uses to describe one check without
// restating the parts of preflight.Check that policy fixes anyway.
type checkSpec struct {
	id       string
	severity preflight.Severity
	message  string
	repair   string
}

// buildReport assembles a canonical preflight report from specs.
//
// It derives everything policy derives — per-severity exit codes, check order,
// and the aggregate decision — so a fixture built here is a report the real
// preflight.Authorize will accept rather than a lookalike struct.
func buildReport(sessionRef string, specs ...checkSpec) preflight.Report {
	ordered := append([]checkSpec(nil), specs...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].id < ordered[j].id })

	checks := make([]preflight.Check, 0, len(ordered))
	decision := preflight.DecisionReady
	blockExit := 0
	for _, spec := range ordered {
		check := preflight.Check{
			ID:         spec.id,
			Severity:   spec.severity,
			Message:    spec.message,
			Repair:     spec.repair,
			Provenance: workspace.ProvenanceCurrentObservation,
		}
		switch spec.severity {
		case preflight.SeverityInfo:
			check.Status = preflight.StatusMatch
			check.ExitCode = 0
		case preflight.SeverityWarning:
			check.Status = preflight.StatusChanged
			check.ExitCode = exitcode.Safety
			if decision == preflight.DecisionReady {
				decision = preflight.DecisionConfirmationRequired
			}
		case preflight.SeverityBlock:
			check.Status = preflight.StatusMissing
			check.ExitCode = exitcode.Runtime
			decision = preflight.DecisionBlocked
			blockExit = exitcode.Runtime
		}
		checks = append(checks, check)
	}
	return preflight.Report{
		SchemaVersion: preflight.SchemaVersion,
		SessionRef:    sessionRef,
		Decision:      decision,
		BlockExitCode: blockExit,
		Checks:        checks,
	}
}

// fixtureReport is the corpus behind most tests here: two warnings that need a
// decision, one informational check, and one blocker. The mix is the point —
// only the warnings may become checkboxes.
func fixtureReport() preflight.Report {
	return buildReport(reference,
		checkSpec{
			id:       warnNodeVersion,
			severity: preflight.SeverityWarning,
			message:  "node is 22.6.0, the session recorded 20.11.1",
			repair:   "run nvm use 20.11.1 before resuming",
		},
		checkSpec{
			id:       warnTreeDrift,
			severity: preflight.SeverityWarning,
			message:  "the working tree has 3 uncommitted files the session did not see",
			repair:   "commit or stash before resuming",
		},
		checkSpec{
			id:       infoBaseline,
			severity: preflight.SeverityInfo,
			message:  "no prelaunch baseline exists for this session yet",
		},
		checkSpec{
			id:       blockExecutable,
			severity: preflight.SeverityBlock,
			message:  "the claude executable is not on PATH",
			repair:   "install the vendor CLI",
		},
	)
}

// warningsOnlyReport drops the blocker, because preflight.Authorize refuses a
// blocked report outright and the acknowledgement contract can only be tested on
// a report that could actually launch.
func warningsOnlyReport() preflight.Report {
	return buildReport(reference,
		checkSpec{
			id:       warnNodeVersion,
			severity: preflight.SeverityWarning,
			message:  "node is 22.6.0, the session recorded 20.11.1",
			repair:   "run nvm use 20.11.1 before resuming",
		},
		checkSpec{
			id:       warnTreeDrift,
			severity: preflight.SeverityWarning,
			message:  "the working tree has 3 uncommitted files the session did not see",
			repair:   "commit or stash before resuming",
		},
		checkSpec{
			id:       infoBaseline,
			severity: preflight.SeverityInfo,
			message:  "no prelaunch baseline exists for this session yet",
		},
	)
}

// config describes one checklist under test. The zero value is a 100x30
// monochrome resume checklist over the full fixture with no clipboard.
type config struct {
	width, height int
	report        *preflight.Report
	reference     string
	operation     string
	clipboard     tui.ClipboardFunc
}

func start(t *testing.T, cfg config) (*tuitest.Driver, *Checklist) {
	t.Helper()
	if cfg.width == 0 {
		cfg.width = 100
	}
	if cfg.height == 0 {
		cfg.height = 30
	}
	if cfg.reference == "" {
		cfg.reference = reference
	}
	report := fixtureReport()
	if cfg.report != nil {
		report = *cfg.report
	}
	capability := ui.Capability{
		Mode:    ui.ModeFull,
		Color:   ui.ColorNone,
		Unicode: true,
		Width:   cfg.width,
		Height:  cfg.height,
	}
	checklist := NewChecklist(Options{
		Theme:      ui.NewTheme(capability),
		Capability: capability,
		Report:     report,
		Reference:  cfg.reference,
		Operation:  cfg.operation,
		Clipboard:  cfg.clipboard,
	})
	return tuitest.New(t, checklist, cfg.width, cfg.height), checklist
}

func reportPtr(report preflight.Report) *preflight.Report { return &report }

// itemIDs returns the identifiers the checklist turned into rows, in order.
func itemIDs(c *Checklist) []string {
	ids := make([]string, 0, len(c.items))
	for _, current := range c.items {
		ids = append(ids, current.check.ID)
	}
	return ids
}

// assertZeroIntent checks a cancelled surface decided nothing at all. Intent
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

// assertSameSet compares identifier sets without depending on order.
func assertSameSet(t *testing.T, got, want []string, context string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if strings.Join(gotSorted, ",") != strings.Join(wantSorted, ",") {
		t.Fatalf("%s: got %v, want %v", context, gotSorted, wantSorted)
	}
}

// TestFixtureIsARealPreflightReport keeps the fixtures honest. If they drifted
// out of the canonical shape, every acknowledgement assertion below would be
// testing a struct the engine would never produce.
func TestFixtureIsARealPreflightReport(t *testing.T) {
	for name, report := range map[string]preflight.Report{
		"with a blocker": fixtureReport(),
		"warnings only":  warningsOnlyReport(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := preflight.ValidateReport(report); err != nil {
				t.Fatalf("fixture is not a valid preflight report: %v", err)
			}
		})
	}
	if got := warningsOnlyReport().Decision; got != preflight.DecisionConfirmationRequired {
		t.Fatalf("decision = %q, want %q", got, preflight.DecisionConfirmationRequired)
	}
	if got := fixtureReport().Decision; got != preflight.DecisionBlocked {
		t.Fatalf("decision = %q, want %q", got, preflight.DecisionBlocked)
	}
}

// TestOnlyWarningsBecomeItems is the honesty rule of the whole surface. A
// blocker cannot be acknowledged by anyone, so offering a checkbox for one would
// promise something preflight will refuse; informational checks need no
// decision at all.
func TestOnlyWarningsBecomeItems(t *testing.T) {
	driver, checklist := start(t, config{})

	want := []string{warnNodeVersion, warnTreeDrift}
	if got := itemIDs(checklist); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("items = %v, want %v", got, want)
	}
	if got := len(checklist.items); got != 2 {
		t.Fatalf("item count = %d, want 2", got)
	}

	frame := driver.View()
	glyphs := ui.NewTheme(ui.Capability{Unicode: true}).Glyphs
	for _, line := range strings.Split(frame, "\n") {
		hasBox := strings.Contains(line, glyphs.CheckOff) || strings.Contains(line, glyphs.CheckOn)
		if hasBox && strings.Contains(line, blockExecutable) {
			t.Fatalf("the blocking check is offered as a checkbox:\n%q", line)
		}
		if hasBox && strings.Contains(line, infoBaseline) {
			t.Fatalf("an informational check is offered as a checkbox:\n%q", line)
		}
	}
	if strings.Contains(frame, blockExecutable) {
		t.Errorf("the blocking check identifier appears in the acknowledgement frame:\n%s", frame)
	}
	if !strings.Contains(frame, warnNodeVersion) || !strings.Contains(frame, warnTreeDrift) {
		t.Errorf("a warning identifier is missing from the frame:\n%s", frame)
	}
	if !strings.Contains(frame, "2 environment warnings") {
		t.Errorf("the frame does not count the warnings:\n%s", frame)
	}
}

// TestNothingStartsAcknowledged guards against the one-keystroke mistake:
// a pre-ticked box lets a distracted reader accept every warning by pressing
// enter without reading a single line.
func TestNothingStartsAcknowledged(t *testing.T) {
	_, checklist := start(t, config{})

	if got := checklist.Acknowledged(); len(got) != 0 {
		t.Fatalf("Acknowledged = %v on a fresh checklist, want nothing", got)
	}
	if checklist.AllAccepted() {
		t.Fatal("AllAccepted is true with two unacknowledged warnings")
	}
	if checklist.Confirmed() {
		t.Fatal("Confirmed is true before the user did anything")
	}
	assertZeroIntent(t, checklist.Intent())
	if err := checklist.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTogglingAcknowledgement(t *testing.T) {
	t.Run("space toggles the item under the cursor", func(t *testing.T) {
		driver, checklist := start(t, config{})

		driver.Key(" ")
		assertSameSet(t, checklist.Acknowledged(), []string{warnNodeVersion}, "after one space")
		if checklist.AllAccepted() {
			t.Fatal("AllAccepted is true with one of two warnings ticked")
		}

		driver.Key(" ")
		if got := checklist.Acknowledged(); len(got) != 0 {
			t.Fatalf("space did not untick: Acknowledged = %v", got)
		}
	})

	t.Run("down moves to the next item", func(t *testing.T) {
		driver, checklist := start(t, config{})

		driver.Keys("down", " ")
		assertSameSet(t, checklist.Acknowledged(), []string{warnTreeDrift}, "after down then space")
		if checklist.cursor != 1 {
			t.Fatalf("cursor = %d, want 1", checklist.cursor)
		}

		driver.Keys("up", " ")
		assertSameSet(t, checklist.Acknowledged(), []string{warnNodeVersion, warnTreeDrift}, "after up then space")
		if !checklist.AllAccepted() {
			t.Fatal("AllAccepted is false with both warnings ticked")
		}
	})

	t.Run("the cursor stops at both ends", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Keys("up", "up", "up")
		if checklist.cursor != 0 {
			t.Fatalf("cursor = %d after moving up past the start, want 0", checklist.cursor)
		}
		driver.Keys("down", "down", "down", "down")
		if want := len(checklist.items) - 1; checklist.cursor != want {
			t.Fatalf("cursor = %d after moving down past the end, want %d", checklist.cursor, want)
		}
	})

	t.Run("a toggles every item on then off", func(t *testing.T) {
		driver, checklist := start(t, config{})

		driver.Key("a")
		assertSameSet(t, checklist.Acknowledged(), []string{warnNodeVersion, warnTreeDrift}, "after a")
		if !checklist.AllAccepted() {
			t.Fatal("AllAccepted is false after acknowledging all")
		}

		driver.Key("a")
		if got := checklist.Acknowledged(); len(got) != 0 {
			t.Fatalf("a second a left %v ticked, want nothing", got)
		}
	})

	t.Run("a completes a partial selection rather than inverting it", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Keys(" ", "a")
		if !checklist.AllAccepted() {
			t.Fatalf("a over a partial selection left %v ticked, want everything", checklist.Acknowledged())
		}
	})
}

// TestEnterRefusesUntilEveryWarningIsAcknowledged is the same rule preflight
// enforces, moved forward in time. Refusing here, with an explanation, beats
// launching and having preflight refuse afterwards with a message about
// identifiers the user never typed.
func TestEnterRefusesUntilEveryWarningIsAcknowledged(t *testing.T) {
	t.Run("nothing ticked", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Key("enter")

		if checklist.Confirmed() {
			t.Fatal("enter confirmed with no warnings acknowledged")
		}
		if checklist.quitting {
			t.Fatal("enter quit the surface with warnings outstanding")
		}
		assertZeroIntent(t, checklist.Intent())
		if checklist.status == "" {
			t.Fatal("enter refused silently; the user is owed a reason")
		}
		frame := driver.View()
		if frame == "" {
			t.Fatal("the surface stopped rendering after a refused enter")
		}
		if !strings.Contains(frame, checklist.status) {
			t.Fatalf("the refusal %q is not visible in the frame:\n%s", checklist.status, frame)
		}
	})

	t.Run("partially ticked", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Keys(" ", "enter")

		if checklist.Confirmed() {
			t.Fatal("enter confirmed with one of two warnings acknowledged")
		}
		if checklist.quitting {
			t.Fatal("enter quit the surface with a warning outstanding")
		}
	})

	t.Run("fully ticked", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Keys("a", "enter")

		if !checklist.Confirmed() {
			t.Fatal("enter did not confirm with every warning acknowledged")
		}
		if !checklist.quitting {
			t.Fatal("a confirmed checklist should quit")
		}
		if frame := driver.View(); frame != "" {
			t.Fatalf("a quitting surface must leave no frame behind, got:\n%s", frame)
		}

		intent := checklist.Intent()
		if intent.Action != tui.ActionResume {
			t.Fatalf("action = %q, want %q", intent.Action, tui.ActionResume)
		}
		if intent.Reference != reference {
			t.Fatalf("reference = %q, want %q", intent.Reference, reference)
		}
		if !intent.Chosen() {
			t.Fatal("a confirmed checklist should report a choice")
		}
		assertSameSet(t, intent.AcknowledgedWarnings, preflight.WarningIDs(fixtureReport()),
			"intent acknowledgements")
		if checklist.Err() != nil {
			t.Fatalf("unexpected error: %v", checklist.Err())
		}
	})

	t.Run("fork echoes the operation it was given", func(t *testing.T) {
		driver, checklist := start(t, config{operation: "fork"})
		driver.Keys("a", "enter")

		if got := checklist.Intent().Action; got != tui.ActionFork {
			t.Fatalf("action = %q, want %q", got, tui.ActionFork)
		}
	})

	t.Run("an unrecognised operation stays a resume", func(t *testing.T) {
		driver, checklist := start(t, config{operation: "handoff"})
		driver.Keys("a", "enter")

		if got := checklist.Intent().Action; got != tui.ActionResume {
			t.Fatalf("action = %q, want %q", got, tui.ActionResume)
		}
	})
}

// TestCancelKeysDecideNothing checks the exits. A cancelled acknowledgement must
// look exactly like never having opened the checklist, or the caller cannot tell
// refusal from a partial yes.
func TestCancelKeysDecideNothing(t *testing.T) {
	for _, key := range []string{"esc", "ctrl+c", "q"} {
		t.Run(key, func(t *testing.T) {
			driver, checklist := start(t, config{})
			// Tick everything first: cancelling must discard the ticks, not
			// carry them out of the surface.
			driver.Keys("a", key)

			if checklist.Confirmed() {
				t.Fatalf("%q confirmed the checklist", key)
			}
			if !checklist.quitting {
				t.Fatalf("%q did not quit the surface", key)
			}
			assertZeroIntent(t, checklist.Intent())
			if frame := driver.View(); frame != "" {
				t.Fatalf("a quitting surface must leave no frame behind, got:\n%s", frame)
			}
			if checklist.Err() != nil {
				t.Fatalf("unexpected error: %v", checklist.Err())
			}
		})
	}
}

// TestEquivalentCommand is the contract that keeps the interactive surface
// honest: whatever can be done here can be done, and scripted, from one command
// line. It is shown on screen at all times, so it must track the ticks exactly.
func TestEquivalentCommand(t *testing.T) {
	const flag = " --allow-environment-warning "

	tests := []struct {
		name      string
		operation string
		keys      []string
		want      string
	}{
		{
			name: "nothing ticked",
			want: "rein resume " + reference,
		},
		{
			name: "one ticked",
			keys: []string{" "},
			want: "rein resume " + reference + flag + warnNodeVersion,
		},
		{
			name: "the second ticked",
			keys: []string{"down", " "},
			want: "rein resume " + reference + flag + warnTreeDrift,
		},
		{
			name: "both ticked in checklist order",
			keys: []string{"a"},
			want: "rein resume " + reference + flag + warnNodeVersion + flag + warnTreeDrift,
		},
		{
			name: "ticked out of order still renders in checklist order",
			keys: []string{"down", " ", "up", " "},
			want: "rein resume " + reference + flag + warnNodeVersion + flag + warnTreeDrift,
		},
		{
			name: "unticking removes the flag again",
			keys: []string{"a", " "},
			want: "rein resume " + reference + flag + warnTreeDrift,
		},
		{
			name:      "fork",
			operation: "fork",
			keys:      []string{"a"},
			want:      "rein fork " + reference + flag + warnNodeVersion + flag + warnTreeDrift,
		},
		{
			name:      "an empty operation defaults to resume",
			operation: "   ",
			want:      "rein resume " + reference,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, checklist := start(t, config{operation: test.operation})
			driver.Keys(test.keys...)

			if got := checklist.EquivalentCommand(); got != test.want {
				t.Fatalf("EquivalentCommand =\n  %q\nwant\n  %q", got, test.want)
			}
		})
	}

	// The command is not merely computable; it is on screen. A wide terminal
	// shows it unwrapped, which is what makes it copyable by eye.
	t.Run("the command is visible in the frame", func(t *testing.T) {
		driver, checklist := start(t, config{width: 160, height: 40})
		driver.Key("a")
		frame := driver.View()
		if !strings.Contains(frame, checklist.EquivalentCommand()) {
			t.Fatalf("the equivalent command is not in the frame:\n%s", frame)
		}
		if !strings.Contains(frame, "equivalent command") {
			t.Errorf("the equivalent command is not labelled:\n%s", frame)
		}
	})

	// The flag identifiers must be the exact strings the checklist collects,
	// not a prettified rendering of them.
	t.Run("the flags carry the acknowledged identifiers verbatim", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Key("a")
		command := checklist.EquivalentCommand()
		for _, id := range checklist.Acknowledged() {
			if !strings.Contains(command, flag+id) {
				t.Fatalf("command %q does not carry %q", command, id)
			}
		}
		if got := strings.Count(command, flag); got != len(checklist.Acknowledged()) {
			t.Fatalf("command has %d flags for %d acknowledgements: %q",
				got, len(checklist.Acknowledged()), command)
		}
	})
}

// TestAcknowledgementIsHonouredByTheRealPolicy is the load-bearing test of this
// package. The checklist claims to grant exactly what the flags grant, so the
// identifiers it collects are fed to the real preflight.Authorize — the same
// call the --allow-environment-warning path makes. If this passes and the
// partial case is refused, the interactive surface has neither widened nor
// narrowed what a user can authorize.
func TestAcknowledgementIsHonouredByTheRealPolicy(t *testing.T) {
	report := warningsOnlyReport()
	if got := len(preflight.WarningIDs(report)); got != 2 {
		t.Fatalf("fixture has %d warnings, want 2", got)
	}

	t.Run("a complete acknowledgement authorizes the launch", func(t *testing.T) {
		driver, checklist := start(t, config{report: reportPtr(report)})
		driver.Key("a")

		acknowledged := checklist.Acknowledged()
		assertSameSet(t, acknowledged, preflight.WarningIDs(report), "Acknowledged versus preflight.WarningIDs")

		authorization, err := preflight.Authorize(report, acknowledged)
		if err != nil {
			t.Fatalf("preflight.Authorize refused a complete acknowledgement: %v", err)
		}
		if !authorization.Allowed {
			t.Fatalf("authorization = %+v, want Allowed", authorization)
		}
		if len(authorization.Warnings) != 0 {
			t.Fatalf("authorization still lists outstanding warnings: %v", authorization.Warnings)
		}
	})

	t.Run("a partial acknowledgement does not", func(t *testing.T) {
		driver, checklist := start(t, config{report: reportPtr(report)})
		driver.Key(" ")

		acknowledged := checklist.Acknowledged()
		if len(acknowledged) != 1 {
			t.Fatalf("Acknowledged = %v, want exactly one identifier", acknowledged)
		}

		authorization, err := preflight.Authorize(report, acknowledged)
		if err == nil {
			t.Fatal("preflight.Authorize allowed a partial acknowledgement")
		}
		if authorization.Allowed {
			t.Fatalf("authorization = %+v, want refusal", authorization)
		}
		if authorization.ExitCode != exitcode.Safety {
			t.Fatalf("exit code = %d, want %d", authorization.ExitCode, exitcode.Safety)
		}
		// And the checklist refuses to hand the caller a confirmation at all,
		// so this list never reaches Authorize in production.
		if checklist.AllAccepted() {
			t.Fatal("AllAccepted is true with one of two warnings ticked")
		}
	})

	t.Run("no acknowledgement does not", func(t *testing.T) {
		_, checklist := start(t, config{report: reportPtr(report)})
		authorization, err := preflight.Authorize(report, checklist.Acknowledged())
		if err == nil || authorization.Allowed {
			t.Fatalf("preflight.Authorize allowed an empty acknowledgement: %+v", authorization)
		}
		assertSameSet(t, authorization.Warnings, preflight.WarningIDs(report), "outstanding warnings")
	})

	// A blocker cannot be acknowledged at all, and the checklist never offers
	// one — so the full fixture's acknowledgement is still refused by policy.
	t.Run("a blocked report is refused whatever is ticked", func(t *testing.T) {
		blocked := fixtureReport()
		driver, checklist := start(t, config{report: reportPtr(blocked)})
		driver.Key("a")

		authorization, err := preflight.Authorize(blocked, checklist.Acknowledged())
		if err == nil || authorization.Allowed {
			t.Fatalf("preflight.Authorize allowed a blocked report: %+v", authorization)
		}
	})
}

func TestClipboardCopiesTheEquivalentCommand(t *testing.T) {
	for _, key := range []string{"c", "y"} {
		t.Run("with a clipboard: "+key, func(t *testing.T) {
			var copied []string
			driver, checklist := start(t, config{
				clipboard: func(text string) error {
					copied = append(copied, text)
					return nil
				},
			})
			driver.Key("a")
			want := checklist.EquivalentCommand()

			driver.Key(key)

			if len(copied) != 1 {
				t.Fatalf("clipboard called %d times, want once", len(copied))
			}
			if copied[0] != want {
				t.Fatalf("copied %q, want %q", copied[0], want)
			}
			if checklist.status != "copied "+want {
				t.Fatalf("status = %q, want %q", checklist.status, "copied "+want)
			}
			if checklist.quitting {
				t.Fatal("copying should not quit the surface")
			}
			if checklist.Confirmed() {
				t.Fatal("copying should not confirm the checklist")
			}
		})
	}

	t.Run("the status comes from the clipboard message", func(t *testing.T) {
		driver, checklist := start(t, config{clipboard: func(string) error { return nil }})
		driver.Send(tui.ClipboardMsg{Text: "rein resume " + reference})
		if want := "copied rein resume " + reference; checklist.status != want {
			t.Fatalf("status = %q, want %q", checklist.status, want)
		}
		if !strings.Contains(driver.View(), "copied") {
			t.Errorf("the copy status is not visible in the frame:\n%s", driver.View())
		}
	})

	t.Run("a failing clipboard reports why", func(t *testing.T) {
		driver, checklist := start(t, config{
			clipboard: func(string) error { return errors.New("no terminal") },
		})
		driver.Key("c")
		if !strings.Contains(checklist.status, "could not copy") {
			t.Fatalf("status = %q, want a copy failure", checklist.status)
		}
		if !strings.Contains(driver.View(), "could not copy") {
			t.Errorf("the copy failure is not visible in the frame:\n%s", driver.View())
		}
	})

	// Without a clipboard the command still has to reach the user somehow. It
	// goes to the status line rather than nowhere, and certainly rather than a
	// nil call.
	t.Run("without a clipboard the command is shown instead", func(t *testing.T) {
		driver, checklist := start(t, config{})
		driver.Keys("a", "c")

		if checklist.status != checklist.EquivalentCommand() {
			t.Fatalf("status = %q, want the equivalent command %q",
				checklist.status, checklist.EquivalentCommand())
		}
		if checklist.quitting {
			t.Fatal("copying without a clipboard quit the surface")
		}
		if frame := driver.View(); frame == "" {
			t.Fatal("the surface stopped rendering after a clipboard-less copy")
		}
	})
}

// TestUntrustedCheckTextCannotRepaintTheTerminal is a security invariant, not a
// cosmetic one. Check messages describe vendor-adjacent inspection — a runtime
// version string, a branch name, a path — so an escape sequence surviving into a
// frame would let the inspected environment drive the user's terminal.
func TestUntrustedCheckTextCannotRepaintTheTerminal(t *testing.T) {
	hostile := buildReport(reference,
		checkSpec{
			id:       warnNodeVersion,
			severity: preflight.SeverityWarning,
			message:  "node reported \x1b[31mv22.6.0\x1b[0m\nand the session recorded 20.11.1",
			repair:   "run \x1b]0;pwned\a nvm use 20.11.1\nthen resume",
		},
		checkSpec{
			id:       warnTreeDrift,
			severity: preflight.SeverityWarning,
			message:  "branch \x1b[2J\x1b[H is not the recorded branch",
			repair:   "checkout\tthe recorded branch",
		},
	)

	// Each state starts from a fresh checklist. Replaying keys onto one driver
	// would let an earlier state tick every box, at which point enter confirms,
	// the surface quits, and every later assertion passes against an empty
	// frame without inspecting anything.
	states := []struct {
		name string
		keys []string
	}{
		{name: "fresh"},
		{name: "ticked", keys: []string{"a"}},
		{name: "moved", keys: []string{"down"}},
		{name: "refused", keys: []string{"enter"}},
		{name: "copied", keys: []string{"c"}},
	}
	sizes := []struct{ width, height int }{{160, 40}, {100, 30}, {80, 24}, {60, 20}, {40, 12}}
	for _, size := range sizes {
		for _, state := range states {
			t.Run(fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name), func(t *testing.T) {
				driver, checklist := start(t, config{
					width:  size.width,
					height: size.height,
					report: reportPtr(hostile),
				})
				driver.Keys(state.keys...)

				frame := driver.View()
				if frame == "" {
					t.Fatal("the surface stopped rendering; this assertion would be vacuous")
				}
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("frame contains a raw escape byte:\n%q", frame)
				}
				assertFrameWidth(t, frame, size.width)

				// The text is still readable, just inert: the escape introducer
				// is gone and its payload is left as plain characters.
				if size.width >= 100 && !strings.Contains(frame, "[31mv22.6.0") {
					t.Errorf("the sanitized message is unreadable:\n%s", frame)
				}
				if checklist.Confirmed() {
					t.Fatal("a hostile report was confirmed without every box ticked")
				}
			})
		}
	}

	// Summary is the degraded, non-interactive rendering of the same report and
	// must not be the hole the escape gets through.
	t.Run("summary", func(t *testing.T) {
		theme := ui.NewTheme(ui.Capability{Unicode: true})
		lines := Summary(theme, hostile)
		if len(lines) != 2 {
			t.Fatalf("Summary produced %d lines, want one per warning (2)", len(lines))
		}
		for _, line := range lines {
			if strings.ContainsRune(line, 0x1b) {
				t.Fatalf("Summary line contains a raw escape byte: %q", line)
			}
		}
		if !strings.Contains(lines[0], warnNodeVersion) {
			t.Errorf("Summary line does not identify the check: %q", lines[0])
		}
		if Summary(theme, buildReport(reference, checkSpec{
			id: infoBaseline, severity: preflight.SeverityInfo, message: "nothing to do",
		})) != nil {
			t.Error("Summary described a report with no warnings")
		}
	})
}

// TestFrameWidthNeverExceedsTerminal is the layout regression net: every state
// of the surface, at every supported size, measured in display cells.
func TestFrameWidthNeverExceedsTerminal(t *testing.T) {
	sizes := []struct{ width, height int }{
		{40, 10}, // the narrowest interactive terminal ui.Detect will hand us
		{52, 14},
		{60, 20},
		{70, 20},
		{80, 24},
		{100, 30},
		{120, 40},
		{200, 50},
	}
	empty := buildReport(reference, checkSpec{
		id: infoBaseline, severity: preflight.SeverityInfo, message: "no prelaunch baseline exists yet",
	})
	states := []struct {
		name string
		cfg  config
		keys []string
	}{
		{name: "fresh"},
		{name: "all ticked", keys: []string{"a"}},
		{name: "partly ticked", keys: []string{"down", " "}},
		{name: "refused", keys: []string{"enter"}},
		{name: "status", keys: []string{"a", "c"}},
		{name: "fork", cfg: config{operation: "fork"}, keys: []string{"a"}},
		{name: "no warnings", cfg: config{report: reportPtr(empty)}},
		{
			name: "a long reference",
			cfg:  config{reference: strings.Repeat("claude:0123456789abcdef", 12)},
			keys: []string{"a"},
		},
	}
	for _, size := range sizes {
		for _, state := range states {
			t.Run(fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name), func(t *testing.T) {
				cfg := state.cfg
				cfg.width, cfg.height = size.width, size.height
				driver, _ := start(t, cfg)
				driver.Keys(state.keys...)
				assertFrameWidth(t, driver.View(), size.width)
			})
		}
	}
}

// TestAReportWithNoWarningsConfirmsImmediately covers the vacuous case. It is
// reachable: a report can require confirmation for a warning that a repair
// cleared between the probe and the prompt, and a checklist with no boxes must
// not become a dead end.
func TestAReportWithNoWarningsConfirmsImmediately(t *testing.T) {
	empty := buildReport(reference, checkSpec{
		id: infoBaseline, severity: preflight.SeverityInfo, message: "no prelaunch baseline exists yet",
	})
	driver, checklist := start(t, config{report: reportPtr(empty)})

	if got := len(checklist.items); got != 0 {
		t.Fatalf("items = %d, want none", got)
	}
	if !checklist.AllAccepted() {
		t.Fatal("AllAccepted is false with no warnings to accept")
	}
	frame := driver.View()
	if !strings.Contains(frame, "No warnings to acknowledge.") {
		t.Fatalf("the empty state is not explained:\n%s", frame)
	}
	if !strings.Contains(frame, "0 environment warnings") {
		t.Errorf("the empty state does not count the warnings:\n%s", frame)
	}

	// Keys that move or toggle nothing must not misbehave on an empty list.
	driver.Keys("down", "up", " ", "a")
	if got := checklist.Acknowledged(); len(got) != 0 {
		t.Fatalf("Acknowledged = %v on an empty checklist, want nothing", got)
	}
	if got := checklist.EquivalentCommand(); got != "rein resume "+reference {
		t.Fatalf("EquivalentCommand = %q, want a bare command", got)
	}

	driver.Key("enter")
	if !checklist.Confirmed() {
		t.Fatal("enter did not confirm a checklist with nothing to acknowledge")
	}
	intent := checklist.Intent()
	if intent.Action != tui.ActionResume || intent.Reference != reference {
		t.Fatalf("intent = %+v, want a resume of %q", intent, reference)
	}
	if len(intent.AcknowledgedWarnings) != 0 {
		t.Fatalf("intent acknowledged %v, want nothing", intent.AcknowledgedWarnings)
	}
}

// TestUnhandledMessagesAreInert guards the update loop against reacting to
// messages it does not own.
func TestUnhandledMessagesAreInert(t *testing.T) {
	driver, checklist := start(t, config{})
	before := driver.View()

	driver.Send(tea.KeyMsg{Type: tea.KeyLeft})
	driver.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("zz")})
	driver.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	driver.Send(struct{ unknown int }{})

	if got := driver.View(); got != before {
		t.Errorf("an unhandled message changed the frame")
	}
	if checklist.quitting || checklist.Confirmed() {
		t.Fatal("an unhandled message decided something")
	}
	assertZeroIntent(t, checklist.Intent())
}

// TestGoldenFrames pins what a user actually sees. Regenerate with
// `go test ./internal/tui/readiness/ -update-golden` and review the diff: an
// unexplained change to one of these is a regression in the surface.
func TestGoldenFrames(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config
		keys   []string
		golden string
	}{
		{
			name:   "nothing acknowledged 100x30",
			cfg:    config{width: 100, height: 30},
			golden: "checklist_fresh_100x30",
		},
		{
			name:   "all acknowledged 100x30",
			cfg:    config{width: 100, height: 30},
			keys:   []string{"a"},
			golden: "checklist_acknowledged_100x30",
		},
		{
			name:   "narrow 60x20",
			cfg:    config{width: 60, height: 20},
			keys:   []string{"down", " "},
			golden: "checklist_narrow_60x20",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, _ := start(t, test.cfg)
			driver.Keys(test.keys...)
			frame := driver.View()

			assertFrameWidth(t, frame, test.cfg.width)
			if strings.ContainsRune(frame, 0x1b) {
				t.Fatal("a golden frame must contain no escape sequences")
			}
			tuitest.AssertGolden(t, test.golden, frame)
		})
	}
}
