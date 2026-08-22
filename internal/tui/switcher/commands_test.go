// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package switcher

import (
	"sort"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/tui"
)

// The two halves of runCommand's switch, transcribed from the source.
//
// handledIDs are the identifiers runCommand names explicitly and turns into an
// intent or a state change here. passthroughIDs fall to the default branch and
// become ActionCommand, which the caller runs as a `rein` subcommand.
//
// They are written out rather than derived because a Go switch cannot be
// enumerated at run time. TestRunCommandCoversEveryPaletteCommand then checks
// them against the table from both ends, and every entry is exercised for the
// behaviour claimed by the half it was filed under, so a command that gains a
// branch, loses one, or is added to the table without one fails here.
var (
	handledIDs     = []string{"resume", "fork", "handoff", "inspect", "copy", "scope", "refresh", "quit"}
	passthroughIDs = []string{"doctor", "status", "push", "pull"}
)

// TestRunCommandCoversEveryPaletteCommand is the seam between the palette and
// the switcher. The palette can only ever return an identifier from Commands,
// so runCommand's switch is exhaustive over a closed set — but only for as long
// as the two are kept in step. A table entry with no branch silently becomes a
// subcommand that does not exist; a branch with no table entry is dead code.
func TestRunCommandCoversEveryPaletteCommand(t *testing.T) {
	t.Run("the table and the switch agree", func(t *testing.T) {
		covered := make(map[string]string, len(handledIDs)+len(passthroughIDs))
		for _, id := range handledIDs {
			covered[id] = "handled"
		}
		for _, id := range passthroughIDs {
			if previous, duplicate := covered[id]; duplicate {
				t.Fatalf("%q is listed as both %s and passthrough", id, previous)
			}
			covered[id] = "passthrough"
		}

		inTable := make(map[string]bool, len(Commands))
		for _, command := range Commands {
			inTable[command.ID] = true
			if covered[command.ID] == "" {
				t.Errorf("the palette offers %q but runCommand has no branch for it, "+
					"so choosing it would run `rein %s`", command.ID, command.ID)
			}
		}
		for id, kind := range covered {
			if !inTable[id] {
				t.Errorf("runCommand has a %s branch for %q, which no palette entry can produce", kind, id)
			}
		}

		if len(covered) != len(Commands) {
			t.Fatalf("runCommand covers %d identifiers, the table has %d: %s vs %s",
				len(covered), len(Commands), sortedKeys(covered), commandIDs())
		}
	})

	t.Run("an empty identifier decides nothing", func(t *testing.T) {
		_, model, loader := start(t, config{project: projectReinstate})
		calls := loader.calls()

		model.runCommand("")

		if model.Intent().Chosen() || model.quitting {
			t.Fatalf("a cancelled palette resolved the surface: %+v", model.Intent())
		}
		if loader.calls() != calls {
			t.Error("a cancelled palette triggered a reload")
		}
	})

	// Each handled identifier must do the thing it was filed under, so the list
	// above cannot drift into a set of names nobody checks.
	handled := []struct {
		id     string
		assert func(t *testing.T, model *Model, loader *fakeLoader)
	}{
		{id: "resume", assert: assertChooses(tui.ActionResume)},
		{id: "fork", assert: assertChooses(tui.ActionFork)},
		{id: "handoff", assert: assertChooses(tui.ActionHandoff)},
		{id: "inspect", assert: assertChooses(tui.ActionInspect)},
		{
			id: "copy",
			assert: func(t *testing.T, model *Model, _ *fakeLoader) {
				t.Helper()
				// No clipboard is configured, so the reference is surfaced for
				// the reader to select by hand instead.
				if want := mustSelected(t, model).Reference(); model.status != want {
					t.Errorf("status = %q, want the session reference %q", model.status, want)
				}
			},
		},
		{
			id: "scope",
			assert: func(t *testing.T, model *Model, loader *fakeLoader) {
				t.Helper()
				if model.scope != ScopeAll {
					t.Errorf("scope = %v, want it toggled to every project", model.scope)
				}
				if filter := loader.last(t); filter.Project != "" {
					t.Errorf("the reload still scopes to project %q", filter.Project)
				}
			},
		},
		{
			id: "refresh",
			assert: func(t *testing.T, _ *Model, loader *fakeLoader) {
				t.Helper()
				if loader.calls() < 2 {
					t.Errorf("the index was loaded %d times, want a rescan on top of the first load",
						loader.calls())
				}
			},
		},
		{
			id: "quit",
			assert: func(t *testing.T, model *Model, _ *fakeLoader) {
				t.Helper()
				if !model.quitting {
					t.Error("quit must end the surface")
				}
				assertZeroIntent(t, model.Intent())
			},
		},
	}
	for _, test := range handled {
		t.Run("handled/"+test.id, func(t *testing.T) {
			driver, model, loader := start(t, config{project: projectReinstate})
			_, cmd := model.runCommand(test.id)
			if cmd != nil {
				driver.Send(cmd())
			}

			if model.Intent().Action == tui.ActionCommand {
				t.Fatalf("%q fell through to the default branch and became `rein %s`",
					test.id, model.Intent().Command)
			}
			if model.Intent().Command != "" {
				t.Fatalf("%q set Intent.Command = %q without being a subcommand",
					test.id, model.Intent().Command)
			}
			test.assert(t, model, loader)
		})
	}

	// Every passthrough identifier must reach the default branch unchanged: the
	// switcher quits, and the caller runs `rein <id>` in its place.
	for _, id := range passthroughIDs {
		t.Run("passthrough/"+id, func(t *testing.T) {
			_, model, _ := start(t, config{project: projectReinstate})

			model.runCommand(id)

			intent := model.Intent()
			if intent.Action != tui.ActionCommand {
				t.Fatalf("action = %q, want %q so the caller runs the subcommand",
					intent.Action, tui.ActionCommand)
			}
			if intent.Command != id {
				t.Fatalf("command = %q, want %q", intent.Command, id)
			}
			if intent.Reference != "" {
				t.Errorf("a global command carried the session reference %q", intent.Reference)
			}
			if !model.quitting {
				t.Error("a subcommand must quit the switcher; it cannot run underneath it")
			}
		})
	}
}

// TestPaletteChoicesReachRunCommand closes the loop: the identifiers checked
// above are the ones the overlay actually produces, typed rather than injected.
func TestPaletteChoicesReachRunCommand(t *testing.T) {
	tests := []struct {
		name   string
		typed  string
		assert func(t *testing.T, model *Model)
	}{
		{
			name:  "a session action",
			typed: "fork session",
			assert: func(t *testing.T, model *Model) {
				t.Helper()
				if model.Intent().Action != tui.ActionFork {
					t.Fatalf("intent = %+v, want a fork", model.Intent())
				}
			},
		},
		{
			name:  "a subcommand",
			typed: "sync status",
			assert: func(t *testing.T, model *Model) {
				t.Helper()
				if got := model.Intent(); got.Action != tui.ActionCommand || got.Command != "status" {
					t.Fatalf("intent = %+v, want `rein status`", got)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			driver, model, _ := start(t, config{project: projectReinstate})
			driver.Key("ctrl+k")
			if model.mode != modePalette {
				t.Fatal("ctrl+k did not open the palette")
			}
			driver.Type(test.typed)
			driver.Key("enter")

			if model.mode != modeList {
				t.Fatalf("mode = %v after choosing, want the list back", model.mode)
			}
			if model.palette != nil {
				t.Error("the overlay outlived the choice")
			}
			test.assert(t, model)
		})
	}

	t.Run("esc runs nothing", func(t *testing.T) {
		driver, model, loader := start(t, config{project: projectReinstate})
		driver.Key("ctrl+k")
		driver.Type("quit")
		calls := loader.calls()
		driver.Key("esc")

		if model.mode != modeList || model.palette != nil {
			t.Fatalf("esc left the overlay behind: mode %v, palette %v", model.mode, model.palette)
		}
		if model.Intent().Chosen() || model.quitting {
			t.Fatalf("a dismissed palette resolved the surface: %+v", model.Intent())
		}
		if loader.calls() != calls {
			t.Error("a dismissed palette reloaded the index")
		}
		// Typing into the overlay must not have leaked into the list filter.
		if model.filter != "" {
			t.Errorf("filter = %q; the overlay let keys through to the list", model.filter)
		}
	})
}

// assertChooses builds the assertion for an identifier that resolves to an
// action on the selected session.
func assertChooses(action tui.Action) func(t *testing.T, model *Model, loader *fakeLoader) {
	return func(t *testing.T, model *Model, _ *fakeLoader) {
		t.Helper()
		intent := model.Intent()
		if intent.Action != action {
			t.Fatalf("action = %q, want %q", intent.Action, action)
		}
		if intent.Reference == "" {
			t.Fatal("an action on a session carries no session reference")
		}
		if !model.quitting {
			t.Error("choosing an action must quit the surface")
		}
	}
}

func sortedKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

func commandIDs() string {
	ids := make([]string, 0, len(Commands))
	for _, command := range Commands {
		ids = append(ids, command.ID)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}
