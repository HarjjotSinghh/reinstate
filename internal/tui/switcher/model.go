// Package switcher is the interactive session list behind bare `rein`.
//
// It replaces the numbered re-print loop: rows are navigated with the arrow
// keys, typing filters in place, and the selected session is previewed beside
// the list. Like every surface in internal/tui it returns an Intent rather than
// acting, so the terminal is fully restored before a vendor process inherits it.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package switcher

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/tui"
	"github.com/HarjjotSinghh/reinstate/internal/tui/palette"
	"github.com/HarjjotSinghh/reinstate/internal/tui/readiness"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Scope is which sessions the list is showing.
type Scope int

const (
	// ScopeProject shows only the project the user is standing in. This is the
	// default inside a known repository, because it is almost always what the
	// reader means, and it removes the need to filter at all.
	ScopeProject Scope = iota
	// ScopeAll shows every indexed session.
	ScopeAll
)

// mode is the surface's input mode. Keeping this explicit is what lets typing
// filter without stealing the action keys.
type mode int

const (
	// modeList is the default. Printable characters extend the filter, so the
	// fastest path to any session is simply to type part of its name.
	modeList mode = iota
	// modeActions is the Tab menu. Printable characters are accelerators here,
	// because the filter is not accepting input.
	modeActions
	// modePalette is the ctrl+k overlay, which owns the keyboard while open.
	modePalette
)

// Commands are the palette entries the switcher offers.
//
// Session actions appear alongside global ones so there is a single place to
// look for "what can I do", rather than one list on the key bar, another in the
// Tab menu, and the rest only in the manual.
var Commands = []palette.Command{
	{ID: "resume", Title: "Resume session", Detail: "continue in its own agent", Keys: []string{"continue", "open"}, NeedsSession: true},
	{ID: "fork", Title: "Fork session", Detail: "branch through the vendor's native fork", Keys: []string{"branch", "copy"}, NeedsSession: true},
	{ID: "handoff", Title: "Hand off to another agent", Detail: "new session from a briefing", Keys: []string{"transfer", "move", "switch"}, NeedsSession: true},
	{ID: "inspect", Title: "Inspect session", Detail: "full metadata and environment report", Keys: []string{"details", "info"}, NeedsSession: true},
	{ID: "copy", Title: "Copy session reference", Detail: "put agent:id on the clipboard", Keys: []string{"yank", "clipboard"}, NeedsSession: true},
	{ID: "scope", Title: "Toggle project scope", Detail: "this project or every project", Keys: []string{"all", "filter"}},
	{ID: "refresh", Title: "Refresh the index", Detail: "rescan every agent now", Keys: []string{"rescan", "reload"}},
	{ID: "doctor", Title: "Run diagnostics", Detail: "rein doctor", Keys: []string{"health", "check"}},
	{ID: "status", Title: "Sync status", Detail: "rein status", Keys: []string{"remote", "compare"}},
	{ID: "push", Title: "Push sessions", Detail: "rein push", Keys: []string{"upload", "sync"}},
	{ID: "pull", Title: "Pull sessions", Detail: "rein pull", Keys: []string{"download", "sync", "restore"}},
	{ID: "quit", Title: "Quit", Detail: "close the switcher", Keys: []string{"exit", "close"}},
}

// Loader supplies session records. It is an interface so the surface can be
// driven from fixtures with no index, no filesystem, and no vendor.
type Loader interface {
	// Load returns records matching filter, newest first.
	Load(filter sessionindex.Filter) ([]sessionindex.Record, error)
}

// ReadinessProvider computes and caches how resumable each record is.
//
// It is an interface rather than a plain function because the answer is not
// instant: a preflight report runs real workspace and vendor checks. Lookup
// serves whatever is already known without blocking a redraw, and Probe starts
// the work for rows that reach the screen.
//
// A nil provider, or one reporting Enabled false, hides the status column
// entirely rather than showing placeholders that never resolve.
type ReadinessProvider interface {
	Enabled() bool
	Lookup(record sessionindex.Record) ui.Readiness
	Probe(ctx context.Context, records []sessionindex.Record) tea.Cmd
}

// Options configure a switcher.
type Options struct {
	Theme      ui.Theme
	Capability ui.Capability
	Loader     Loader
	// Readiness is optional. When nil the status column is omitted.
	Readiness ReadinessProvider
	// Context bounds background readiness probes.
	Context context.Context
	// Project is the path used to scope the list to the repository the user is
	// standing in. Empty means they are not in a known project, so the scope
	// starts at all. It is matched against the index, not displayed.
	Project string
	// ProjectLabel is what the header shows for that scope. Empty falls back to
	// the base name of Project, because an absolute path is both too long for
	// the header and more of the user's filesystem than needs to be on screen.
	ProjectLabel string
	// Now is the reference time for relative ages and section bucketing.
	// The zero value selects time.Now once, at construction.
	Now time.Time
	// Limit bounds how many records are loaded.
	Limit int
	// Clipboard copies a session reference. nil disables the copy action.
	Clipboard tui.ClipboardFunc
}

// Model is the switcher's Bubble Tea model.
type Model struct {
	theme      ui.Theme
	capability ui.Capability
	loader     Loader
	readiness  ReadinessProvider
	clipboard  tui.ClipboardFunc
	ctx        context.Context
	now        time.Time
	limit      int

	width  int
	height int

	project      string
	projectLabel string
	scope        Scope

	// records is everything loaded for the current scope, newest first.
	records []sessionindex.Record
	// rows is the rendered list: section headers interleaved with sessions.
	rows []row
	// cursor indexes rows and always lands on a session, never a header.
	cursor int
	// offset is the first visible row, maintained so the cursor stays on screen.
	offset int

	filter  string
	mode    mode
	status  string
	loadErr error
	palette *palette.Model

	intent   tui.Intent
	err      error
	quitting bool
}

// row is one rendered line: either a section heading or a session.
type row struct {
	header string
	record sessionindex.Record
}

func (r row) isHeader() bool { return r.header != "" }

// New builds a switcher. It does not load; Init does, so construction stays
// pure and testable.
func New(opts Options) *Model {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	scope := ScopeProject
	if strings.TrimSpace(opts.Project) == "" {
		scope = ScopeAll
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = sessionindex.DefaultLimit
	}
	probeContext := opts.Context
	if probeContext == nil {
		probeContext = context.Background()
	}
	// A provider that reports itself disabled is dropped here rather than
	// checked on every row render, so the status column simply does not exist.
	provider := opts.Readiness
	if provider != nil && !provider.Enabled() {
		provider = nil
	}
	return &Model{
		ctx:          probeContext,
		theme:        opts.Theme,
		capability:   opts.Capability,
		loader:       opts.Loader,
		readiness:    provider,
		clipboard:    opts.Clipboard,
		now:          now,
		limit:        limit,
		project:      strings.TrimSpace(opts.Project),
		projectLabel: projectLabel(opts),
		scope:        scope,
		width:        opts.Capability.Width,
		height:       opts.Capability.Height,
	}
}

// projectLabel resolves the header text for the project scope.
func projectLabel(opts Options) string {
	if label := strings.TrimSpace(opts.ProjectLabel); label != "" {
		return label
	}
	project := strings.TrimSpace(opts.Project)
	if project == "" {
		return ""
	}
	return filepath.Base(project)
}

// Intent implements tui.Surface.
func (m *Model) Intent() tui.Intent { return m.intent }

// Err implements tui.Surface.
func (m *Model) Err() error { return m.err }

// Init loads the first page.
func (m *Model) Init() tea.Cmd { return m.loadCmd() }

// loadedMsg carries a completed load back into the update loop.
type loadedMsg struct {
	records []sessionindex.Record
	err     error
}

func (m *Model) loadCmd() tea.Cmd {
	filter := m.currentFilter()
	loader := m.loader
	return func() tea.Msg {
		if loader == nil {
			return loadedMsg{}
		}
		records, err := loader.Load(filter)
		return loadedMsg{records: records, err: err}
	}
}

// currentFilter translates the surface's state into an index filter. Query and
// project go to the index rather than being applied in the view, so the
// switcher and `rein search` match identically by construction.
func (m *Model) currentFilter() sessionindex.Filter {
	filter := sessionindex.Filter{
		Query: strings.TrimSpace(m.filter),
		Limit: m.limit,
	}
	if m.scope == ScopeProject && m.project != "" {
		filter.Project = m.project
	}
	return filter
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		if m.palette != nil {
			m.palette.Resize(m.width, m.height)
		}
		m.clampOffset()
		return m, nil

	case loadedMsg:
		m.loadErr = typed.err
		m.records = typed.records
		m.rebuildRows()
		return m, m.probeVisible()

	case readiness.ProbedMsg:
		// The cache is the state; this message only says it is worth reading
		// again, so a redraw is the entire handler.
		return m, nil

	case tui.ClipboardMsg:
		m.status = typed.String()
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modePalette:
			return m.updatePalette(typed)
		case modeActions:
			return m.updateActions(typed)
		default:
			return m.updateList(typed)
		}
	}
	return m, nil
}

// updateList handles the default mode, where printable input filters.
func (m *Model) updateList(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
	case tea.KeyCtrlC:
		return m.cancel()

	case tea.KeyEsc:
		// Escape backs out one level: it clears a filter first, and only quits
		// when there is nothing left to clear. Quitting on a stray escape while
		// a filter is typed would lose the user's work.
		if m.filter != "" {
			m.filter = ""
			return m, m.loadCmd()
		}
		return m.cancel()

	case tea.KeyUp, tea.KeyCtrlP:
		m.moveCursor(-1)
		return m, m.probeVisible()

	case tea.KeyDown, tea.KeyCtrlN:
		m.moveCursor(1)
		return m, m.probeVisible()

	case tea.KeyPgUp:
		m.moveCursor(-m.pageSize())
		return m, m.probeVisible()

	case tea.KeyPgDown:
		m.moveCursor(m.pageSize())
		return m, m.probeVisible()

	case tea.KeyHome:
		m.cursor = 0
		m.snapCursorToSession(1)
		m.clampOffset()
		return m, nil

	case tea.KeyEnd:
		m.cursor = len(m.rows) - 1
		m.snapCursorToSession(-1)
		m.clampOffset()
		return m, nil

	case tea.KeyEnter:
		return m.choose(tui.ActionResume)

	case tea.KeyTab:
		if m.selected() != nil {
			m.mode = modeActions
		}
		return m, nil

	case tea.KeyBackspace:
		if m.filter == "" {
			return m, nil
		}
		runes := []rune(m.filter)
		m.filter = string(runes[:len(runes)-1])
		return m, m.loadCmd()

	case tea.KeyCtrlA:
		return m.toggleScope()

	case tea.KeyCtrlR:
		return m, m.loadCmd()

	case tea.KeyCtrlK:
		m.openPalette()
		return m, nil

	case tea.KeySpace:
		m.filter += " "
		return m, m.loadCmd()

	case tea.KeyRunes:
		m.filter += string(key.Runes)
		return m, m.loadCmd()
	}
	return m, nil
}

// updateActions handles the Tab menu, where printable input is an accelerator.
func (m *Model) updateActions(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Type == tea.KeyCtrlC {
		return m.cancel()
	}
	if key.Type == tea.KeyEsc || key.Type == tea.KeyTab {
		m.mode = modeList
		return m, nil
	}
	if key.Type == tea.KeyEnter {
		m.mode = modeList
		return m.choose(tui.ActionResume)
	}
	if key.Type != tea.KeyRunes || len(key.Runes) != 1 {
		return m, nil
	}
	m.mode = modeList
	switch key.Runes[0] {
	case 'r':
		return m.choose(tui.ActionResume)
	case 'f':
		return m.choose(tui.ActionFork)
	case 'h':
		return m.choose(tui.ActionHandoff)
	case 'i':
		return m.choose(tui.ActionInspect)
	case 'y':
		return m.copyReference()
	case 'a':
		return m.toggleScope()
	case 'q':
		return m.cancel()
	default:
		return m, nil
	}
}

// openPalette shows the command overlay, hiding entries that need a session
// when none is selected.
func (m *Model) openPalette() {
	available := make([]palette.Command, 0, len(Commands))
	hasSession := m.selected() != nil
	for _, command := range Commands {
		if command.NeedsSession && !hasSession {
			continue
		}
		available = append(available, command)
	}
	m.palette = palette.New(m.theme, available, m.width, m.height)
	m.mode = modePalette
	m.status = ""
}

// updatePalette forwards keys to the overlay and acts on its choice.
func (m *Model) updatePalette(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.palette == nil {
		m.mode = modeList
		return m, nil
	}
	m.palette.Update(key)
	if m.palette.Open() {
		return m, nil
	}
	chosen := m.palette.Chosen()
	m.palette = nil
	m.mode = modeList
	return m.runCommand(chosen)
}

// runCommand maps a palette identifier onto an action.
//
// The identifier comes from the Commands table, never from typed text, so this
// switch is exhaustive over a closed set rather than parsing user input.
func (m *Model) runCommand(id string) (tea.Model, tea.Cmd) {
	switch id {
	case "":
		return m, nil
	case "resume":
		return m.choose(tui.ActionResume)
	case "fork":
		return m.choose(tui.ActionFork)
	case "handoff":
		return m.choose(tui.ActionHandoff)
	case "inspect":
		return m.choose(tui.ActionInspect)
	case "copy":
		return m.copyReference()
	case "scope":
		return m.toggleScope()
	case "refresh":
		return m, m.loadCmd()
	case "quit":
		return m.cancel()
	default:
		m.intent = tui.Intent{Action: tui.ActionCommand, Command: id}
		m.quitting = true
		return m, tea.Quit
	}
}

func (m *Model) toggleScope() (tea.Model, tea.Cmd) {
	if m.project == "" {
		m.status = "not inside a known project; already showing every session"
		return m, nil
	}
	if m.scope == ScopeProject {
		m.scope = ScopeAll
	} else {
		m.scope = ScopeProject
	}
	m.status = ""
	return m, m.loadCmd()
}

// copyReference puts the selected session reference on the clipboard using
// OSC 52, which travels over SSH and through tmux without a helper binary.
func (m *Model) copyReference() (tea.Model, tea.Cmd) {
	record := m.selected()
	if record == nil {
		return m, nil
	}
	reference := record.Reference()
	if m.clipboard == nil {
		// Without a clipboard the reference is still worth surfacing, since
		// the reader can select it by hand.
		m.status = reference
		return m, nil
	}
	copyFunc := m.clipboard
	return m, func() tea.Msg {
		return tui.ClipboardMsg{Text: reference, Err: copyFunc(reference)}
	}
}

func (m *Model) cancel() (tea.Model, tea.Cmd) {
	m.intent = tui.Intent{}
	m.quitting = true
	return m, tea.Quit
}

// choose records an intent and quits. The surface never performs the action.
func (m *Model) choose(action tui.Action) (tea.Model, tea.Cmd) {
	record := m.selected()
	if record == nil {
		return m, nil
	}
	m.intent = tui.Intent{Action: action, Reference: record.Reference()}
	m.quitting = true
	return m, tea.Quit
}

// probeVisible starts readiness probes for the rows currently on screen.
//
// Scoping to the viewport is deliberate. Probing the whole index would run
// workspace and vendor checks for hundreds of sessions the reader will never
// look at, which costs far more than it tells anyone.
func (m *Model) probeVisible() tea.Cmd {
	if m.readiness == nil || len(m.rows) == 0 {
		return nil
	}
	height := m.listHeight()
	end := m.offset + height
	if end > len(m.rows) {
		end = len(m.rows)
	}
	visible := make([]sessionindex.Record, 0, height)
	for index := m.offset; index < end; index++ {
		if !m.rows[index].isHeader() {
			visible = append(visible, m.rows[index].record)
		}
	}
	return m.readiness.Probe(m.ctx, visible)
}

// selected returns the record under the cursor, or nil when the list is empty.
func (m *Model) selected() *sessionindex.Record {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}
	current := m.rows[m.cursor]
	if current.isHeader() {
		return nil
	}
	return &m.rows[m.cursor].record
}

// rebuildRows regroups records into time sections and repairs the cursor.
func (m *Model) rebuildRows() {
	previousKey := ""
	if current := m.selected(); current != nil {
		previousKey = current.Key
	}
	m.rows = m.rows[:0]
	lastSection := ui.Section(-1)
	for _, record := range m.records {
		section := ui.SectionFor(record.UpdatedAt, m.now)
		if section != lastSection {
			m.rows = append(m.rows, row{header: section.Title()})
			lastSection = section
		}
		m.rows = append(m.rows, row{record: record})
	}
	// Keep the cursor on the same session across a refresh when it survived
	// the new filter; otherwise fall back to the first selectable row.
	m.cursor = 0
	if previousKey != "" {
		for index, candidate := range m.rows {
			if !candidate.isHeader() && candidate.record.Key == previousKey {
				m.cursor = index
				break
			}
		}
	}
	m.snapCursorToSession(1)
	m.clampOffset()
}

// moveCursor steps by delta rows, skipping headers and stopping at the ends.
func (m *Model) moveCursor(delta int) {
	if len(m.rows) == 0 || delta == 0 {
		return
	}
	direction := 1
	if delta < 0 {
		direction = -1
	}
	remaining := delta
	if remaining < 0 {
		remaining = -remaining
	}
	for ; remaining > 0; remaining-- {
		next := m.cursor + direction
		for next >= 0 && next < len(m.rows) && m.rows[next].isHeader() {
			next += direction
		}
		if next < 0 || next >= len(m.rows) {
			break
		}
		m.cursor = next
	}
	m.clampOffset()
}

// snapCursorToSession moves the cursor off a header in the given direction,
// then in the opposite direction if that ran off the end.
func (m *Model) snapCursorToSession(direction int) {
	if len(m.rows) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	index := m.cursor
	for index >= 0 && index < len(m.rows) && m.rows[index].isHeader() {
		index += direction
	}
	if index < 0 || index >= len(m.rows) {
		index = m.cursor
		for index >= 0 && index < len(m.rows) && m.rows[index].isHeader() {
			index -= direction
		}
	}
	if index >= 0 && index < len(m.rows) {
		m.cursor = index
	}
}

// pageSize is how many session rows fit in the list viewport.
func (m *Model) pageSize() int {
	size := m.listHeight()
	if size < 1 {
		return 1
	}
	return size
}

// clampOffset scrolls the viewport the minimum amount that keeps the cursor
// visible, and pulls a section header along when the cursor sits just under it.
func (m *Model) clampOffset() {
	height := m.listHeight()
	if height <= 0 || len(m.rows) == 0 {
		m.offset = 0
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+height {
		m.offset = m.cursor - height + 1
	}
	// A selected row directly beneath its heading looks orphaned when the
	// heading scrolls off, so keep the heading on screen when it is adjacent.
	if m.offset > 0 && m.offset == m.cursor && m.rows[m.offset-1].isHeader() {
		m.offset--
	}
	maxOffset := len(m.rows) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.offset > maxOffset {
		m.offset = maxOffset
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// StaticReadiness adapts a pure function to ReadinessProvider.
//
// It answers instantly and probes nothing, which is what a caller with
// precomputed readiness wants, and what a deterministic test needs. A nil
// function yields a disabled provider, so the status column is hidden.
func StaticReadiness(lookup func(sessionindex.Record) ui.Readiness) ReadinessProvider {
	if lookup == nil {
		return nil
	}
	return staticReadiness(lookup)
}

type staticReadiness func(sessionindex.Record) ui.Readiness

func (s staticReadiness) Enabled() bool { return s != nil }

func (s staticReadiness) Lookup(record sessionindex.Record) ui.Readiness { return s(record) }

func (s staticReadiness) Probe(context.Context, []sessionindex.Record) tea.Cmd { return nil }
