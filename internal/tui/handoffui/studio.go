// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package handoffui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/handoff"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Policies are the projection policies, ordered from least to most carried.
// The order is meaningful: left and right move along a spectrum.
var Policies = []string{
	string(handoff.PolicyCheckpoint),
	string(handoff.PolicyBalanced),
	string(handoff.PolicyFull),
}

// policyBlurb is the one-line explanation shown beside each policy. It says
// what the policy does, not how large it is; size is measured and shown live.
var policyBlurb = map[string]string{
	string(handoff.PolicyCheckpoint): "task boundary only, no verbatim conversation",
	string(handoff.PolicyBalanced):   "newest turns that fit the prompt budget",
	string(handoff.PolicyFull):       "every portable turn, up to the hard cap",
}

// Model is the handoff studio.
type Model struct {
	theme      ui.Theme
	capability ui.Capability
	planner    *Planner
	clipboard  tui.ClipboardFunc
	ctx        context.Context

	reference    string
	sourceAgent  string
	destinations []string

	destinationIndex int
	policyIndex      int

	width  int
	height int

	status string
	export bool
	// acknowledged records that the user has accepted the current selection's
	// warnings. It is cleared whenever the selection changes, because the
	// warnings belong to one destination-and-policy pair, not to the surface.
	acknowledged bool
	confirm      bool
	quitting     bool
	err          error
}

// Options configure a studio.
type Options struct {
	Theme      ui.Theme
	Capability ui.Capability
	Planner    *Planner
	Clipboard  tui.ClipboardFunc
	Context    context.Context
	// Reference is the canonical source agent:session-id.
	Reference string
	// SourceAgent is the agent the session belongs to, shown in the header.
	SourceAgent string
	// Destinations are the agent keys that can receive a handoff. The list is
	// supplied by the caller from the catalog rather than hardcoded here, so a
	// new destination tier never needs a change in the view layer.
	Destinations []string
	// Policy is the initially selected projection policy.
	Policy string
}

// New builds a studio.
func New(opts Options) *Model {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	policyIndex := indexOf(Policies, strings.ToLower(strings.TrimSpace(opts.Policy)))
	if policyIndex < 0 {
		policyIndex = indexOf(Policies, string(handoff.PolicyBalanced))
	}
	return &Model{
		theme:        opts.Theme,
		capability:   opts.Capability,
		planner:      opts.Planner,
		clipboard:    opts.Clipboard,
		ctx:          ctx,
		reference:    opts.Reference,
		sourceAgent:  opts.SourceAgent,
		destinations: append([]string(nil), opts.Destinations...),
		policyIndex:  policyIndex,
		width:        opts.Capability.Width,
		height:       opts.Capability.Height,
	}
}

func indexOf(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

// Destination returns the currently selected destination agent.
func (m *Model) Destination() string {
	if m.destinationIndex < 0 || m.destinationIndex >= len(m.destinations) {
		return ""
	}
	return m.destinations[m.destinationIndex]
}

// Policy returns the currently selected projection policy.
func (m *Model) Policy() string { return Policies[m.policyIndex] }

// ExportRequested reports whether the user asked to export rather than launch.
func (m *Model) ExportRequested() bool { return m.export }

// Warnings returns the warning IDs for the current selection.
func (m *Model) Warnings() []string {
	preview, ready := m.current()
	if !ready || preview.Err != nil {
		return nil
	}
	return preview.Warnings
}

// Acknowledged reports whether the user accepted the current warnings.
func (m *Model) Acknowledged() bool { return m.acknowledged }

// Intent implements tui.Surface.
//
// The acknowledged warning IDs travel with the intent. Without them the caller
// re-enters `rein handoff` with none, and the pipeline requires every current
// warning to be acknowledged — so a studio launch would refuse every time.
func (m *Model) Intent() tui.Intent {
	if !m.confirm {
		return tui.Intent{}
	}
	intent := tui.Intent{
		Action:      tui.ActionHandoff,
		Reference:   m.reference,
		Destination: m.Destination(),
		Policy:      m.Policy(),
	}
	if m.acknowledged {
		intent.AcknowledgedWarnings = append([]string(nil), m.Warnings()...)
	}
	return intent
}

// Err implements tui.Surface.
func (m *Model) Err() error { return m.err }

// Init implements tea.Model. It computes the preview for the opening selection.
func (m *Model) Init() tea.Cmd { return m.computeCurrent() }

func (m *Model) computeCurrent() tea.Cmd {
	if m.planner == nil || m.Destination() == "" {
		return nil
	}
	return m.planner.Compute(m.ctx, m.Destination(), m.Policy())
}

// current returns the preview for the current selection, and whether it is
// ready. A pending preview renders as "measuring" rather than as zeroes, since
// zeroes would read as "this handoff carries nothing".
func (m *Model) current() (Preview, bool) {
	if m.planner == nil {
		return Preview{}, false
	}
	return m.planner.Lookup(m.Destination(), m.Policy())
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = typed.Width, typed.Height
		return m, nil

	case PreviewedMsg:
		return m, nil

	case tui.ClipboardMsg:
		m.status = typed.String()
		return m, nil

	case tea.KeyMsg:
		return m.updateKey(typed)
	}
	return m, nil
}

func (m *Model) updateKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.confirm = false
		m.quitting = true
		return m, tea.Quit

	case tea.KeyLeft:
		m.cyclePolicy(-1)
		return m, m.computeCurrent()

	case tea.KeyRight:
		m.cyclePolicy(1)
		return m, m.computeCurrent()

	case tea.KeyUp:
		m.cycleDestination(-1)
		return m, m.computeCurrent()

	case tea.KeyDown, tea.KeyTab:
		m.cycleDestination(1)
		return m, m.computeCurrent()

	case tea.KeyEnter:
		return m.send()

	case tea.KeyRunes:
		if len(key.Runes) != 1 {
			return m, nil
		}
		switch key.Runes[0] {
		case 'a':
			if len(m.Warnings()) > 0 {
				m.acknowledged = !m.acknowledged
				m.status = ""
			}
			return m, nil
		case 'e':
			m.export = true
			return m.send()
		case 'c', 'y':
			return m, m.copyCommand()
		case 'q':
			m.confirm = false
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// send refuses to proceed on a preview that failed to build. Launching a
// destination from a plan that could not be produced would put the user in a
// new session with no briefing and no explanation.
func (m *Model) send() (tea.Model, tea.Cmd) {
	if m.Destination() == "" {
		m.status = "no destination agent is available for this source"
		m.export = false
		return m, nil
	}
	if preview, ready := m.current(); ready && preview.Err != nil {
		m.status = "this handoff cannot be planned: " + ui.Sanitize(preview.Err.Error())
		m.export = false
		return m, nil
	}
	// Warnings must be accepted before the handoff runs, exactly as the flag
	// path requires one --allow-warning per current warning. Sending without
	// them would be refused downstream with a message about identifiers the
	// user never typed.
	if warnings := m.Warnings(); len(warnings) > 0 && !m.acknowledged {
		m.status = "press a to acknowledge " + plural(len(warnings), "warning", "warnings") + " before sending"
		m.export = false
		return m, nil
	}
	m.confirm = true
	m.quitting = true
	return m, tea.Quit
}

func (m *Model) cyclePolicy(delta int) {
	m.policyIndex = wrap(m.policyIndex+delta, len(Policies))
	m.selectionChanged()
}

func (m *Model) cycleDestination(delta int) {
	if len(m.destinations) == 0 {
		return
	}
	m.destinationIndex = wrap(m.destinationIndex+delta, len(m.destinations))
	m.selectionChanged()
}

// selectionChanged drops an acknowledgement that no longer applies. A different
// policy or destination produces a different warning set, so carrying the
// acceptance across would acknowledge warnings the user never saw.
func (m *Model) selectionChanged() {
	m.acknowledged = false
	m.status = ""
}

func wrap(index, length int) int {
	if length <= 0 {
		return 0
	}
	return ((index % length) + length) % length
}

func (m *Model) copyCommand() tea.Cmd {
	command := m.EquivalentCommand()
	if m.clipboard == nil {
		m.status = command
		return nil
	}
	copyFunc := m.clipboard
	return func() tea.Msg {
		return tui.ClipboardMsg{Text: command, Err: copyFunc(command)}
	}
}

// EquivalentCommand renders the exact non-interactive command for the current
// selection, so the studio teaches the flag form rather than replacing it.
func (m *Model) EquivalentCommand() string {
	var builder strings.Builder
	builder.WriteString("rein handoff ")
	builder.WriteString(m.reference)
	builder.WriteString(" --to ")
	builder.WriteString(m.Destination())
	builder.WriteString(" --policy ")
	builder.WriteString(m.Policy())
	if m.export {
		builder.WriteString(" --no-launch")
	}
	if m.acknowledged {
		for _, warning := range m.Warnings() {
			builder.WriteString(" --allow-warning ")
			builder.WriteString(warning)
		}
	}
	return builder.String()
}
