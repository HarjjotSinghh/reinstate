// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package palette

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// TestMain pins the rendering environment. lipgloss resolves its colour profile
// from stdout exactly once, on the first render, so forcing NO_COLOR before any
// frame is drawn makes a rendered overlay identical whether the suite runs under
// `go test` (piped stdout) or as a bare test binary in a colour terminal.
func TestMain(m *testing.M) {
	if err := os.Setenv("NO_COLOR", "1"); err != nil {
		fmt.Fprintln(os.Stderr, "set NO_COLOR:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// fixtureCommands is the synthetic table behind the tests in this file. It is
// deliberately a lookalike of the switcher's real table rather than the table
// itself: these tests are about the palette's own rules, so they must not fail
// the day a product decision renames a command. It carries more entries than
// maxVisible so the scrolling window is always under test, and every Title fits
// in the 20-cell title column so a row can be matched by substring.
//
// The real table is exercised against the palette in commands_test.go.
func fixtureCommands() []Command {
	return []Command{
		{ID: "resume", Title: "Resume session", Detail: "continue in its own agent", Keys: []string{"continue", "open"}, NeedsSession: true},
		{ID: "fork", Title: "Fork session", Detail: "branch the transcript", Keys: []string{"branch"}, NeedsSession: true},
		{ID: "handoff", Title: "Hand off session", Detail: "brief another agent", Keys: []string{"transfer"}, NeedsSession: true},
		{ID: "inspect", Title: "Inspect session", Detail: "full metadata report", Keys: []string{"details"}, NeedsSession: true},
		{ID: "copy", Title: "Copy reference", Detail: "put agent:id on the clipboard", Keys: []string{"yank"}, NeedsSession: true},
		{ID: "scope", Title: "Toggle scope", Detail: "this project or every project"},
		{ID: "refresh", Title: "Refresh the index", Detail: "rescan every agent now", Keys: []string{"rescan"}},
		{ID: "doctor", Title: "Run diagnostics", Detail: "rein doctor", Keys: []string{"health"}},
		{ID: "status", Title: "Sync status", Detail: "rein status"},
		{ID: "push", Title: "Push sessions", Detail: "rein push", Keys: []string{"upload"}},
		{ID: "pull", Title: "Pull sessions", Detail: "rein pull", Keys: []string{"download"}},
		{ID: "quit", Title: "Quit", Detail: "close the switcher", Keys: []string{"exit"}},
	}
}

// fixtureTheme is the monochrome Unicode theme every test renders with. Colour
// is off so an assertion about escape bytes means what it says.
func fixtureTheme() ui.Theme {
	return ui.NewTheme(ui.Capability{Mode: ui.ModeFull, Color: ui.ColorNone, Unicode: true, Width: 110, Height: 30})
}

// open builds a palette over the fixture table.
func open(width, height int) *Model {
	return New(fixtureTheme(), fixtureCommands(), width, height)
}

// press sends one key and asserts the overlay consumed it. Consumption is not
// incidental: while the palette is open it owns the keyboard, so any key it
// declines would also reach the surface underneath.
func press(t *testing.T, m *Model, key tea.KeyMsg) {
	t.Helper()
	if !m.Update(key) {
		t.Fatalf("key %+v was not consumed by the open overlay", key)
	}
}

// typeInto sends s one key at a time, the way a person filtering a list
// actually produces input. Spaces arrive as tea.KeySpace, as they do from a
// real terminal, rather than as a rune.
func typeInto(t *testing.T, m *Model, s string) {
	t.Helper()
	for _, r := range s {
		if r == ' ' {
			press(t, m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
			continue
		}
		press(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func key(t *testing.T, m *Model, keyType tea.KeyType) {
	t.Helper()
	press(t, m, tea.KeyMsg{Type: keyType})
}

// filteredIDs is the visible result list, in rank order.
func filteredIDs(m *Model) []string {
	ids := make([]string, 0, len(m.filtered))
	for _, command := range m.filtered {
		ids = append(ids, command.ID)
	}
	return ids
}

func joinIDs(ids []string) string { return strings.Join(ids, ",") }

// filter builds a palette and types query into it.
func filter(t *testing.T, query string) *Model {
	t.Helper()
	model := open(110, 30)
	typeInto(t, model, query)
	return model
}

// TestSubsequenceScoreMatchesTheWayPeopleType is the matching contract: the
// query's letters must appear in order, but need not be adjacent, so "hof"
// finds "hand off" the way a palette user expects.
func TestSubsequenceScoreMatchesTheWayPeopleType(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		query     string
		match     bool
	}{
		{name: "empty query matches anything", candidate: "resume session", query: "", match: true},
		{name: "empty query on empty candidate", candidate: "", query: "", match: true},
		{name: "empty candidate cannot match", candidate: "", query: "a", match: false},
		{name: "exact", candidate: "quit", query: "quit", match: true},
		{name: "prefix", candidate: "resume session", query: "res", match: true},
		{name: "infix", candidate: "resume session", query: "sume", match: true},
		{name: "suffix", candidate: "resume session", query: "sion", match: true},
		{name: "gapped across a word boundary", candidate: "hand off", query: "hof", match: true},
		{name: "gapped initials", candidate: "toggle project scope", query: "tps", match: true},
		{name: "gapped inside one word", candidate: "diagnostics", query: "dgc", match: true},
		{name: "every letter of the candidate", candidate: "quit", query: "qit", match: true},
		{name: "out of order is not a subsequence", candidate: "hand off", query: "foh", match: false},
		{name: "missing letter", candidate: "hand off", query: "hoft", match: false},
		{name: "longer than the candidate", candidate: "quit", query: "quitting", match: false},
		{name: "case sensitive at this level", candidate: "resume", query: "R", match: false},
		{name: "repeated letter needs repeats", candidate: "off", query: "fff", match: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			score := subsequenceScore(test.candidate, test.query)
			if test.match && score < 0 {
				t.Fatalf("subsequenceScore(%q, %q) = %d, want a match", test.candidate, test.query, score)
			}
			if !test.match && score >= 0 {
				t.Fatalf("subsequenceScore(%q, %q) = %d, want -1", test.candidate, test.query, score)
			}
			if test.query == "" && score != 0 {
				t.Fatalf("an empty query scored %d, want 0", score)
			}
		})
	}
}

// TestSubsequenceScoreRanksTighterEarlierMatchesBetter is what makes the
// palette usable rather than merely correct. The assertions are relative, never
// absolute, so the scoring function can be tuned without rewriting the test.
func TestSubsequenceScoreRanksTighterEarlierMatchesBetter(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		better string
		worse  string
		why    string
	}{
		{
			name: "start beats middle", query: "fork",
			better: "fork session", worse: "please fork session",
			why: "a match at the very start is what the reader meant",
		},
		{
			name: "start beats end", query: "sync",
			better: "sync status", worse: "the index sync",
			why: "the earlier the match lands, the better it reads",
		},
		{
			name: "contiguous beats gapped", query: "hof",
			better: "hoffman", worse: "hand off",
			why: "a whole word beats letters scattered across two",
		},
		{
			name: "tight gap beats wide gap", query: "rs",
			better: "resume", worse: "refresh the sessions",
			why: "a short span is a stronger signal than a long one",
		},
		{
			name: "full word beats initials", query: "push",
			better: "push sessions", worse: "put upstream shards home",
			why: "an initialism match is the weakest kind",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			better := subsequenceScore(test.better, test.query)
			worse := subsequenceScore(test.worse, test.query)
			if better < 0 || worse < 0 {
				t.Fatalf("both candidates must match: %q = %d, %q = %d", test.better, better, test.worse, worse)
			}
			if better >= worse {
				t.Fatalf("%q scored %d and %q scored %d; %s, so the first must rank lower",
					test.better, better, test.worse, worse, test.why)
			}
		})
	}

	t.Run("a contiguous match at position zero is the best possible score", func(t *testing.T) {
		const query = "resume"
		best := subsequenceScore("resume session", query)
		for _, candidate := range []string{"a resume", "r e s u m e", "the resume session", "rescue me"} {
			score := subsequenceScore(candidate, query)
			if score < 0 {
				continue
			}
			if score <= best {
				t.Errorf("subsequenceScore(%q, %q) = %d, which is not worse than the exact prefix match %d",
					candidate, query, score, best)
			}
		}
	})
}

func TestNewOpensWithEveryCommandVisible(t *testing.T) {
	model := open(110, 30)

	if !model.Open() {
		t.Fatal("a new palette must already be showing; nothing else opens it")
	}
	if model.Chosen() != "" {
		t.Fatalf("Chosen() = %q on a fresh palette, want empty", model.Chosen())
	}
	if model.cursor != 0 {
		t.Fatalf("cursor = %d on a fresh palette, want 0", model.cursor)
	}
	want := make([]string, 0, len(fixtureCommands()))
	for _, command := range fixtureCommands() {
		want = append(want, command.ID)
	}
	if got := joinIDs(filteredIDs(model)); got != joinIDs(want) {
		t.Fatalf("visible = %s, want every command in table order: %s", got, joinIDs(want))
	}
}

// TestTypingNarrowsAndBackspaceWidens walks one query in and back out again,
// checking the result set at every step. A palette that cannot be un-typed
// strands the reader who mistypes one letter.
func TestTypingNarrowsAndBackspaceWidens(t *testing.T) {
	model := open(110, 30)
	all := len(model.filtered)

	typeInto(t, model, "pu")
	if got := joinIDs(filteredIDs(model)); got != "pull,push" {
		t.Fatalf("after typing \"pu\" the matches are %s, want pull,push", got)
	}

	// "pus" still matches "Pull sessions" as a subsequence, but only loosely,
	// so the tight match rises to the top.
	typeInto(t, model, "s")
	if got := joinIDs(filteredIDs(model)); got != "push,pull" {
		t.Fatalf("after typing \"pus\" the matches are %s, want push,pull", got)
	}

	typeInto(t, model, "h")
	if got := joinIDs(filteredIDs(model)); got != "push" {
		t.Fatalf("after typing \"push\" the matches are %s, want push", got)
	}

	key(t, model, tea.KeyBackspace)
	key(t, model, tea.KeyBackspace)
	if got := joinIDs(filteredIDs(model)); got != "pull,push" {
		t.Fatalf("after backspacing to \"pu\" the matches are %s, want pull,push", got)
	}

	key(t, model, tea.KeyBackspace)
	key(t, model, tea.KeyBackspace)
	if len(model.filtered) != all {
		t.Fatalf("an emptied filter shows %d commands, want all %d back", len(model.filtered), all)
	}
	if model.filter != "" {
		t.Fatalf("filter = %q after backspacing it away, want empty", model.filter)
	}

	// Backspace on an empty filter is a no-op rather than an error.
	key(t, model, tea.KeyBackspace)
	if model.filter != "" || len(model.filtered) != all {
		t.Fatalf("backspace on an empty filter changed the state: filter %q, %d matches",
			model.filter, len(model.filtered))
	}

	t.Run("a space is part of the query", func(t *testing.T) {
		model := filter(t, "hand off")
		if got := joinIDs(filteredIDs(model)); got != "handoff" {
			t.Fatalf("matches for \"hand off\" = %s, want handoff", got)
		}
	})
}

// TestCursorResetsOnEveryRefilter is what makes enter safe. The reader types,
// looks at the top row, and presses enter; if the cursor survived a refilter it
// would be pointing at a row that has since moved or vanished.
func TestCursorResetsOnEveryRefilter(t *testing.T) {
	steps := []struct {
		name string
		act  func(t *testing.T, m *Model)
	}{
		{name: "typing a rune", act: func(t *testing.T, m *Model) { typeInto(t, m, "s") }},
		{name: "typing a space", act: func(t *testing.T, m *Model) { typeInto(t, m, " ") }},
		{name: "backspacing", act: func(t *testing.T, m *Model) { key(t, m, tea.KeyBackspace) }},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			model := filter(t, "se")
			key(t, model, tea.KeyDown)
			key(t, model, tea.KeyDown)
			if model.cursor == 0 {
				t.Fatal("the fixture must allow the cursor to move, or this test proves nothing")
			}

			step.act(t, model)

			if model.cursor != 0 {
				t.Fatalf("cursor = %d after %s, want 0 so enter runs the top match", model.cursor, step.name)
			}
		})
	}
}

// TestUppercaseQueryStillMatches covers the reader who leaves caps lock on, or
// who types a command the way it is printed.
func TestUppercaseQueryStillMatches(t *testing.T) {
	for _, query := range []string{"RESUME", "Resume", "ReSuMe", "RESUME SESSION"} {
		t.Run(query, func(t *testing.T) {
			model := filter(t, query)
			if len(model.filtered) == 0 {
				t.Fatalf("query %q matched nothing; case must not decide", query)
			}
			if got := model.filtered[0].ID; got != "resume" {
				t.Fatalf("top match for %q is %q, want resume", query, got)
			}
		})
	}
}

// TestMovementWrapsInBothDirections keeps the last entry one keystroke away
// from the first, which is the whole point of a short list.
func TestMovementWrapsInBothDirections(t *testing.T) {
	pairs := []struct {
		name string
		up   tea.KeyType
		down tea.KeyType
	}{
		{name: "arrows", up: tea.KeyUp, down: tea.KeyDown},
		{name: "ctrl+p and ctrl+n", up: tea.KeyCtrlP, down: tea.KeyCtrlN},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			model := open(110, 30)
			last := len(model.filtered) - 1
			if last < 2 {
				t.Fatalf("the fixture has %d commands, too few to test wrapping", len(model.filtered))
			}

			key(t, model, pair.up)
			if model.cursor != last {
				t.Fatalf("up from the first row landed on %d, want the last row %d", model.cursor, last)
			}

			key(t, model, pair.down)
			if model.cursor != 0 {
				t.Fatalf("down from the last row landed on %d, want the first row 0", model.cursor)
			}

			for step := 0; step < last; step++ {
				key(t, model, pair.down)
			}
			if model.cursor != last {
				t.Fatalf("after %d downs the cursor is %d, want %d", last, model.cursor, last)
			}
			key(t, model, pair.down)
			if model.cursor != 0 {
				t.Fatalf("down past the end landed on %d, want 0", model.cursor)
			}
		})
	}

	t.Run("moving with no matches is inert", func(t *testing.T) {
		model := filter(t, "zzz")
		if len(model.filtered) != 0 {
			t.Fatalf("query \"zzz\" matched %d commands, want none", len(model.filtered))
		}
		for _, keyType := range []tea.KeyType{tea.KeyDown, tea.KeyDown, tea.KeyUp, tea.KeyCtrlN, tea.KeyCtrlP} {
			key(t, model, keyType)
			if model.cursor != 0 {
				t.Fatalf("cursor = %d with nothing to select, want 0", model.cursor)
			}
		}
	})
}

// TestEnterChoosesTheHighlightedCommand is the palette's entire output.
func TestEnterChoosesTheHighlightedCommand(t *testing.T) {
	tests := []struct {
		name  string
		query string
		downs int
		want  string
	}{
		{name: "top of an unfiltered list", want: "resume"},
		{name: "after moving down", downs: 2, want: "handoff"},
		{name: "after wrapping to the end", downs: -1, want: "quit"},
		{name: "top of a filtered list", query: "diag", want: "doctor"},
		{name: "second row of a filtered list", query: "sessions", downs: 1, want: "push"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := filter(t, test.query)
			if test.downs < 0 {
				key(t, model, tea.KeyUp)
			}
			for step := 0; step < test.downs; step++ {
				key(t, model, tea.KeyDown)
			}
			highlighted := model.filtered[model.cursor].ID

			key(t, model, tea.KeyEnter)

			if model.Open() {
				t.Fatal("enter must close the overlay")
			}
			if model.Chosen() != test.want {
				t.Fatalf("Chosen() = %q, want %q", model.Chosen(), test.want)
			}
			if model.Chosen() != highlighted {
				t.Fatalf("Chosen() = %q but the highlighted row was %q", model.Chosen(), highlighted)
			}
		})
	}
}

// TestEnterWithNoMatchesChoosesNothing is a safety rule, not a nicety. Running
// a command the reader cannot see on screen is indistinguishable, from their
// side, from the palette doing something at random.
func TestEnterWithNoMatchesChoosesNothing(t *testing.T) {
	model := filter(t, "zzz")
	if len(model.filtered) != 0 {
		t.Fatalf("query \"zzz\" matched %d commands, want none", len(model.filtered))
	}

	key(t, model, tea.KeyEnter)

	if model.Open() {
		t.Fatal("enter must close the overlay even when it chooses nothing")
	}
	if model.Chosen() != "" {
		t.Fatalf("Chosen() = %q with nothing on screen, want empty", model.Chosen())
	}
}

// TestCancelKeysChooseNothing checks the exits.
func TestCancelKeysChooseNothing(t *testing.T) {
	tests := []struct {
		name    string
		keyType tea.KeyType
	}{
		{name: "esc", keyType: tea.KeyEsc},
		{name: "ctrl+c", keyType: tea.KeyCtrlC},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, query := range []string{"", "resume", "zzz"} {
				model := filter(t, query)
				key(t, model, test.keyType)

				if model.Open() {
					t.Fatalf("%s left the overlay open (query %q)", test.name, query)
				}
				if model.Chosen() != "" {
					t.Fatalf("%s chose %q (query %q), want nothing", test.name, model.Chosen(), query)
				}
				if lines := model.Lines(110); lines != nil {
					t.Fatalf("%s left %d lines on screen", test.name, len(lines))
				}
			}
		})
	}
}

// TestEveryKeyIsConsumed is the overlay contract with its host. The switcher
// forwards a key to the palette and acts on it too if the palette declines it,
// so a key that leaks would filter the list underneath, or worse, resume a
// session, while the reader believes they are typing into a search box.
func TestEveryKeyIsConsumed(t *testing.T) {
	keys := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{name: "rune", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}},
		{name: "uppercase rune", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}},
		{name: "digit", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}}},
		{name: "multi-byte rune", msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'認'}}},
		{name: "space", msg: tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}},
		{name: "backspace", msg: tea.KeyMsg{Type: tea.KeyBackspace}},
		{name: "up", msg: tea.KeyMsg{Type: tea.KeyUp}},
		{name: "down", msg: tea.KeyMsg{Type: tea.KeyDown}},
		{name: "left", msg: tea.KeyMsg{Type: tea.KeyLeft}},
		{name: "right", msg: tea.KeyMsg{Type: tea.KeyRight}},
		{name: "enter", msg: tea.KeyMsg{Type: tea.KeyEnter}},
		{name: "esc", msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "tab", msg: tea.KeyMsg{Type: tea.KeyTab}},
		{name: "home", msg: tea.KeyMsg{Type: tea.KeyHome}},
		{name: "end", msg: tea.KeyMsg{Type: tea.KeyEnd}},
		{name: "pgup", msg: tea.KeyMsg{Type: tea.KeyPgUp}},
		{name: "pgdown", msg: tea.KeyMsg{Type: tea.KeyPgDown}},
		{name: "ctrl+a", msg: tea.KeyMsg{Type: tea.KeyCtrlA}},
		{name: "ctrl+c", msg: tea.KeyMsg{Type: tea.KeyCtrlC}},
		{name: "ctrl+k", msg: tea.KeyMsg{Type: tea.KeyCtrlK}},
		{name: "ctrl+n", msg: tea.KeyMsg{Type: tea.KeyCtrlN}},
		{name: "ctrl+p", msg: tea.KeyMsg{Type: tea.KeyCtrlP}},
		{name: "ctrl+r", msg: tea.KeyMsg{Type: tea.KeyCtrlR}},
		{name: "f1, which the palette does not handle", msg: tea.KeyMsg{Type: tea.KeyF1}},
		{name: "shift+tab, which the palette does not handle", msg: tea.KeyMsg{Type: tea.KeyShiftTab}},
		{name: "delete, which the palette does not handle", msg: tea.KeyMsg{Type: tea.KeyDelete}},
	}
	for _, test := range keys {
		t.Run(test.name, func(t *testing.T) {
			model := open(110, 30)
			if !model.Update(test.msg) {
				t.Fatalf("key %q leaked to the host surface", test.name)
			}
		})
	}
}

// TestRankingIsDeterministic proves the sort never reshuffles. Two palettes
// asked the same question must answer in the same order, or the reader who
// reopens the palette and repeats a query finds enter runs something else.
func TestRankingIsDeterministic(t *testing.T) {
	for _, query := range []string{"", "s", "se", "session", "re", "o", "e", "sync"} {
		t.Run("query "+query, func(t *testing.T) {
			first := joinIDs(filteredIDs(filter(t, query)))
			for run := 0; run < 5; run++ {
				again := joinIDs(filteredIDs(filter(t, query)))
				if again != first {
					t.Fatalf("run %d ordered %q as %s, the first run ordered it %s", run, query, again, first)
				}
			}
		})
	}

	// Typing a query and backspacing back to it must land on the same order as
	// typing it once, since both paths go through the same refilter.
	t.Run("reached by a different route", func(t *testing.T) {
		direct := filter(t, "se")
		roundTrip := filter(t, "sess")
		key(t, roundTrip, tea.KeyBackspace)
		key(t, roundTrip, tea.KeyBackspace)
		if got, want := joinIDs(filteredIDs(roundTrip)), joinIDs(filteredIDs(direct)); got != want {
			t.Fatalf("backspacing to %q ordered the matches %s, typing it ordered them %s",
				roundTrip.filter, got, want)
		}
	})
}

// TestEqualScoresBreakAlphabetically pins the tiebreak itself. Without one, two
// equally good matches would come back in whatever order the sort happened to
// leave them, and the top row would be a coin toss.
func TestEqualScoresBreakAlphabetically(t *testing.T) {
	// Declaration order is neither alphabetical nor the expected result, so a
	// dropped tiebreak cannot pass by accident.
	tied := []Command{
		{ID: "sync", Title: "Sync now", Detail: "d"},
		{ID: "scope", Title: "Scope toggle", Detail: "d"},
		{ID: "search", Title: "Search index", Detail: "d"},
	}
	model := New(fixtureTheme(), tied, 110, 30)
	typeInto(t, model, "s")

	if got, want := joinIDs(filteredIDs(model)), "scope,search,sync"; got != want {
		t.Fatalf("tied matches ordered %s, want them alphabetical by title: %s", got, want)
	}
	for _, command := range model.filtered {
		if score := subsequenceScore(strings.ToLower(command.Title), "s"); score != 0 {
			t.Fatalf("%q scored %d, so this fixture is not testing a tie", command.Title, score)
		}
	}
}

// The advisory lines the overlay draws in place of, or after, the command rows.
const (
	noMatchMarker = "no command matches"
	moreMarker    = "more match"
)

// commandRows returns just the command rows of a rendered overlay.
//
// The frame is a rule, the prompt, a rule, the body, and a closing rule; the
// body is the command rows plus at most one advisory line. Peeling the frame
// here keeps every layout assertion below about what the reader is choosing
// between rather than about line arithmetic.
func commandRows(t *testing.T, lines []string) []string {
	t.Helper()
	if len(lines) < 4 {
		t.Fatalf("overlay has %d lines, too few to contain a body:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	rows := make([]string, 0, len(lines))
	for _, line := range lines[3 : len(lines)-1] {
		if strings.Contains(line, noMatchMarker) || strings.Contains(line, moreMarker) {
			continue
		}
		rows = append(rows, line)
	}
	return rows
}

// advisory returns the single advisory line, or empty when there is none.
func advisory(t *testing.T, lines []string) string {
	t.Helper()
	var found []string
	for _, line := range lines[3 : len(lines)-1] {
		if strings.Contains(line, noMatchMarker) || strings.Contains(line, moreMarker) {
			found = append(found, line)
		}
	}
	if len(found) > 1 {
		t.Fatalf("overlay drew %d advisory lines, want at most one: %q", len(found), found)
	}
	if len(found) == 0 {
		return ""
	}
	return strings.TrimSpace(found[0])
}

// windowStart locates the rendered rows within the filtered list and checks
// they are one contiguous run of it, which is what makes the overlay a window
// onto the results rather than an arbitrary selection from them.
func windowStart(t *testing.T, m *Model, rows []string) int {
	t.Helper()
	if len(rows) == 0 {
		return 0
	}
	start := -1
	for index, command := range m.filtered {
		if strings.Contains(rows[0], command.Title) {
			start = index
			break
		}
	}
	if start < 0 {
		t.Fatalf("the first rendered row matches no command: %q", rows[0])
	}
	for offset, row := range rows {
		if start+offset >= len(m.filtered) {
			t.Fatalf("row %d has no command behind it: %q", offset, row)
		}
		if want := m.filtered[start+offset].Title; !strings.Contains(row, want) {
			t.Fatalf("row %d is %q, want the window to continue with %q", offset, row, want)
		}
	}
	return start
}

// markedRow returns the single row carrying the selection marker.
func markedRow(t *testing.T, m *Model, rows []string) string {
	t.Helper()
	marker := " " + m.theme.Glyphs.Cursor + " "
	var found []string
	for _, row := range rows {
		if strings.HasPrefix(row, marker) {
			found = append(found, row)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%d rows carry the selection marker, want exactly one:\n%s", len(found), strings.Join(rows, "\n"))
	}
	return found[0]
}

func TestLinesRenderNothingWhenClosed(t *testing.T) {
	t.Run("after esc", func(t *testing.T) {
		model := open(110, 30)
		if len(model.Lines(110)) == 0 {
			t.Fatal("an open palette must render something, or the next assertion is vacuous")
		}
		key(t, model, tea.KeyEsc)
		if lines := model.Lines(110); lines != nil {
			t.Fatalf("a closed palette rendered %d lines", len(lines))
		}
	})
	t.Run("after enter", func(t *testing.T) {
		model := open(110, 30)
		key(t, model, tea.KeyEnter)
		if lines := model.Lines(110); lines != nil {
			t.Fatalf("a chosen palette rendered %d lines", len(lines))
		}
	})
	t.Run("nil model", func(t *testing.T) {
		var model *Model
		if lines := model.Lines(110); lines != nil {
			t.Fatalf("a nil palette rendered %d lines", len(lines))
		}
	})
}

// TestNoLineIsWiderThanTheOverlay is the layout contract. The palette is drawn
// over a list that is still on screen, so a line one cell too wide does not
// wrap harmlessly: it pushes the rest of the frame sideways.
func TestNoLineIsWiderThanTheOverlay(t *testing.T) {
	states := []struct {
		name  string
		query string
		keys  []tea.KeyType
	}{
		{name: "unfiltered", query: ""},
		{name: "more matches than fit", query: "e"},
		{name: "scrolled past the window", query: "", keys: []tea.KeyType{
			tea.KeyDown, tea.KeyDown, tea.KeyDown, tea.KeyDown,
			tea.KeyDown, tea.KeyDown, tea.KeyDown, tea.KeyDown, tea.KeyDown,
		}},
		{name: "wrapped to the last row", query: "", keys: []tea.KeyType{tea.KeyUp}},
		{name: "filtered to a few", query: "session"},
		{name: "filtered to one", query: "diagnostics"},
		{name: "no match", query: "zzz"},
		{name: "long query", query: "a query far longer than the prompt row can hold"},
		{name: "uppercase", query: "RESUME"},
	}
	for _, width := range []int{40, 70, 110, 200} {
		for _, state := range states {
			t.Run(fmt.Sprintf("%d/%s", width, state.name), func(t *testing.T) {
				model := open(width, 30)
				typeInto(t, model, state.query)
				for _, keyType := range state.keys {
					key(t, model, keyType)
				}

				lines := model.Lines(width)
				if len(lines) == 0 {
					t.Fatal("the overlay stopped rendering; this assertion would be vacuous")
				}
				for index, line := range lines {
					if got := ui.Width(line); got > width {
						t.Errorf("line %d is %d cells wide, the overlay is %d\n%q", index+1, got, width, line)
					}
				}
			})
		}
	}
}

// TestTheWindowIsBoundedAndFollowsTheCursor is why the palette reads as an
// overlay: it never grows past maxVisible rows, it says how many matches it is
// not showing, and the highlighted row is always one of the ones on screen.
func TestTheWindowIsBoundedAndFollowsTheCursor(t *testing.T) {
	model := open(110, 30)
	total := len(model.filtered)
	if total <= maxVisible {
		t.Fatalf("the fixture has %d commands, which does not exceed maxVisible (%d)", total, maxVisible)
	}

	for step := 0; step <= total; step++ {
		lines := model.Lines(110)
		rows := commandRows(t, lines)
		if len(rows) > maxVisible {
			t.Fatalf("step %d drew %d rows, the overlay allows %d", step, len(rows), maxVisible)
		}
		start := windowStart(t, model, rows)

		// The highlighted command must be one of the drawn rows, or the reader
		// is pressing enter on something they cannot see.
		selected := model.filtered[model.cursor]
		if !strings.Contains(markedRow(t, model, rows), selected.Title) {
			t.Fatalf("step %d marks a row that is not the selection %q:\n%s",
				step, selected.Title, strings.Join(rows, "\n"))
		}

		// The advisory counts every match that is off-screen, above the window
		// as well as below it. Counting only what is below would read as
		// "scroll down for the rest", which stops being true the moment the
		// list has scrolled at all.
		hidden := total - len(rows)
		want := ""
		if hidden > 0 {
			want = plural(hidden, "more match", "more matches")
		}
		_ = start
		if got := advisory(t, lines); got != want {
			t.Fatalf("step %d (cursor %d, window from %d) says %q, want %q", step, model.cursor, start, got, want)
		}
		key(t, model, tea.KeyDown)
	}

	t.Run("a short list needs no advisory", func(t *testing.T) {
		model := filter(t, "session")
		lines := model.Lines(110)
		if got := len(commandRows(t, lines)); got != len(model.filtered) || got == 0 {
			t.Fatalf("drew %d rows for %d matches, want all of them", got, len(model.filtered))
		}
		if got := advisory(t, lines); got != "" {
			t.Fatalf("a fully visible list says %q, want nothing", got)
		}
	})

	t.Run("the count is singular for one hidden match", func(t *testing.T) {
		// One more command than the window holds, so exactly one is hidden.
		commands := fixtureCommands()[:maxVisible+1]
		model := New(fixtureTheme(), commands, 110, 30)
		if got, want := advisory(t, model.Lines(110)), "1 more match"; got != want {
			t.Fatalf("advisory = %q, want %q", got, want)
		}
	})
}

// TestNoMatchStateNamesTheQuery keeps a dead end explicable: the reader must be
// able to see what was searched for, not just that nothing came back.
func TestNoMatchStateNamesTheQuery(t *testing.T) {
	model := filter(t, "zzz")
	lines := model.Lines(110)

	if rows := commandRows(t, lines); len(rows) != 0 {
		t.Fatalf("a no-match overlay drew %d command rows:\n%s", len(rows), strings.Join(rows, "\n"))
	}
	if got, want := advisory(t, lines), "no command matches zzz"; got != want {
		t.Fatalf("advisory = %q, want %q", got, want)
	}
	if got := strings.TrimSpace(lines[1]); !strings.HasSuffix(got, "zzz") {
		t.Fatalf("prompt row = %q, want it to echo the query", got)
	}
}

// TestHostileCommandTextCannotRepaintTheTerminal is a boundary test, not a
// prediction about today's table. Palette entries are internal now, but the
// view is the last thing between a command's text and the terminal, and it must
// not be the component that assumes its input is safe.
func TestHostileCommandTextCannotRepaintTheTerminal(t *testing.T) {
	hostile := []Command{
		{ID: "hostile", Title: "\x1b[31mrepaint\x1b[0m attempt", Detail: "\x1b]0;pwned\a title\tsetter"},
		{ID: "newline", Title: "two\nlines", Detail: "carriage\rreturn"},
		{ID: "safe", Title: "Quit", Detail: "close the switcher"},
	}
	states := []struct {
		name  string
		query string
		keys  []tea.KeyType
	}{
		{name: "fresh"},
		{name: "selected", keys: []tea.KeyType{tea.KeyUp}},
		{name: "filtered to the hostile row", query: "repaint"},
		{name: "no match", query: "zzz"},
	}
	for _, width := range []int{40, 70, 110, 200} {
		for _, state := range states {
			t.Run(fmt.Sprintf("%d/%s", width, state.name), func(t *testing.T) {
				model := New(fixtureTheme(), hostile, width, 30)
				typeInto(t, model, state.query)
				for _, keyType := range state.keys {
					key(t, model, keyType)
				}

				lines := model.Lines(width)
				if len(lines) == 0 {
					t.Fatal("the overlay stopped rendering; this assertion would be vacuous")
				}
				frame := strings.Join(lines, "\n")
				if strings.ContainsRune(frame, 0x1b) {
					t.Fatalf("the overlay contains a raw escape byte:\n%q", frame)
				}
				if strings.ContainsAny(frame, "\r\a") {
					t.Fatalf("the overlay contains a raw control byte:\n%q", frame)
				}
				if lines := strings.Count(frame, "\n") + 1; lines != len(model.Lines(width)) {
					t.Fatalf("a command title added a line to the overlay:\n%q", frame)
				}
				// The text stays readable; only its power over the terminal is
				// removed.
				if width >= 70 && state.name != "no match" && !strings.Contains(frame, "[31mrepaint") {
					t.Errorf("the sanitized title is unreadable:\n%s", frame)
				}
			})
		}
	}
}
