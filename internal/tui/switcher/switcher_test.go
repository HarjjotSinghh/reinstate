// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package switcher

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
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

// fixtureNow is the reference instant every fixture timestamp is measured from.
// It is a local wall-clock time on purpose: ui.SectionFor buckets by local
// calendar day, so building the fixture in time.Local keeps the section headings
// identical in every timezone the suite runs in. Mid-June avoids every daylight
// saving transition in both hemispheres.
func fixtureNow() time.Time {
	return time.Date(2026, time.June, 15, 14, 30, 0, 0, time.Local)
}

const (
	projectReinstate = "reinstate"
	projectWebsite   = "reinstate-website" // 17 cells: exercises project truncation
	projectToolchain = "toolchain"
	projectNotes     = "notes-app"
	projectArchive   = "archive-tool"
)

// fixtureRecords is the fixed corpus behind every test in this file: five
// agents, five projects, and timestamps chosen relative to fixtureNow so the
// rows land in the today / yesterday / this week / this month / older buckets.
// It is written newest first, the order a Loader is contracted to return.
//
// Three titles are deliberately hostile input: one CJK, one emoji, and one
// carrying a raw ANSI sequence and a newline. Session titles come from vendor
// files that Reinstate does not control, so they belong in the fixture rather
// than in a single special-case test.
func fixtureRecords() []sessionindex.Record {
	now := fixtureNow()
	return []sessionindex.Record{
		{
			Key:           "claude:5f1c0b2a",
			ID:            "5f1c0b2a",
			Agent:         sessionindex.AgentClaude,
			Title:         "Fix the auth token refresh loop",
			Project:       projectReinstate,
			Branch:        "feat/auth-refresh",
			UpdatedAt:     now.Add(-25 * time.Minute),
			MessageCount:  142,
			PromptPreview: "the refresh loop retries forever when the token endpoint answers 401, so find the exit condition",
			Files: []string{
				"internal/auth/token.go",
				"internal/auth/refresh.go",
				"internal/auth/token_test.go",
			},
			CanResume: true,
			CanFork:   true,
		},
		{
			Key:           "codex:9b7d4118",
			ID:            "9b7d4118",
			Agent:         sessionindex.AgentCodex,
			Title:         "認証リファクタを修正する",
			Project:       projectReinstate,
			Branch:        "main",
			UpdatedAt:     now.Add(-3 * time.Hour),
			MessageCount:  31,
			PromptPreview: "認証まわりのリファクタで壊れたテストを直す",
			Files:         []string{"internal/auth/session.go"},
			CanResume:     true,
			CanFork:       true,
		},
		{
			Key:           "gemini:a13f7702",
			ID:            "a13f7702",
			Agent:         sessionindex.AgentGemini,
			Title:         "🚀 Ship the launch checklist",
			Project:       projectWebsite,
			Branch:        "release/0.5.2",
			UpdatedAt:     now.Add(-6 * time.Hour),
			MessageCount:  8,
			PromptPreview: "walk the launch checklist and mark what is still open",
			CanResume:     true,
		},
		{
			Key:            "grok:4c2e9055",
			ID:             "4c2e9055",
			Agent:          sessionindex.AgentGrok,
			Title:          "\x1b[31mmalicious\x1b[0m\nrepaint attempt",
			Project:        projectWebsite,
			UpdatedAt:      now.Add(-26 * time.Hour),
			MessageCount:   3,
			PromptPreview:  "untrusted \x1b[2J preview text",
			ReadOnlyReason: "fixture session is read-only",
		},
		{
			Key:           "opencode:71aa30e4",
			ID:            "71aa30e4",
			Agent:         sessionindex.AgentOpenCode,
			Title:         "Rewrite the build pipeline",
			Project:       projectToolchain,
			Branch:        "chore/build",
			UpdatedAt:     now.Add(-30 * time.Hour),
			MessageCount:  57,
			PromptPreview: "split the release job so the notarisation step can retry on its own",
			Files:         []string{"Makefile", ".github/workflows/release.yml"},
			CanResume:     true,
			CanFork:       true,
		},
		{
			Key:          "claude:c8043b19",
			ID:           "c8043b19",
			Agent:        sessionindex.AgentClaude,
			Title:        "Bisect the flaky Windows path test",
			Project:      projectToolchain,
			Branch:       "fix/pathmap-windows",
			UpdatedAt:    now.Add(-3 * 24 * time.Hour),
			MessageCount: 96,
			Files:        []string{"internal/pathmap/rewrite.go"},
			CanResume:    true,
			CanFork:      true,
		},
		{
			Key:          "codex:2d55f0ab",
			ID:           "2d55f0ab",
			Agent:        sessionindex.AgentCodex,
			Title:        "Draft the release notes",
			Project:      projectNotes,
			UpdatedAt:    now.Add(-5 * 24 * time.Hour),
			MessageCount: 12,
			CanResume:    true,
		},
		{
			Key:          "gemini:6e90cc31",
			ID:           "6e90cc31",
			Agent:        sessionindex.AgentGemini,
			Title:        "Investigate the pathmap regression",
			Project:      projectReinstate,
			Branch:       "main",
			UpdatedAt:    now.Add(-12 * 24 * time.Hour),
			MessageCount: 44,
			CanResume:    true,
		},
		{
			Key:          "claude:0917bd6f",
			ID:           "0917bd6f",
			Agent:        sessionindex.AgentClaude,
			Title:        "Port the old importer",
			Project:      projectArchive,
			UpdatedAt:    now.Add(-90 * 24 * time.Hour),
			MessageCount: 5,
		},
	}
}

// fixtureReadiness is a synthetic readiness rule. It reads only the record's own
// resumability flags, so it never asserts anything about a real vendor.
func fixtureReadiness(record sessionindex.Record) ui.Readiness {
	switch {
	case record.ReadOnlyReason != "":
		return ui.ReadinessBlocked
	case !record.CanResume:
		return ui.ReadinessWarn
	default:
		return ui.ReadinessReady
	}
}

// fakeLoader answers Load from a canned corpus and records every filter it was
// handed, which is how the tests prove filtering happens in the index rather
// than in the view.
type fakeLoader struct {
	records []sessionindex.Record
	err     error
	// respond overrides the canned answer so a reload can return a different
	// page without the test reaching into the model.
	respond func(sessionindex.Filter) ([]sessionindex.Record, error)
	filters []sessionindex.Filter
}

func (l *fakeLoader) Load(filter sessionindex.Filter) ([]sessionindex.Record, error) {
	l.filters = append(l.filters, filter)
	if l.respond != nil {
		return l.respond(filter)
	}
	return l.records, l.err
}

func (l *fakeLoader) calls() int { return len(l.filters) }

func (l *fakeLoader) last(t *testing.T) sessionindex.Filter {
	t.Helper()
	if len(l.filters) == 0 {
		t.Fatal("loader was never called")
	}
	return l.filters[len(l.filters)-1]
}

// config describes one switcher under test. The zero value is a full-mode
// 120x40 switcher over the whole fixture with no project scope.
type config struct {
	width     int
	height    int
	mode      ui.Mode
	project   string
	limit     int
	records   []sessionindex.Record
	loadErr   error
	respond   func(sessionindex.Filter) ([]sessionindex.Record, error)
	noRecords bool
	// plainReadiness omits the readiness column, exercising the preview's
	// read-only branch instead.
	plainReadiness bool
	clipboard      tui.ClipboardFunc
	// daemon is the daemon summary for the status line.
	daemon string
}

// start builds a switcher and drives it through the deterministic harness.
func start(t *testing.T, cfg config) (*tuitest.Driver, *Model, *fakeLoader) {
	t.Helper()
	if cfg.width == 0 {
		cfg.width = 120
	}
	if cfg.height == 0 {
		cfg.height = 40
	}
	if cfg.mode == ui.ModePlain {
		cfg.mode = ui.ModeFull
	}
	if cfg.records == nil && !cfg.noRecords {
		cfg.records = fixtureRecords()
	}
	capability := ui.Capability{
		Mode:    cfg.mode,
		Color:   ui.ColorNone,
		Unicode: true,
		Width:   cfg.width,
		Height:  cfg.height,
	}
	loader := &fakeLoader{records: cfg.records, err: cfg.loadErr, respond: cfg.respond}
	readiness := StaticReadiness(fixtureReadiness)
	if cfg.plainReadiness {
		readiness = nil
	}
	model := New(Options{
		Theme:      ui.NewTheme(capability),
		Capability: capability,
		Loader:     loader,
		Readiness:  readiness,
		Project:    cfg.project,
		Now:        fixtureNow(),
		Limit:      cfg.limit,
		Clipboard:  cfg.clipboard,
		Daemon:     cfg.daemon,
	})
	driver := tuitest.New(t, model, cfg.width, cfg.height)
	return driver, model, loader
}

// sessionRows returns the indexes of every selectable row.
func sessionRows(m *Model) []int {
	indexes := make([]int, 0, len(m.rows))
	for index, current := range m.rows {
		if !current.isHeader() {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func mustSelected(t *testing.T, m *Model) sessionindex.Record {
	t.Helper()
	record := m.selected()
	if record == nil {
		t.Fatalf("no session selected at cursor %d of %d rows", m.cursor, len(m.rows))
	}
	return *record
}

// assertOnSession is the load-bearing invariant of the whole surface: headers
// are interleaved with sessions, so a cursor that can rest on one would make
// enter do nothing at all.
func assertOnSession(t *testing.T, m *Model, context string) {
	t.Helper()
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		t.Fatalf("%s: cursor %d out of range for %d rows", context, m.cursor, len(m.rows))
	}
	if m.rows[m.cursor].isHeader() {
		t.Fatalf("%s: cursor %d landed on section header %q", context, m.cursor, m.rows[m.cursor].header)
	}
	if m.selected() == nil {
		t.Fatalf("%s: cursor %d selects no record", context, m.cursor)
	}
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

func TestFixtureCoversEverySection(t *testing.T) {
	_, model, _ := start(t, config{})

	want := []string{
		ui.SectionToday.Title(),
		ui.SectionYesterday.Title(),
		ui.SectionThisWeek.Title(),
		ui.SectionThisMonth.Title(),
		ui.SectionOlder.Title(),
	}
	var got []string
	for _, current := range model.rows {
		if current.isHeader() {
			got = append(got, current.header)
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("section headings = %v, want %v", got, want)
	}
	if len(model.rows) != len(fixtureRecords())+len(want) {
		t.Fatalf("rows = %d, want %d sessions plus %d headings",
			len(model.rows), len(fixtureRecords()), len(want))
	}
}

// TestCursorNeverLandsOnSectionHeader walks the list end to end in both
// directions and after every single move asserts the selection is a session.
func TestCursorNeverLandsOnSectionHeader(t *testing.T) {
	driver, model, _ := start(t, config{})
	total := len(model.rows)
	if total < 10 {
		t.Fatalf("fixture is too small to exercise scrolling: %d rows", total)
	}
	assertOnSession(t, model, "initial")

	for step := 0; step < total+3; step++ {
		driver.Key("down")
		assertOnSession(t, model, fmt.Sprintf("down step %d", step))
	}
	if model.cursor != sessionRows(model)[len(sessionRows(model))-1] {
		t.Fatalf("down past the end stopped at row %d, want the last session row", model.cursor)
	}
	for step := 0; step < total+3; step++ {
		driver.Key("up")
		assertOnSession(t, model, fmt.Sprintf("up step %d", step))
	}
	if model.cursor != sessionRows(model)[0] {
		t.Fatalf("up past the start stopped at row %d, want the first session row", model.cursor)
	}

	// Every other way of moving must honour the same invariant.
	jumps := []struct {
		name string
		keys []string
	}{
		{"end", []string{"end"}},
		{"home", []string{"home"}},
		{"pgdown", []string{"pgdown"}},
		{"pgdown twice", []string{"pgdown", "pgdown"}},
		{"pgup", []string{"pgup"}},
		{"end then pgup", []string{"end", "pgup"}},
		{"ctrl+n", []string{"ctrl+n"}},
		{"ctrl+p", []string{"ctrl+p"}},
		{"home then end", []string{"home", "end"}},
	}
	for _, jump := range jumps {
		t.Run(jump.name, func(t *testing.T) {
			driver, model, _ := start(t, config{height: 14})
			for _, key := range jump.keys {
				driver.Key(key)
				assertOnSession(t, model, jump.name+" "+key)
			}
		})
	}
}

func TestEnterReturnsResumeIntentForSelectedRecord(t *testing.T) {
	const moves = 4
	driver, model, _ := start(t, config{})
	for i := 0; i < moves; i++ {
		driver.Key("down")
	}
	want := mustSelected(t, model)
	if want.Reference() != "opencode:71aa30e4" {
		t.Fatalf("after %d downs the selection is %q; the fixture or the cursor rule changed",
			moves, want.Reference())
	}

	driver.Key("enter")

	intent := model.Intent()
	if intent.Action != tui.ActionResume {
		t.Fatalf("action = %q, want %q", intent.Action, tui.ActionResume)
	}
	if intent.Reference != want.Reference() {
		t.Fatalf("reference = %q, want %q", intent.Reference, want.Reference())
	}
	if !intent.Chosen() {
		t.Fatal("intent should report a choice")
	}
	if !model.quitting {
		t.Fatal("enter should quit the surface")
	}
	if frame := driver.View(); frame != "" {
		t.Fatalf("a quitting surface must leave no frame behind, got:\n%s", frame)
	}
	if model.Err() != nil {
		t.Fatalf("unexpected error: %v", model.Err())
	}
}

// TestListModeRunesFilterRatherThanAct is the reason the action menu exists.
// If a bare "f" ever forks, a user typing a session name destroys their work.
func TestListModeRunesFilterRatherThanAct(t *testing.T) {
	for _, key := range []string{"f", "h", "i", "r", "y", "a", "q"} {
		t.Run(key, func(t *testing.T) {
			driver, model, loader := start(t, config{})
			driver.Key(key)

			if model.Intent().Chosen() {
				t.Fatalf("key %q produced intent %+v in list mode", key, model.Intent())
			}
			if model.quitting {
				t.Fatalf("key %q quit the surface in list mode", key)
			}
			if model.filter != key {
				t.Fatalf("filter = %q, want %q", model.filter, key)
			}
			if got := loader.last(t).Query; got != key {
				t.Fatalf("loader query = %q, want %q", got, key)
			}
		})
	}
}

func TestActionMenu(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		wantAction tui.Action
		wantQuit   bool
		wantMode   mode
	}{
		{name: "resume", key: "r", wantAction: tui.ActionResume, wantQuit: true, wantMode: modeList},
		{name: "fork", key: "f", wantAction: tui.ActionFork, wantQuit: true, wantMode: modeList},
		{name: "handoff", key: "h", wantAction: tui.ActionHandoff, wantQuit: true, wantMode: modeList},
		{name: "inspect", key: "i", wantAction: tui.ActionInspect, wantQuit: true, wantMode: modeList},
		{name: "enter resumes", key: "enter", wantAction: tui.ActionResume, wantQuit: true, wantMode: modeList},
		{name: "quit cancels", key: "q", wantAction: tui.ActionNone, wantQuit: true, wantMode: modeList},
		{name: "ctrl+c cancels", key: "ctrl+c", wantAction: tui.ActionNone, wantQuit: true, wantMode: modeActions},
		{name: "esc returns to the list", key: "esc", wantAction: tui.ActionNone, wantQuit: false, wantMode: modeList},
		{name: "tab returns to the list", key: "tab", wantAction: tui.ActionNone, wantQuit: false, wantMode: modeList},
		{name: "unbound key is inert", key: "z", wantAction: tui.ActionNone, wantQuit: false, wantMode: modeList},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, model, _ := start(t, config{})
			driver.Keys("down", "down")
			want := mustSelected(t, model)

			driver.Key("tab")
			if model.mode != modeActions {
				t.Fatal("tab did not open the action menu")
			}
			if !strings.Contains(driver.View(), "actions") {
				t.Error("the action menu is not visible in the frame")
			}

			driver.Key(test.key)

			if model.mode != test.wantMode {
				t.Errorf("mode = %d, want %d", model.mode, test.wantMode)
			}
			if got := model.Intent().Action; got != test.wantAction {
				t.Fatalf("action = %q, want %q", got, test.wantAction)
			}
			if model.quitting != test.wantQuit {
				t.Errorf("quitting = %v, want %v", model.quitting, test.wantQuit)
			}
			if test.wantAction == tui.ActionNone {
				assertZeroIntent(t, model.Intent())
				return
			}
			if got := model.Intent().Reference; got != want.Reference() {
				t.Errorf("reference = %q, want %q", got, want.Reference())
			}
			// The filter must never see the accelerator.
			if model.filter != "" {
				t.Errorf("accelerator %q leaked into the filter as %q", test.key, model.filter)
			}
		})
	}
}

func TestTabIsInertWithNothingSelected(t *testing.T) {
	driver, model, _ := start(t, config{noRecords: true})
	driver.Key("tab")
	if model.mode != modeList {
		t.Fatal("tab opened the action menu with no session selected")
	}
}

func TestTypingFiltersThroughTheLoader(t *testing.T) {
	driver, model, loader := start(t, config{})

	driver.Type("auth")

	want := sessionindex.Filter{Query: "auth", Limit: sessionindex.DefaultLimit}
	if got := loader.last(t); got != want {
		t.Fatalf("filter = %+v, want %+v", got, want)
	}
	// One load per keystroke, each carrying the query typed so far.
	wantQueries := []string{"", "a", "au", "aut", "auth"}
	var gotQueries []string
	for _, filter := range loader.filters {
		gotQueries = append(gotQueries, filter.Query)
	}
	if strings.Join(gotQueries, "|") != strings.Join(wantQueries, "|") {
		t.Fatalf("queries = %v, want %v", gotQueries, wantQueries)
	}
	if model.filter != "auth" {
		t.Fatalf("filter text = %q, want %q", model.filter, "auth")
	}
	// The fake loader ignores the query, so every record must still be shown:
	// the view is not allowed to filter a second time.
	if len(model.records) != len(fixtureRecords()) {
		t.Fatalf("view kept %d of %d records; filtering belongs to the index",
			len(model.records), len(fixtureRecords()))
	}
	if !strings.Contains(driver.View(), "auth") {
		t.Error("the filter text is not shown in the frame")
	}
}

func TestSpaceExtendsTheFilter(t *testing.T) {
	driver, model, loader := start(t, config{})
	driver.Type("auth")
	driver.Key(" ")
	driver.Type("token")

	if model.filter != "auth token" {
		t.Fatalf("filter = %q, want %q", model.filter, "auth token")
	}
	if got := loader.last(t).Query; got != "auth token" {
		t.Fatalf("loader query = %q, want %q", got, "auth token")
	}
}

func TestBackspaceAndEscape(t *testing.T) {
	t.Run("backspace shortens the filter", func(t *testing.T) {
		driver, model, loader := start(t, config{})
		driver.Type("auth")
		before := loader.calls()

		driver.Key("backspace")

		if model.filter != "aut" {
			t.Fatalf("filter = %q, want %q", model.filter, "aut")
		}
		if loader.calls() != before+1 {
			t.Fatalf("loader calls = %d, want %d", loader.calls(), before+1)
		}
		if got := loader.last(t).Query; got != "aut" {
			t.Fatalf("loader query = %q, want %q", got, "aut")
		}
	})

	t.Run("backspace on an empty filter does not reload", func(t *testing.T) {
		driver, model, loader := start(t, config{})
		before := loader.calls()

		driver.Key("backspace")

		if loader.calls() != before {
			t.Fatalf("backspace on an empty filter reloaded %d times", loader.calls()-before)
		}
		if model.filter != "" {
			t.Fatalf("filter = %q, want empty", model.filter)
		}
	})

	t.Run("escape clears the filter without quitting", func(t *testing.T) {
		driver, model, loader := start(t, config{})
		driver.Type("auth")

		driver.Key("esc")

		if model.filter != "" {
			t.Fatalf("filter = %q, want it cleared", model.filter)
		}
		if model.quitting {
			t.Fatal("escape quit the surface while a filter was set")
		}
		if model.Intent().Chosen() {
			t.Fatalf("escape produced intent %+v", model.Intent())
		}
		if got := loader.last(t).Query; got != "" {
			t.Fatalf("loader query after clearing = %q, want empty", got)
		}
	})

	t.Run("escape on an empty filter cancels", func(t *testing.T) {
		driver, model, _ := start(t, config{})

		driver.Key("esc")

		assertZeroIntent(t, model.Intent())
		if !model.quitting {
			t.Fatal("escape on an empty filter should quit")
		}
	})

	t.Run("ctrl+c always cancels", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		driver.Type("auth")

		driver.Key("ctrl+c")

		assertZeroIntent(t, model.Intent())
		if !model.quitting {
			t.Fatal("ctrl+c should quit")
		}
	})
}

func TestScopeToggle(t *testing.T) {
	t.Run("inside a project", func(t *testing.T) {
		driver, model, loader := start(t, config{project: projectReinstate})

		if model.scope != ScopeProject {
			t.Fatal("a switcher started inside a project should scope to it")
		}
		if got := loader.last(t).Project; got != projectReinstate {
			t.Fatalf("initial filter project = %q, want %q", got, projectReinstate)
		}

		driver.Key("ctrl+a")
		if model.scope != ScopeAll {
			t.Fatal("ctrl+a did not widen the scope")
		}
		if got := loader.last(t).Project; got != "" {
			t.Fatalf("filter project after widening = %q, want empty", got)
		}
		if !strings.Contains(driver.View(), "all projects") {
			t.Error("the widened scope is not shown in the header")
		}

		driver.Key("ctrl+a")
		if model.scope != ScopeProject {
			t.Fatal("ctrl+a did not narrow the scope back")
		}
		if got := loader.last(t).Project; got != projectReinstate {
			t.Fatalf("filter project after narrowing = %q, want %q", got, projectReinstate)
		}
		if !strings.Contains(driver.View(), projectReinstate) {
			t.Error("the project scope is not shown in the header")
		}
	})

	t.Run("outside a project", func(t *testing.T) {
		driver, model, loader := start(t, config{})

		if model.scope != ScopeAll {
			t.Fatal("a switcher started outside a project should show every session")
		}
		before := loader.calls()

		driver.Key("ctrl+a")

		if model.scope != ScopeAll {
			t.Fatal("ctrl+a changed the scope with no project to scope to")
		}
		if loader.calls() != before {
			t.Fatalf("ctrl+a reloaded %d times, want a no-op", loader.calls()-before)
		}
		if model.status == "" {
			t.Fatal("ctrl+a should explain why it did nothing")
		}
		if !strings.Contains(driver.View(), model.status) {
			t.Errorf("status %q is not visible in the frame", model.status)
		}
	})

	t.Run("the scope keeps the typed filter", func(t *testing.T) {
		driver, model, loader := start(t, config{project: projectReinstate})
		driver.Type("auth")

		driver.Key("ctrl+a")

		want := sessionindex.Filter{Query: "auth", Limit: sessionindex.DefaultLimit}
		if got := loader.last(t); got != want {
			t.Fatalf("filter = %+v, want %+v", got, want)
		}
		if model.filter != "auth" {
			t.Fatalf("filter text = %q, want it preserved", model.filter)
		}
	})

	t.Run("the menu accelerator toggles too", func(t *testing.T) {
		driver, model, _ := start(t, config{project: projectReinstate})
		driver.Keys("tab", "a")
		if model.scope != ScopeAll {
			t.Fatal("the menu's a accelerator did not widen the scope")
		}
	})
}

func TestLimitReachesTheLoader(t *testing.T) {
	_, _, loader := start(t, config{limit: 7})
	if got := loader.last(t).Limit; got != 7 {
		t.Fatalf("limit = %d, want 7", got)
	}
}

func TestCursorSurvivesReload(t *testing.T) {
	all := fixtureRecords()
	// The third session row, chosen so it is neither first nor last.
	target := all[2]

	t.Run("the same session is still present", func(t *testing.T) {
		remaining := []sessionindex.Record{all[0], target, all[5]}
		phase := 0
		driver, model, _ := start(t, config{
			respond: func(sessionindex.Filter) ([]sessionindex.Record, error) {
				phase++
				if phase == 1 {
					return all, nil
				}
				return remaining, nil
			},
		})
		driver.Keys("down", "down")
		if got := mustSelected(t, model); got.Key != target.Key {
			t.Fatalf("selected %q, want %q", got.Key, target.Key)
		}

		driver.Key("ctrl+r")

		got := mustSelected(t, model)
		if got.Key != target.Key {
			t.Fatalf("after a reload the cursor moved to %q, want %q", got.Key, target.Key)
		}
		assertOnSession(t, model, "after reload")
	})

	t.Run("the same session is gone", func(t *testing.T) {
		remaining := []sessionindex.Record{all[0], all[5]}
		phase := 0
		driver, model, _ := start(t, config{
			respond: func(sessionindex.Filter) ([]sessionindex.Record, error) {
				phase++
				if phase == 1 {
					return all, nil
				}
				return remaining, nil
			},
		})
		driver.Keys("down", "down")

		driver.Key("ctrl+r")

		got := mustSelected(t, model)
		if got.Key != all[0].Key {
			t.Fatalf("after the selection vanished the cursor is on %q, want the first row %q",
				got.Key, all[0].Key)
		}
		if model.cursor != sessionRows(model)[0] {
			t.Fatalf("cursor = %d, want the first session row %d", model.cursor, sessionRows(model)[0])
		}
	})

	t.Run("an emptied list leaves nothing selected", func(t *testing.T) {
		phase := 0
		driver, model, _ := start(t, config{
			respond: func(sessionindex.Filter) ([]sessionindex.Record, error) {
				phase++
				if phase == 1 {
					return all, nil
				}
				return nil, nil
			},
		})
		driver.Keys("down", "down")

		driver.Key("ctrl+r")

		if model.selected() != nil {
			t.Fatal("an empty list must select nothing")
		}
		if model.cursor != 0 || model.offset != 0 {
			t.Fatalf("cursor/offset = %d/%d, want 0/0", model.cursor, model.offset)
		}
		driver.Key("enter")
		if model.Intent().Chosen() {
			t.Fatalf("enter on an empty list produced %+v", model.Intent())
		}
	})
}

// TestScrollingKeepsTheCursorVisible drives a short viewport to the end and
// checks the viewport invariant after every move, including that the section
// heading directly above the cursor stays on screen.
func TestScrollingKeepsTheCursorVisible(t *testing.T) {
	const height = 12
	driver, model, _ := start(t, config{height: height})
	listHeight := model.listHeight()
	if listHeight >= len(model.rows) {
		t.Fatalf("viewport of %d rows is not shorter than the %d-row list", listHeight, len(model.rows))
	}

	check := func(context string) {
		t.Helper()
		if model.offset < 0 {
			t.Fatalf("%s: offset %d is negative", context, model.offset)
		}
		if model.cursor < model.offset || model.cursor >= model.offset+listHeight {
			t.Fatalf("%s: cursor %d is outside the viewport [%d,%d)",
				context, model.cursor, model.offset, model.offset+listHeight)
		}
		if model.cursor > 0 && model.rows[model.cursor-1].isHeader() && model.offset > model.cursor-1 {
			t.Fatalf("%s: heading %q above the cursor scrolled off (offset %d, cursor %d)",
				context, model.rows[model.cursor-1].header, model.offset, model.cursor)
		}
		// The viewport may never scroll past the end of the list.
		if max := len(model.rows) - listHeight; model.offset > max && max >= 0 {
			t.Fatalf("%s: offset %d scrolled past the last page at %d", context, model.offset, max)
		}
	}

	check("initial")
	for step := 0; step < len(model.rows)+2; step++ {
		driver.Key("down")
		check(fmt.Sprintf("down %d", step))
	}
	if model.offset == 0 {
		t.Fatal("driving to the end never scrolled the viewport")
	}
	for step := 0; step < len(model.rows)+2; step++ {
		driver.Key("up")
		check(fmt.Sprintf("up %d", step))
	}
	if model.offset != 0 {
		t.Fatalf("returning to the top left the viewport at offset %d", model.offset)
	}
	for _, key := range []string{"end", "pgup", "pgdown", "home", "pgdown", "end", "home"} {
		driver.Key(key)
		check(key)
	}

	// A heading immediately above the cursor is the case the viewport has to
	// special-case: scrolling back up would otherwise leave the selected row
	// orphaned at the top with its section lost. Assert the fixture reaches it.
	reached := false
	driver.Key("end")
	for step := 0; step < len(model.rows); step++ {
		driver.Key("up")
		check(fmt.Sprintf("heading walk up %d", step))
		if model.cursor > 0 && model.rows[model.cursor-1].isHeader() && model.offset == model.cursor-1 {
			reached = true
		}
	}
	if !reached {
		t.Fatal("the fixture never scrolled a heading to the top of the viewport")
	}
}

func TestResizeKeepsTheCursorVisible(t *testing.T) {
	driver, model, _ := start(t, config{height: 40})
	driver.Key("end")
	driver.Resize(80, 12)

	listHeight := model.listHeight()
	if model.cursor < model.offset || model.cursor >= model.offset+listHeight {
		t.Fatalf("after a resize the cursor %d is outside the viewport [%d,%d)",
			model.cursor, model.offset, model.offset+listHeight)
	}
	assertFrameWidth(t, driver.View(), 80)
}

func TestEmptyStates(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config
		typed   string
		want    []string
		notWant []string
	}{
		{
			name: "nothing indexed",
			cfg:  config{noRecords: true, width: 100, height: 30},
			want: []string{
				"No coding-agent sessions found on this device.",
				"rein doctor --agents",
			},
		},
		{
			name:  "nothing matches the filter",
			cfg:   config{noRecords: true, width: 100, height: 30},
			typed: "zzz",
			want: []string{
				"No sessions match",
				"zzz",
				"Backspace to widen the filter.",
			},
			notWant: []string{"No sessions indexed for"},
		},
		{
			name:  "nothing matches inside the project",
			cfg:   config{noRecords: true, width: 100, height: 30, project: projectReinstate},
			typed: "zzz",
			want: []string{
				"No sessions match",
				"ctrl+a searches every project.",
			},
		},
		{
			name: "nothing indexed for the project",
			cfg:  config{noRecords: true, width: 100, height: 30, project: projectReinstate},
			want: []string{
				"No sessions indexed for",
				projectReinstate,
				"ctrl+a shows every project.",
			},
			notWant: []string{"No coding-agent sessions found"},
		},
		{
			name: "the index could not be read",
			cfg: config{
				noRecords: true,
				width:     100,
				height:    30,
				loadErr:   errors.New("open index.db: permission denied"),
			},
			want: []string{
				"could not read the local session index",
				"open index.db: permission denied",
			},
			notWant: []string{"No coding-agent sessions found"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, model, _ := start(t, test.cfg)
			if test.typed != "" {
				driver.Type(test.typed)
			}
			frame := driver.View()
			for _, want := range test.want {
				if !strings.Contains(frame, want) {
					t.Errorf("frame does not mention %q\n%s", want, frame)
				}
			}
			for _, notWant := range test.notWant {
				if strings.Contains(frame, notWant) {
					t.Errorf("frame should not mention %q\n%s", notWant, frame)
				}
			}
			assertFrameWidth(t, frame, test.cfg.width)
			if model.selected() != nil {
				t.Error("an empty list must select nothing")
			}
		})
	}
}

func TestClipboardCopiesTheReference(t *testing.T) {
	t.Run("with a clipboard", func(t *testing.T) {
		var copied []string
		driver, model, _ := start(t, config{
			clipboard: func(text string) error {
				copied = append(copied, text)
				return nil
			},
		})
		driver.Keys("down", "down")
		want := mustSelected(t, model)

		driver.Keys("tab", "y")

		if len(copied) != 1 {
			t.Fatalf("clipboard called %d times, want once", len(copied))
		}
		if copied[0] != want.Reference() {
			t.Fatalf("copied %q, want %q", copied[0], want.Reference())
		}
		if copied[0] != "gemini:a13f7702" {
			t.Fatalf("copied %q, want an agent:id reference", copied[0])
		}
		if model.status != "copied "+want.Reference() {
			t.Fatalf("status = %q, want %q", model.status, "copied "+want.Reference())
		}
		if !strings.Contains(driver.View(), model.status) {
			t.Errorf("status %q is not visible in the frame", model.status)
		}
		if model.quitting {
			t.Fatal("copying should not quit the surface")
		}
		if model.mode != modeList {
			t.Fatal("copying should return to the list")
		}
	})

	t.Run("a failing clipboard reports why", func(t *testing.T) {
		driver, model, _ := start(t, config{
			clipboard: func(string) error { return errors.New("no terminal") },
		})
		driver.Keys("tab", "y")
		if !strings.Contains(model.status, "could not copy") {
			t.Fatalf("status = %q, want a copy failure", model.status)
		}
	})

	t.Run("without a clipboard the reference is still shown", func(t *testing.T) {
		driver, model, _ := start(t, config{})
		want := mustSelected(t, model)
		driver.Keys("tab", "y")
		if model.status != want.Reference() {
			t.Fatalf("status = %q, want the reference %q", model.status, want.Reference())
		}
	})
}

// TestDaemonLineFillsTheStatusLine covers the daemon summary the CLI passes
// in: it is the status line while nothing transient is being said, and a
// transient message wins, then gives the line back.
func TestDaemonLineFillsTheStatusLine(t *testing.T) {
	const line = `daemon running · pushed just now · pulled just now · 2 device(s) · "desktop" wants to join: rein devices approve`
	driver, model, _ := start(t, config{daemon: line})
	if !strings.Contains(driver.View(), line) {
		t.Fatalf("daemon line missing from the frame:\n%s", driver.View())
	}

	// A transient status (the reference shown when there is no clipboard)
	// takes the line over.
	want := mustSelected(t, model)
	driver.Keys("tab", "y")
	frame := driver.View()
	if model.status != want.Reference() || !strings.Contains(frame, want.Reference()) {
		t.Fatalf("transient status %q should be shown, status=%q", want.Reference(), model.status)
	}
	if strings.Contains(frame, "daemon running") {
		t.Fatal("the transient status must replace the daemon line, not sit beside it")
	}

	// Clearing the transient status hands the line back to the daemon.
	model.status = ""
	if !strings.Contains(driver.View(), "daemon running") {
		t.Fatal("daemon line should return once the transient status clears")
	}

	// Without a daemon summary and nothing to say, there is no status line.
	plain, _, _ := start(t, config{})
	if strings.Contains(plain.View(), "daemon") {
		t.Fatal("no daemon line expected when none was given")
	}
}

// TestUntrustedTitlesCannotRepaintTheTerminal is a security invariant, not a
// cosmetic one. Titles and prompt previews are read out of vendor session files,
// so an escape sequence surviving into a frame would let a session file drive
// the user's terminal.
func TestUntrustedTitlesCannotRepaintTheTerminal(t *testing.T) {
	sizes := []struct{ width, height int }{
		{120, 40}, {80, 24}, {70, 20}, {40, 10},
	}
	for _, size := range sizes {
		name := fmt.Sprintf("%dx%d", size.width, size.height)
		t.Run(name, func(t *testing.T) {
			mode := ui.ModeFull
			if size.width < ui.MinSplitWidth {
				mode = ui.ModeCompact
			}
			driver, model, _ := start(t, config{width: size.width, height: size.height, mode: mode})
			// Walk the whole list so the hostile record is rendered selected, in
			// the list, and in the preview pane.
			for step := 0; step < len(model.rows); step++ {
				frame := driver.View()
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("frame contains a raw escape byte at cursor %d:\n%q", model.cursor, frame)
				}
				if lines := strings.Count(frame, "\n") + 1; lines > size.height {
					t.Fatalf("frame has %d lines, terminal has %d", lines, size.height)
				}
				assertFrameWidth(t, frame, size.width)
				driver.Key("down")
			}
		})
	}
}

func TestHostileTitleIsRenderedInert(t *testing.T) {
	driver, model, _ := start(t, config{})
	driver.Keys("down", "down", "down")
	record := mustSelected(t, model)
	if !strings.ContainsRune(record.Title, 0x1b) {
		t.Fatalf("the fixture at this row is %q; it should be the hostile title", record.Title)
	}
	frame := driver.View()
	if strings.ContainsRune(frame, 0x1b) {
		t.Fatal("the hostile title reached the frame with its escape intact")
	}
	if !strings.Contains(frame, "[31mmalicious") {
		t.Errorf("the hostile title should still be readable, with the escape removed:\n%s", frame)
	}
}

// TestFrameWidthNeverExceedsTerminal is the layout regression net: every state
// of the surface, at every supported size, measured in display cells.
func TestFrameWidthNeverExceedsTerminal(t *testing.T) {
	sizes := []struct{ width, height int }{
		{40, 10}, // the narrowest interactive terminal ui.Detect will hand us
		{52, 14},
		{70, 20},
		{79, 24},
		{80, 24},
		{100, 30},
		{120, 40},
		{200, 50},
	}
	states := []struct {
		name string
		cfg  config
		keys []string
	}{
		{name: "list"},
		{name: "scrolled", keys: []string{"end"}},
		{name: "actions", keys: []string{"tab"}},
		{name: "filtered", keys: []string{"a", "u", "t", "h"}},
		{name: "status", keys: []string{"tab", "y"}},
		{name: "no readiness column", cfg: config{plainReadiness: true}},
		{name: "scoped", cfg: config{project: projectWebsite}},
		{name: "empty", cfg: config{noRecords: true}},
		{name: "load error", cfg: config{noRecords: true, loadErr: errors.New("open index.db: permission denied")}},
	}
	for _, size := range sizes {
		for _, state := range states {
			name := fmt.Sprintf("%dx%d/%s", size.width, size.height, state.name)
			t.Run(name, func(t *testing.T) {
				cfg := state.cfg
				cfg.width, cfg.height = size.width, size.height
				if size.width < ui.MinSplitWidth {
					cfg.mode = ui.ModeCompact
				} else {
					cfg.mode = ui.ModeFull
				}
				driver, _, _ := start(t, cfg)
				driver.Keys(state.keys...)
				assertFrameWidth(t, driver.View(), size.width)
			})
		}
	}
}

func TestFrameHeightFitsTheTerminal(t *testing.T) {
	for _, size := range []struct{ width, height int }{{120, 40}, {80, 24}, {70, 20}, {40, 10}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			mode := ui.ModeFull
			if size.width < ui.MinSplitWidth {
				mode = ui.ModeCompact
			}
			driver, _, _ := start(t, config{width: size.width, height: size.height, mode: mode})
			frame := driver.Model().View()
			if lines := strings.Count(frame, "\n") + 1; lines != size.height {
				t.Fatalf("frame has %d lines, terminal has %d", lines, size.height)
			}
		})
	}
}

// TestGoldenFrames pins what a user actually sees. Regenerate with
// `go test ./internal/tui/switcher/ -update-golden` and review the diff: an
// unexplained change to one of these is a regression in the surface.
func TestGoldenFrames(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config
		keys   []string
		golden string
	}{
		{
			name:   "full 120x40",
			cfg:    config{width: 120, height: 40, mode: ui.ModeFull, project: projectReinstate},
			golden: "switcher_full_120x40",
		},
		{
			name:   "full 80x24",
			cfg:    config{width: 80, height: 24, mode: ui.ModeFull, project: projectReinstate},
			keys:   []string{"down", "down"},
			golden: "switcher_full_80x24",
		},
		{
			name:   "compact 70x20",
			cfg:    config{width: 70, height: 20, mode: ui.ModeCompact, project: projectReinstate},
			keys:   []string{"down"},
			golden: "switcher_compact_70x20",
		},
		{
			name:   "actions 120x40",
			cfg:    config{width: 120, height: 40, mode: ui.ModeFull, project: projectReinstate},
			keys:   []string{"down", "down", "down", "tab"},
			golden: "switcher_actions_120x40",
		},
		{
			name:   "empty 100x30",
			cfg:    config{width: 100, height: 30, mode: ui.ModeFull, project: projectReinstate, noRecords: true},
			keys:   []string{"a", "u", "t", "h"},
			golden: "switcher_empty_100x30",
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
}

// TestUnhandledMessagesAreInert guards the update loop against reacting to
// messages it does not own.
func TestUnhandledMessagesAreInert(t *testing.T) {
	driver, model, loader := start(t, config{})
	before := driver.View()
	calls := loader.calls()

	driver.Send(tea.KeyMsg{Type: tea.KeyLeft})
	driver.Send(struct{ unknown int }{})

	if got := driver.View(); got != before {
		t.Error("an unhandled message changed the frame")
	}
	if loader.calls() != calls {
		t.Error("an unhandled message triggered a reload")
	}
	if model.Intent().Chosen() || model.quitting {
		t.Error("an unhandled message resolved the surface")
	}
}

// TestBlockedReadOnlySessionExplainsWhy asserts the preview names the reason a
// session cannot resume.
//
// A read-only agent is blocked by the index, not by the environment, and
// nothing the reader does will change that. "CANNOT RESUME" on its own reads
// like a fault to go and fix; the reason is what tells them it is not.
func TestBlockedReadOnlySessionExplainsWhy(t *testing.T) {
	const reason = "Grok Build sessions are source-only in Phase 4"
	records := []sessionindex.Record{{
		Key:            "grok:demo",
		ID:             "demo",
		Agent:          "grok",
		Title:          "Probe the vendor session paths",
		Project:        "reinstate",
		UpdatedAt:      fixtureNow().Add(-time.Hour),
		CanResume:      false,
		ReadOnlyReason: reason,
	}}

	for _, test := range []struct {
		name      string
		readiness bool
	}{
		{name: "with readiness being computed", readiness: true},
		{name: "with readiness disabled", readiness: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver, _, _ := start(t, config{records: records, plainReadiness: !test.readiness})
			frame := driver.View()
			// The reason wraps to the preview pane's width, so the assertion
			// is on the text rather than on where the line breaks fall.
			if !strings.Contains(flattenFrame(frame), reason) {
				t.Fatalf("the reason %q is nowhere on screen:\n%s", reason, frame)
			}
		})
	}
}

// flattenFrame joins a rendered frame into one whitespace-collapsed line so a
// test can assert that some text is present without pinning the column it
// happens to wrap at.
//
// The pane separator is dropped first: it sits between the two columns on every
// row, so text wrapped across rows would otherwise have a separator spliced
// into the middle of it.
func flattenFrame(frame string) string {
	replacer := strings.NewReplacer("\n", " ", "│", " ", "|", " ")
	return strings.Join(strings.Fields(replacer.Replace(frame)), " ")
}
