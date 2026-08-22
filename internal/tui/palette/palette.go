// Package palette is the command palette: one keystroke that reaches every
// verb, from anywhere.
//
// It exists so the interactive surfaces have a single discovery mechanism.
// Without it, every action has to earn a key on a key bar, and the key bar has
// room for about five. With it, the key bar can show the five that matter and
// everything else stays one ctrl+k away, findable by typing part of its name
// rather than by remembering a letter.
//
// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.
package palette

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/HarjjotSinghh/reinstate/internal/ui"
)

// Command is one entry.
type Command struct {
	// ID is the stable identifier returned when the entry is chosen.
	ID string
	// Title is what the reader sees.
	Title string
	// Detail explains what it does, in one short line.
	Detail string
	// Keys are alternative words that should match this entry, for the times
	// the reader's word for something is not ours.
	Keys []string
	// NeedsSession marks entries that act on the selected session, so they are
	// hidden when nothing is selected rather than failing after being chosen.
	NeedsSession bool
}

// Model is the palette overlay.
type Model struct {
	theme    ui.Theme
	commands []Command

	filter   string
	filtered []Command
	cursor   int

	width  int
	height int

	chosen string
	open   bool
}

// New builds a palette over the given commands.
func New(theme ui.Theme, commands []Command, width, height int) *Model {
	model := &Model{
		theme:    theme,
		commands: commands,
		width:    width,
		height:   height,
		open:     true,
	}
	model.refilter()
	return model
}

// Open reports whether the palette is still showing.
func (m *Model) Open() bool { return m != nil && m.open }

// Chosen returns the ID of the chosen command, or empty if none was.
func (m *Model) Chosen() string { return m.chosen }

// Resize updates the overlay dimensions.
func (m *Model) Resize(width, height int) {
	m.width, m.height = width, height
}

// Update applies one key press. It returns true when the key was consumed, so
// a host surface knows not to also act on it.
func (m *Model) Update(key tea.KeyMsg) bool {
	switch key.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.open = false
		return true
	case tea.KeyUp, tea.KeyCtrlP:
		m.move(-1)
		return true
	case tea.KeyDown, tea.KeyCtrlN:
		m.move(1)
		return true
	case tea.KeyEnter:
		if m.cursor >= 0 && m.cursor < len(m.filtered) {
			m.chosen = m.filtered[m.cursor].ID
		}
		m.open = false
		return true
	case tea.KeyBackspace:
		if m.filter != "" {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.refilter()
		}
		return true
	case tea.KeySpace:
		m.filter += " "
		m.refilter()
		return true
	case tea.KeyRunes:
		m.filter += string(key.Runes)
		m.refilter()
		return true
	}
	return true
}

func (m *Model) move(delta int) {
	if len(m.filtered) == 0 {
		m.cursor = 0
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

// refilter recomputes the visible entries.
//
// Matching is subsequence-based rather than substring, so "hof" finds "hand
// off" — which is how people actually type into a palette. Results are ranked
// by how early and how tightly the match lands, with a stable alphabetical
// tiebreak so the list never reshuffles between identical queries.
func (m *Model) refilter() {
	query := strings.ToLower(strings.TrimSpace(m.filter))
	m.filtered = m.filtered[:0]
	if query == "" {
		m.filtered = append(m.filtered, m.commands...)
		m.cursor = 0
		return
	}

	type scored struct {
		command Command
		score   int
	}
	matches := make([]scored, 0, len(m.commands))
	for _, command := range m.commands {
		best := -1
		for _, candidate := range append([]string{command.Title, command.ID}, command.Keys...) {
			if score := subsequenceScore(strings.ToLower(candidate), query); score >= 0 {
				if best < 0 || score < best {
					best = score
				}
			}
		}
		if best >= 0 {
			matches = append(matches, scored{command: command, score: best})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		return matches[i].command.Title < matches[j].command.Title
	})
	for _, match := range matches {
		m.filtered = append(m.filtered, match.command)
	}
	m.cursor = 0
}

// subsequenceScore returns a rank for query as a subsequence of candidate, or
// -1 when it does not match. Lower is better: a contiguous match at the start
// scores best.
func subsequenceScore(candidate, query string) int {
	if query == "" {
		return 0
	}
	queryRunes := []rune(query)
	index := 0
	firstAt := -1
	for position, char := range []rune(candidate) {
		if char != queryRunes[index] {
			continue
		}
		if firstAt < 0 {
			firstAt = position
		}
		index++
		if index == len(queryRunes) {
			// Span beyond the query length is the gap penalty; the start
			// offset breaks ties toward matches near the beginning.
			return (position - firstAt - len(queryRunes) + 1) + firstAt
		}
	}
	return -1
}
