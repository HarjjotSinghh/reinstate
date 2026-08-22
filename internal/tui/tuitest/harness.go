// Package tuitest is the deterministic frame harness for Reinstate's
// interactive surfaces.
//
// It drives a Bubble Tea model by hand rather than running a program: messages
// go in, View strings come out, and every command the model returns is executed
// synchronously to a fixed depth. There is no terminal, no goroutine, no clock,
// and no ordering nondeterminism, so a frame can be committed as a golden file
// and compared byte for byte on macOS and Windows alike.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package tuitest

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// maxCommandDepth bounds the synchronous command drain. A model that schedules
// commands forever is a bug; failing the test beats hanging it.
const maxCommandDepth = 64

// Driver holds a model under test and applies messages to it.
type Driver struct {
	t     *testing.T
	model tea.Model
}

// New starts a driver, applying Init and a window size so the first frame is
// laid out exactly as a real terminal of that size would be.
func New(t *testing.T, model tea.Model, width, height int) *Driver {
	t.Helper()
	driver := &Driver{t: t, model: model}
	driver.drain(model.Init())
	driver.Send(tea.WindowSizeMsg{Width: width, Height: height})
	return driver
}

// Model returns the current model, for asserting on state rather than pixels.
func (d *Driver) Model() tea.Model { return d.model }

// Send applies one message and drains every command it produces.
func (d *Driver) Send(msg tea.Msg) *Driver {
	d.t.Helper()
	model, cmd := d.model.Update(msg)
	d.model = model
	d.drain(cmd)
	return d
}

// Type sends each rune of s as an individual key press, the way a person
// filtering a list actually produces input.
func (d *Driver) Type(s string) *Driver {
	d.t.Helper()
	for _, r := range s {
		d.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return d
}

// Key sends one named key. Names match the tea.KeyType spellings used in the
// surfaces themselves ("up", "down", "enter", "esc", "tab", "ctrl+k", " ").
func (d *Driver) Key(name string) *Driver {
	d.t.Helper()
	d.Send(keyMsg(d.t, name))
	return d
}

// Keys sends several named keys in order.
func (d *Driver) Keys(names ...string) *Driver {
	d.t.Helper()
	for _, name := range names {
		d.Key(name)
	}
	return d
}

// Resize re-lays out the model at a new size.
func (d *Driver) Resize(width, height int) *Driver {
	d.t.Helper()
	return d.Send(tea.WindowSizeMsg{Width: width, Height: height})
}

// View renders the current frame with trailing whitespace stripped per line, so
// a golden file never depends on invisible padding.
func (d *Driver) View() string {
	d.t.Helper()
	return Normalize(d.model.View())
}

// drain executes a command tree synchronously. Batches and sequences are walked
// so a model that fans out work still reaches a settled state before View.
func (d *Driver) drain(cmd tea.Cmd) {
	d.t.Helper()
	d.drainDepth(cmd, 0)
}

func (d *Driver) drainDepth(cmd tea.Cmd, depth int) {
	d.t.Helper()
	if cmd == nil {
		return
	}
	if depth >= maxCommandDepth {
		d.t.Fatalf("tuitest: command depth exceeded %d; the model schedules work without settling", maxCommandDepth)
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	switch typed := msg.(type) {
	case tea.QuitMsg:
		// Quitting is terminal state, not a message the model handles.
		return
	case tea.BatchMsg:
		for _, next := range typed {
			d.drainDepth(next, depth+1)
		}
		return
	default:
		model, next := d.model.Update(msg)
		d.model = model
		d.drainDepth(next, depth+1)
	}
}

// Normalize strips trailing whitespace from every line and trailing blank lines
// from the frame. Golden comparisons use it on both sides.
func Normalize(frame string) string {
	lines := strings.Split(strings.ReplaceAll(frame, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func keyMsg(t *testing.T, name string) tea.KeyMsg {
	t.Helper()
	switch name {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case " ", "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+a":
		return tea.KeyMsg{Type: tea.KeyCtrlA}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+k":
		return tea.KeyMsg{Type: tea.KeyCtrlK}
	case "ctrl+n":
		return tea.KeyMsg{Type: tea.KeyCtrlN}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	case "ctrl+r":
		return tea.KeyMsg{Type: tea.KeyCtrlR}
	default:
		runes := []rune(name)
		if len(runes) != 1 {
			t.Fatalf("tuitest: unknown key name %q", name)
			return tea.KeyMsg{}
		}
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
	}
}
