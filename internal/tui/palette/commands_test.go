// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

// This file is an external test package on purpose. The palette is imported by
// the switcher, so an in-package test cannot import the switcher back without
// creating an import cycle; palette_test can. Everything here is about the
// palette meeting the real command table rather than a fixture, so it belongs
// on this side of that boundary.
package palette_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/tui/palette"
	"github.com/HarjjotSinghh/reinstate/internal/tui/switcher"
	"github.com/HarjjotSinghh/reinstate/internal/tui/tuitest"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// theme is the monochrome Unicode theme the goldens are rendered with. NO_COLOR
// is pinned by the TestMain in palette_test.go, which covers this package too:
// both files compile into the same test binary.
func theme() ui.Theme {
	return ui.NewTheme(ui.Capability{Mode: ui.ModeFull, Color: ui.ColorNone, Unicode: true, Width: 110, Height: 30})
}

// query builds a palette over the real command table and types s into it.
func query(t *testing.T, s string, width, height int) *palette.Model {
	t.Helper()
	model := palette.New(theme(), switcher.Commands, width, height)
	for _, r := range s {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		if r == ' ' {
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		}
		if !model.Update(msg) {
			t.Fatalf("key %q leaked out of the open overlay", string(r))
		}
	}
	return model
}

// TestCommandTableIsWellFormed checks the data the palette is built from. Every
// field here reaches the screen or the dispatcher, so a blank one is a row the
// reader cannot read or an action nothing can run.
func TestCommandTableIsWellFormed(t *testing.T) {
	if len(switcher.Commands) == 0 {
		t.Fatal("the command table is empty; every test below would be vacuous")
	}
	seen := make(map[string]int, len(switcher.Commands))
	for index, command := range switcher.Commands {
		if strings.TrimSpace(command.ID) == "" {
			t.Errorf("entry %d has no ID: %+v", index, command)
		}
		if strings.TrimSpace(command.Title) == "" {
			t.Errorf("entry %d (%q) has no Title", index, command.ID)
		}
		if strings.TrimSpace(command.Detail) == "" {
			t.Errorf("entry %d (%q) has no Detail; the reader would see a bare verb", index, command.ID)
		}
		if previous, duplicate := seen[command.ID]; duplicate {
			// A duplicate ID makes one of the two rows unreachable: they rank
			// together and dispatch to the same place.
			t.Errorf("entry %d repeats the ID %q first used by entry %d", index, command.ID, previous)
		}
		seen[command.ID] = index
	}
	for _, command := range switcher.Commands {
		for _, keyword := range command.Keys {
			if strings.TrimSpace(keyword) == "" {
				t.Errorf("command %q has a blank alternative keyword", command.ID)
			}
			if keyword != strings.ToLower(keyword) {
				// refilter lowercases the candidate before matching, so an
				// uppercase keyword is not wrong, merely a claim worth pinning.
				t.Errorf("command %q has the non-lowercase keyword %q", command.ID, keyword)
			}
		}
	}
}

// TestEveryCommandIsFoundByItsOwnTitle is the discoverability floor. A command
// the reader cannot reach by typing its name is a command that exists only for
// whoever already memorised the table.
func TestEveryCommandIsFoundByItsOwnTitle(t *testing.T) {
	for _, command := range switcher.Commands {
		t.Run(command.ID, func(t *testing.T) {
			model := query(t, strings.ToLower(command.Title), 110, 30)
			if !model.Update(tea.KeyMsg{Type: tea.KeyEnter}) {
				t.Fatal("enter leaked out of the open overlay")
			}
			if model.Open() {
				t.Fatal("enter must close the overlay")
			}
			if got := model.Chosen(); got != command.ID {
				t.Fatalf("typing %q and pressing enter ran %q, want %q",
					strings.ToLower(command.Title), got, command.ID)
			}
		})
	}

	t.Run("and by its own identifier", func(t *testing.T) {
		for _, command := range switcher.Commands {
			model := query(t, command.ID, 110, 30)
			if !model.Update(tea.KeyMsg{Type: tea.KeyEnter}) {
				t.Fatal("enter leaked out of the open overlay")
			}
			if got := model.Chosen(); got != command.ID {
				t.Errorf("typing the identifier %q ran %q", command.ID, got)
			}
		}
	})
}

// TestGoldenFrames pins what a reader actually sees when the overlay is up.
// Regenerate with `go test ./internal/tui/palette/ -update-golden` and review
// the diff: an unexplained change to one of these is a regression in the view.
func TestGoldenFrames(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		width  int
		height int
		rows   int
		golden string
	}{
		{
			name: "unfiltered", width: 110, height: 30,
			rows: 8, golden: "palette_unfiltered_110x30",
		},
		{
			name: "filtered to one", query: "diagnostics", width: 110, height: 30,
			rows: 1, golden: "palette_filtered_110x30",
		},
		{
			name: "no match", query: "deploy", width: 110, height: 30,
			rows: 0, golden: "palette_no_match_110x30",
		},
		{
			name: "narrow", width: 60, height: 20,
			rows: 8, golden: "palette_narrow_60x20",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := query(t, test.query, test.width, test.height)
			lines := model.Lines(test.width)
			if len(lines) == 0 {
				t.Fatal("the overlay rendered nothing")
			}
			frame := strings.Join(lines, "\n")

			// A golden frame is compared byte for byte, so it must not depend
			// on the colour profile of whatever terminal ran the suite.
			if strings.ContainsRune(frame, 0x1b) {
				t.Fatal("a golden frame must contain no escape sequences")
			}
			for index, line := range lines {
				if got := ui.Width(line); got > test.width {
					t.Errorf("line %d is %d cells wide, the overlay is %d\n%q",
						index+1, got, test.width, line)
				}
			}
			if got := countRows(lines); got != test.rows {
				t.Fatalf("the overlay drew %d command rows, want %d:\n%s", got, test.rows, frame)
			}
			tuitest.AssertGolden(t, test.golden, frame)
		})
	}
}

// countRows counts the command rows in a rendered overlay: the body, less the
// advisory line the view draws when there is nothing to show or more to see.
func countRows(lines []string) int {
	count := 0
	for _, line := range lines[3 : len(lines)-1] {
		if strings.Contains(line, "no command matches") || strings.Contains(line, "more match") {
			continue
		}
		count++
	}
	return count
}
