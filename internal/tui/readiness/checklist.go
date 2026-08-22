// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package readiness

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/preflight"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Checklist is the interactive acknowledgement of environment warnings.
//
// It exists to delete a specific piece of drudgery. The flag path requires
// reading a check identifier off the screen and typing it back exactly, once
// per warning, with a typo silently meaning "not acknowledged". Here each
// warning is a line and the spacebar is the whole interaction.
//
// It grants nothing the flags could not: the identifiers it collects go to the
// same preflight.Authorize call, and the equivalent command is printed on
// screen so the reader can see, and copy, exactly what they just authorized.
type Checklist struct {
	theme      ui.Theme
	capability ui.Capability
	report     preflight.Report
	reference  string
	operation  string
	clipboard  tui.ClipboardFunc

	items  []item
	cursor int
	status string

	width  int
	height int

	confirmed bool
	quitting  bool
	err       error
}

type item struct {
	check    preflight.Check
	accepted bool
}

// Options configure a checklist.
type Options struct {
	Theme      ui.Theme
	Capability ui.Capability
	// Report is the environment report whose warnings need acknowledgement.
	Report preflight.Report
	// Reference is the canonical agent:session-id being launched.
	Reference string
	// Operation is the verb shown in the equivalent command: resume or fork.
	Operation string
	// Clipboard copies the equivalent command. nil disables the copy key.
	Clipboard tui.ClipboardFunc
}

// NewChecklist builds a checklist from a report.
//
// Only warnings appear. Blocking checks are not acknowledgeable by anyone, and
// offering a checkbox that cannot be honoured would be dishonest; informational
// checks need no decision.
func NewChecklist(opts Options) *Checklist {
	warnings := make([]preflight.Check, 0, len(opts.Report.Checks))
	for _, check := range opts.Report.Checks {
		if check.Severity == preflight.SeverityWarning {
			warnings = append(warnings, check)
		}
	}
	sort.SliceStable(warnings, func(i, j int) bool { return warnings[i].ID < warnings[j].ID })

	items := make([]item, 0, len(warnings))
	for _, check := range warnings {
		// Nothing starts acknowledged. Acknowledging a warning is a decision,
		// and a pre-ticked box would let a distracted reader make it by
		// pressing enter without ever reading the line.
		items = append(items, item{check: check})
	}
	operation := strings.TrimSpace(opts.Operation)
	if operation == "" {
		operation = "resume"
	}
	return &Checklist{
		theme:      opts.Theme,
		capability: opts.Capability,
		report:     opts.Report,
		reference:  opts.Reference,
		operation:  operation,
		clipboard:  opts.Clipboard,
		items:      items,
		width:      opts.Capability.Width,
		height:     opts.Capability.Height,
	}
}

// Intent implements tui.Surface. The checklist decides acknowledgement, not
// which action to take, so it echoes the operation it was given.
func (c *Checklist) Intent() tui.Intent {
	if !c.confirmed {
		return tui.Intent{}
	}
	action := tui.ActionResume
	if c.operation == "fork" {
		action = tui.ActionFork
	}
	return tui.Intent{
		Action:               action,
		Reference:            c.reference,
		AcknowledgedWarnings: c.Acknowledged(),
	}
}

// Err implements tui.Surface.
func (c *Checklist) Err() error { return c.err }

// Acknowledged returns the exact identifiers the user ticked.
func (c *Checklist) Acknowledged() []string {
	ids := make([]string, 0, len(c.items))
	for _, current := range c.items {
		if current.accepted {
			ids = append(ids, current.check.ID)
		}
	}
	return ids
}

// Confirmed reports whether the user accepted rather than cancelled.
func (c *Checklist) Confirmed() bool { return c.confirmed }

// AllAccepted reports whether every warning has been ticked. Launch requires
// this, because preflight authorizes only a complete acknowledgement.
func (c *Checklist) AllAccepted() bool {
	for _, current := range c.items {
		if !current.accepted {
			return false
		}
	}
	return true
}

// Init implements tea.Model.
func (c *Checklist) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (c *Checklist) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		c.width, c.height = typed.Width, typed.Height
		return c, nil

	case tui.ClipboardMsg:
		c.status = typed.String()
		return c, nil

	case tea.KeyMsg:
		switch typed.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			c.confirmed = false
			c.quitting = true
			return c, tea.Quit

		case tea.KeyUp:
			c.move(-1)
			return c, nil

		case tea.KeyDown:
			c.move(1)
			return c, nil

		case tea.KeySpace:
			c.toggle()
			return c, nil

		case tea.KeyEnter:
			// Enter only proceeds once everything is ticked. Refusing here,
			// with an explanation, beats launching and having preflight refuse
			// afterwards with a message about identifiers the user never typed.
			if !c.AllAccepted() {
				c.status = "every warning must be acknowledged before continuing"
				return c, nil
			}
			c.confirmed = true
			c.quitting = true
			return c, tea.Quit

		case tea.KeyRunes:
			if len(typed.Runes) != 1 {
				return c, nil
			}
			switch typed.Runes[0] {
			case 'a':
				accept := !c.AllAccepted()
				for index := range c.items {
					c.items[index].accepted = accept
				}
				return c, nil
			case 'c', 'y':
				return c, c.copyCommand()
			case 'q':
				c.confirmed = false
				c.quitting = true
				return c, tea.Quit
			}
		}
	}
	return c, nil
}

func (c *Checklist) move(delta int) {
	if len(c.items) == 0 {
		return
	}
	c.cursor += delta
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor >= len(c.items) {
		c.cursor = len(c.items) - 1
	}
}

func (c *Checklist) toggle() {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return
	}
	c.items[c.cursor].accepted = !c.items[c.cursor].accepted
	c.status = ""
}

func (c *Checklist) copyCommand() tea.Cmd {
	command := c.EquivalentCommand()
	if c.clipboard == nil {
		c.status = command
		return nil
	}
	copyFunc := c.clipboard
	return func() tea.Msg {
		return tui.ClipboardMsg{Text: command, Err: copyFunc(command)}
	}
}

// EquivalentCommand renders the exact non-interactive command that produces
// what is currently ticked.
//
// This is the contract that keeps the interactive surface honest: whatever can
// be done here can be done, and scripted, from a single command line. It is
// shown on screen at all times, not hidden behind a key.
func (c *Checklist) EquivalentCommand() string {
	var builder strings.Builder
	builder.WriteString("rein ")
	builder.WriteString(c.operation)
	builder.WriteString(" ")
	builder.WriteString(c.reference)
	for _, id := range c.Acknowledged() {
		builder.WriteString(" --allow-environment-warning ")
		builder.WriteString(id)
	}
	return builder.String()
}
