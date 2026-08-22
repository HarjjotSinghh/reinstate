// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package switcher

import (
	"fmt"
	"sort"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Layout constants. The list columns are fixed so rows stay aligned as the
// terminal resizes; only the title column is elastic.
const (
	chromeHeight    = 4 // header, filter, key bar, status
	statusColumn    = 2 // readiness glyph plus one space
	agentColumn     = 9
	projectColumn   = 16
	ageColumn       = 9
	previewFraction = 40 // percent of width given to the preview pane
	minPreviewWidth = 30
	minListWidth    = 34
)

// listHeight is how many rows the list viewport can show.
func (m *Model) listHeight() int {
	height := m.height - chromeHeight
	if height < 1 {
		return 1
	}
	return height
}

// previewWidth is the width of the right pane, or zero in compact mode.
func (m *Model) previewWidth() int {
	if !m.capability.Split() || m.width <= 0 {
		return 0
	}
	width := m.width * previewFraction / 100
	if width < minPreviewWidth {
		width = minPreviewWidth
	}
	if m.width-width-1 < minListWidth {
		return 0
	}
	return width
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		// Leaving a final frame behind would collide with whatever the caller
		// draws next, most often a vendor CLI taking over the terminal.
		return ""
	}
	var builder strings.Builder
	builder.WriteString(m.headerLine())
	builder.WriteString("\n")
	builder.WriteString(m.filterLine())
	builder.WriteString("\n")
	builder.WriteString(m.overlayBody())
	builder.WriteString("\n")
	builder.WriteString(m.statusLine())
	builder.WriteString("\n")
	builder.WriteString(m.keyBar())
	return builder.String()
}

func (m *Model) headerLine() string {
	sessions := len(m.records)
	agentSet := map[string]struct{}{}
	for _, record := range m.records {
		agentSet[record.Agent] = struct{}{}
	}
	scope := "all projects"
	if m.scope == ScopeProject && m.projectLabel != "" {
		scope = m.projectLabel
	}
	left := m.theme.Title.Render("rein")
	middle := m.theme.Muted.Render(fmt.Sprintf(
		"%s · %s · %s",
		plural(sessions, "session", "sessions"),
		plural(len(agentSet), "agent", "agents"),
		ui.Truncate(scope, 28, m.theme.Glyphs.Ellipsis),
	))
	return m.spread(left, middle)
}

func (m *Model) filterLine() string {
	prompt := m.theme.Accent.Render(m.theme.Glyphs.Search)
	text := m.filter
	if text == "" {
		text = m.theme.Muted.Render("type to filter")
	}
	left := prompt + " " + text
	right := ""
	if m.mode == modeActions {
		right = m.theme.Accent.Render("actions")
	}
	return m.spread(left, right)
}

// overlayBody draws the body, with the command palette laid over the middle of
// it when open. The list stays visible around the overlay so the reader keeps
// their place; a palette that replaced the screen would lose it.
func (m *Model) overlayBody() string {
	body := m.bodyBlock()
	if m.palette == nil || !m.palette.Open() {
		return body
	}
	overlay := m.palette.Lines(m.width)
	if len(overlay) == 0 {
		return body
	}
	lines := strings.Split(body, "\n")
	start := (len(lines) - len(overlay)) / 2
	if start < 0 {
		start = 0
	}
	for offset, line := range overlay {
		index := start + offset
		if index >= len(lines) {
			break
		}
		lines[index] = line
	}
	return strings.Join(lines, "\n")
}

// bodyBlock renders the list and, when there is room, the preview pane.
func (m *Model) bodyBlock() string {
	height := m.listHeight()
	previewWidth := m.previewWidth()
	listWidth := m.width
	if previewWidth > 0 {
		listWidth = m.width - previewWidth - 1
	}

	listLines := m.listLines(listWidth, height)
	if previewWidth == 0 {
		return strings.Join(listLines, "\n")
	}
	previewLines := m.previewLines(previewWidth, height)

	separator := m.theme.Border.Render(m.theme.Glyphs.VerticalBar)
	rendered := make([]string, height)
	for index := 0; index < height; index++ {
		left := ""
		if index < len(listLines) {
			left = listLines[index]
		}
		right := ""
		if index < len(previewLines) {
			right = previewLines[index]
		}
		rendered[index] = ui.Pad(left, listWidth) + separator + right
	}
	return strings.Join(rendered, "\n")
}

// listLines renders the visible slice of rows.
func (m *Model) listLines(width, height int) []string {
	lines := make([]string, 0, height)
	if len(m.rows) == 0 {
		return append(lines, m.emptyStateLines(width, height)...)
	}
	for index := m.offset; index < len(m.rows) && len(lines) < height; index++ {
		lines = append(lines, m.renderRow(m.rows[index], index == m.cursor, width))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func (m *Model) renderRow(current row, selected bool, width int) string {
	if current.isHeader() {
		return m.theme.SectionHd.Render(ui.Truncate(current.header, width, m.theme.Glyphs.Ellipsis))
	}
	record := current.record

	cursor := "  "
	if selected {
		cursor = m.theme.Selected.Render(m.theme.Glyphs.Cursor) + " "
	}
	remaining := width - ui.Width(m.theme.Glyphs.Cursor) - 1

	status := ""
	if m.readiness != nil {
		status = m.theme.Glyph(m.readiness.Lookup(record)) + " "
		remaining -= statusColumn
	}

	agent := m.theme.Agent(record.Agent, agentColumn)
	remaining -= agentColumn

	remaining -= projectColumn + 1

	age := ui.Relative(record.UpdatedAt, m.now)
	remaining -= ageColumn + 1

	// The project and title columns share whatever the fixed columns leave, and
	// the title never gets less than the project: it is what the reader is
	// scanning for, while the project is context the preview repeats anyway.
	// The two must also add up exactly, because a row wider than the list pushes
	// the preview separator out of true and, in compact mode, wraps onto the
	// next line.
	projectWidth := projectColumn
	titleWidth := remaining
	if titleWidth < projectWidth {
		budget := projectWidth + titleWidth
		if budget < 0 {
			budget = 0
		}
		titleWidth = (budget + 1) / 2
		projectWidth = budget - titleWidth
	}

	project := ui.Fit(ui.Sanitize(record.Project), projectWidth, m.theme.Glyphs.Ellipsis)
	title := ui.Fit(displayTitle(record), titleWidth, m.theme.Glyphs.Ellipsis)
	if selected {
		title = m.theme.Selected.Render(title)
	}

	return cursor + status + agent + m.theme.Muted.Render(project) + " " + title + " " +
		m.theme.Muted.Render(ui.Fit(age, ageColumn, m.theme.Glyphs.Ellipsis))
}

// emptyStateLines explains what was searched rather than showing a blank pane.
// An empty list is the moment a user most needs to know what Reinstate looked
// at, so the scope and the filter are both restated.
func (m *Model) emptyStateLines(width, height int) []string {
	// Every message is bounded to the pane. These sentences are longer than a
	// narrow list column, and a line that overruns the column wraps into the
	// preview pane instead of staying in its own.
	inner := width - 2
	if inner < 1 {
		inner = 1
	}
	var messages []string
	plain := func(text string) {
		messages = append(messages, ui.Wrap(text, inner)...)
	}
	styled := func(style interface{ Render(...string) string }, text string) {
		for _, line := range ui.Wrap(text, inner) {
			messages = append(messages, style.Render(line))
		}
	}
	switch {
	case m.loadErr != nil:
		styled(m.theme.Blocked, "could not read the local session index")
		styled(m.theme.Muted, ui.Sanitize(m.loadErr.Error()))
	case m.filter != "":
		// The fixed words are 19 cells; the rest of the column is the query.
		messages = append(messages, "No sessions match "+
			m.theme.Accent.Render(ui.Truncate(m.filter, inner-19, m.theme.Glyphs.Ellipsis))+".")
		styled(m.theme.Muted, "Backspace to widen the filter.")
		if m.scope == ScopeProject && m.project != "" {
			styled(m.theme.Muted, "ctrl+a searches every project.")
		}
	case m.scope == ScopeProject && m.project != "":
		messages = append(messages, "No sessions indexed for "+
			m.theme.Accent.Render(ui.Truncate(m.projectLabel, inner-25, m.theme.Glyphs.Ellipsis))+".")
		styled(m.theme.Muted, "ctrl+a shows every project.")
	default:
		plain("No coding-agent sessions found on this device.")
		styled(m.theme.Muted, "Start a session in Claude Code, Codex, or another supported agent,")
		styled(m.theme.Muted, "then run rein again. `rein doctor --agents` lists what was scanned.")
	}
	lines := make([]string, 0, height)
	lines = append(lines, "")
	for _, message := range messages {
		if len(lines) >= height {
			break
		}
		lines = append(lines, "  "+message)
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

// previewLines renders the detail pane for the selected session.
func (m *Model) previewLines(width, height int) []string {
	inner := width - 2
	if inner < 8 {
		inner = 8
	}
	lines := make([]string, 0, height)
	add := func(text string) {
		if len(lines) < height {
			lines = append(lines, " "+text)
		}
	}
	record := m.selected()
	if record == nil {
		for len(lines) < height {
			lines = append(lines, "")
		}
		return lines
	}

	add(m.theme.Agent(record.Agent, 0) + m.theme.Muted.Render(" · ") +
		m.theme.Title.Render(ui.Truncate(ui.Sanitize(record.Project), inner-12, m.theme.Glyphs.Ellipsis)))
	add(m.theme.Title.Render(ui.Truncate(displayTitle(*record), inner, m.theme.Glyphs.Ellipsis)))

	meta := ui.Relative(record.UpdatedAt, m.now)
	if branch := ui.Sanitize(record.Branch); branch != "" {
		meta = ui.Truncate(branch, inner-len(meta)-3, m.theme.Glyphs.Ellipsis) + " · " + meta
	}
	add(m.theme.Muted.Render(ui.Truncate(meta, inner, m.theme.Glyphs.Ellipsis)))
	add("")

	// Readiness, and why. The verdict alone is not enough for a read-only
	// agent: "CANNOT RESUME" with nothing beside it looks like a fault to be
	// fixed, when the truth is that this vendor has no resume path at all and
	// nothing the reader does will change it.
	switch {
	case m.readiness != nil:
		readiness := m.readiness.Lookup(*record)
		add(m.theme.Glyph(readiness) + " " + m.readinessStyle(readiness).Render(strings.ToUpper(readiness.Label())))
		if reason := ui.Sanitize(record.ReadOnlyReason); reason != "" {
			for _, wrapped := range ui.Wrap(reason, inner-2) {
				add("  " + m.theme.Muted.Render(wrapped))
			}
		}
		add("")
	case record.ReadOnlyReason != "":
		add(m.theme.Blocked.Render(m.theme.Glyphs.BlockedMark) + " " + m.theme.Muted.Render("read-only"))
		// Wrapped rather than truncated, for the same reason as above: a reason
		// cut off mid-sentence explains nothing.
		for _, wrapped := range ui.Wrap(ui.Sanitize(record.ReadOnlyReason), inner-2) {
			add("  " + m.theme.Muted.Render(wrapped))
		}
		add("")
	}

	if record.MessageCount > 0 {
		add(m.theme.Muted.Render(plural(record.MessageCount, "message", "messages")))
	}
	if count := len(record.Files); count > 0 {
		add(m.theme.Muted.Render(plural(count, "file touched", "files touched")))
		for _, file := range topFiles(record.Files, 3) {
			add(m.theme.Muted.Render("  " + ui.Truncate(file, inner-2, m.theme.Glyphs.Ellipsis)))
		}
	}

	// The prompt preview is skipped when the title is already that same text.
	// displayTitle falls back to the first prompt for vendors that write no
	// title, so for those sessions the two lines would otherwise be identical.
	if preview := ui.Preview(record.PromptPreview); preview != "" && preview != displayTitle(*record) {
		add("")
		for _, wrapped := range ui.Wrap(preview, inner) {
			add(m.theme.Muted.Render(wrapped))
		}
	}

	// The canonical reference lives at the bottom so it is always in the same
	// place when someone wants to copy it.
	for len(lines) < height-1 {
		lines = append(lines, "")
	}
	if len(lines) < height {
		lines = append(lines, " "+m.theme.Muted.Render(ui.Truncate(record.Reference(), inner, m.theme.Glyphs.Ellipsis)))
	}
	return lines
}

func (m *Model) readinessStyle(readiness ui.Readiness) interface{ Render(...string) string } {
	switch readiness {
	case ui.ReadinessReady:
		return m.theme.Ready
	case ui.ReadinessWarn:
		return m.theme.Warn
	case ui.ReadinessBlocked:
		return m.theme.Blocked
	default:
		return m.theme.Pending
	}
}

func (m *Model) statusLine() string {
	if m.status == "" {
		return ""
	}
	return " " + m.theme.Muted.Render(ui.Truncate(m.status, m.width-1, m.theme.Glyphs.Ellipsis))
}

// keyBar lists the keys that are live in the current mode. Showing keys that do
// nothing right now is worse than showing none.
func (m *Model) keyBar() string {
	if m.mode == modePalette {
		return m.renderKeys([][2]string{
			{m.theme.Glyphs.Enter, "run"},
			{"esc", "close"},
		})
	}
	if m.mode == modeActions {
		return m.renderKeys([][2]string{
			{"r", "resume"},
			{"f", "fork"},
			{"h", "hand off"},
			{"i", "inspect"},
			{"y", "copy ref"},
			{"esc", "back"},
		})
	}
	// Both the scope toggle and the palette stay on the bar. The palette can
	// reach the scope toggle, but a key that only exists inside another menu is
	// a key nobody finds.
	keys := [][2]string{
		{m.theme.Glyphs.Enter, "resume"},
		{"tab", "actions"},
		{"ctrl+a", "scope"},
		{"ctrl+k", "commands"},
	}
	if m.filter != "" {
		keys = append(keys, [2]string{"esc", "clear"})
	} else {
		keys = append(keys, [2]string{"esc", "quit"})
	}
	return m.renderKeys(keys)
}

func (m *Model) renderKeys(pairs [][2]string) string {
	parts := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		parts = append(parts, m.theme.KeyCap.Render(pair[0])+" "+m.theme.KeyBar.Render(pair[1]))
	}
	bar := " " + strings.Join(parts, m.theme.KeyBar.Render("   "))
	return ui.Truncate(bar, m.width, m.theme.Glyphs.Ellipsis)
}

// spread places left and right on one line with the gap between them.
func (m *Model) spread(left, right string) string {
	if right == "" {
		return " " + ui.Truncate(left, m.width-1, m.theme.Glyphs.Ellipsis)
	}
	gap := m.width - ui.Width(left) - ui.Width(right) - 2
	if gap < 1 {
		return " " + ui.Truncate(left, m.width-1, m.theme.Glyphs.Ellipsis)
	}
	return " " + left + strings.Repeat(" ", gap) + right
}

// displayTitle picks the most useful line of text for a session.
//
// Several vendors, Codex among them, never write a title, and the index falls
// back to the native session identifier so the JSON contract always carries
// something. A column of UUIDs is precisely the listing this surface exists to
// replace, so the view prefers the first user prompt whenever the stored title
// is really just the identifier.
//
// This substitution is presentational and deliberately does not reach the
// index: Record.Title stays exactly what --json has always reported.
func displayTitle(record sessionindex.Record) string {
	title := ui.Sanitize(record.Title)
	if title != "" && title != record.ID && title != record.Key {
		return title
	}
	if preview := ui.Sanitize(record.PromptPreview); preview != "" {
		return preview
	}
	if title != "" {
		return title
	}
	return record.ID
}

func plural(count int, singular, pluralWord string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, pluralWord)
}

// topFiles returns a stable, bounded selection of touched files. Sorting keeps
// the preview from reordering between identical loads.
func topFiles(files []string, limit int) []string {
	cleaned := make([]string, 0, len(files))
	for _, file := range files {
		if trimmed := ui.Sanitize(file); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	sort.Strings(cleaned)
	if len(cleaned) > limit {
		cleaned = cleaned[:limit]
	}
	return cleaned
}
