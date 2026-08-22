// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import "github.com/charmbracelet/lipgloss"

// Glyphs are the single-cell symbols the interactive surfaces draw. Two sets
// exist so a terminal that cannot render box drawing still produces aligned,
// readable output rather than mojibake.
type Glyphs struct {
	ReadyMark    string // resume is clear
	WarnMark     string // resume needs acknowledgement
	BlockedMark  string // resume is refused
	PendingMark  string // readiness still being computed
	CheckOn      string // checklist item selected
	CheckOff     string // checklist item not selected
	Included     string // capsule section present
	Excluded     string // capsule section absent
	Cursor       string // selected row marker
	TrailLink    string // continuity trail connector
	LeftArrow    string
	RightArrow   string
	Enter        string
	Ellipsis     string
	VerticalBar  string
	HorizontalBr string
	// Search prefixes a line that accepts typed input.
	//
	// U+276F rather than a magnifier: the magnifier glyphs (U+2315, U+1F50D)
	// are absent from most monospace faces and from every legacy console font,
	// so they arrive as a placeholder box or a stray dot. This one ships with
	// the programming fonts people actually use — it is the character prompts
	// like starship draw — and it reads as "type here", which is what the line
	// is for.
	Search string
}

var unicodeGlyphs = Glyphs{
	ReadyMark:    "●",
	WarnMark:     "◐",
	BlockedMark:  "○",
	PendingMark:  "◌",
	CheckOn:      "[x]",
	CheckOff:     "[ ]",
	Included:     "✔",
	Excluded:     "✖",
	Cursor:       "▸",
	TrailLink:    "⤷",
	LeftArrow:    "◂",
	RightArrow:   "▸",
	Enter:        "↵",
	Ellipsis:     "…",
	VerticalBar:  "│",
	HorizontalBr: "─",
	Search:       "\u276f",
}

var asciiGlyphs = Glyphs{
	ReadyMark:    "*",
	WarnMark:     "!",
	BlockedMark:  "x",
	PendingMark:  ".",
	CheckOn:      "[x]",
	CheckOff:     "[ ]",
	Included:     "+",
	Excluded:     "-",
	Cursor:       ">",
	TrailLink:    "->",
	LeftArrow:    "<",
	RightArrow:   ">",
	Enter:        "enter",
	Ellipsis:     "...",
	VerticalBar:  "|",
	HorizontalBr: "-",
	// Not ">": that is already the ASCII cursor, and a filter prompt that looks
	// exactly like a selected row is one ambiguity too many in a set that has
	// no colour to fall back on.
	Search: "/",
}

// Palette holds the semantic colours. Every field is a lipgloss adaptive
// colour so light and dark terminals both stay legible.
type Palette struct {
	Ready    lipgloss.TerminalColor
	Warn     lipgloss.TerminalColor
	Blocked  lipgloss.TerminalColor
	Pending  lipgloss.TerminalColor
	Primary  lipgloss.TerminalColor
	Muted    lipgloss.TerminalColor
	Border   lipgloss.TerminalColor
	Accent   lipgloss.TerminalColor
	Inverted lipgloss.TerminalColor
}

var colorPalette = Palette{
	Ready:    lipgloss.AdaptiveColor{Light: "#1a7f37", Dark: "#3fb950"},
	Warn:     lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#d29922"},
	Blocked:  lipgloss.AdaptiveColor{Light: "#82071e", Dark: "#f85149"},
	Pending:  lipgloss.AdaptiveColor{Light: "#6e7781", Dark: "#8b949e"},
	Primary:  lipgloss.AdaptiveColor{Light: "#1f2328", Dark: "#e6edf3"},
	Muted:    lipgloss.AdaptiveColor{Light: "#6e7781", Dark: "#8b949e"},
	Border:   lipgloss.AdaptiveColor{Light: "#d0d7de", Dark: "#30363d"},
	Accent:   lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"},
	Inverted: lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#0d1117"},
}

// monoPalette resolves every semantic colour to no colour at all. Style objects
// still exist, so callers need no branching; they simply render unstyled.
var monoPalette = Palette{
	Ready:    lipgloss.NoColor{},
	Warn:     lipgloss.NoColor{},
	Blocked:  lipgloss.NoColor{},
	Pending:  lipgloss.NoColor{},
	Primary:  lipgloss.NoColor{},
	Muted:    lipgloss.NoColor{},
	Border:   lipgloss.NoColor{},
	Accent:   lipgloss.NoColor{},
	Inverted: lipgloss.NoColor{},
}

// Theme is the resolved look for one run: glyph set, palette, and the derived
// styles the views compose.
type Theme struct {
	Glyphs  Glyphs
	Palette Palette
	Color   bool

	Title     lipgloss.Style
	Muted     lipgloss.Style
	Accent    lipgloss.Style
	Ready     lipgloss.Style
	Warn      lipgloss.Style
	Blocked   lipgloss.Style
	Pending   lipgloss.Style
	Selected  lipgloss.Style
	Border    lipgloss.Style
	SectionHd lipgloss.Style
	KeyBar    lipgloss.Style
	KeyCap    lipgloss.Style
	Command   lipgloss.Style
}

// NewTheme resolves a theme from a detected capability.
func NewTheme(capability Capability) Theme {
	glyphs := asciiGlyphs
	if capability.Unicode {
		glyphs = unicodeGlyphs
	}
	palette := monoPalette
	color := capability.Color != ColorNone
	if color {
		palette = colorPalette
	}
	theme := Theme{Glyphs: glyphs, Palette: palette, Color: color}

	theme.Title = lipgloss.NewStyle().Foreground(palette.Primary).Bold(color)
	theme.Muted = lipgloss.NewStyle().Foreground(palette.Muted)
	theme.Accent = lipgloss.NewStyle().Foreground(palette.Accent)
	theme.Ready = lipgloss.NewStyle().Foreground(palette.Ready)
	theme.Warn = lipgloss.NewStyle().Foreground(palette.Warn)
	theme.Blocked = lipgloss.NewStyle().Foreground(palette.Blocked)
	theme.Pending = lipgloss.NewStyle().Foreground(palette.Pending)
	theme.Border = lipgloss.NewStyle().Foreground(palette.Border)
	theme.SectionHd = lipgloss.NewStyle().Foreground(palette.Muted).Bold(color)
	theme.KeyBar = lipgloss.NewStyle().Foreground(palette.Muted)
	theme.KeyCap = lipgloss.NewStyle().Foreground(palette.Accent).Bold(color)
	theme.Command = lipgloss.NewStyle().Foreground(palette.Accent)

	// Selection must remain visible without colour, so it carries bold and a
	// cursor glyph rather than relying on a background wash.
	theme.Selected = lipgloss.NewStyle().Foreground(palette.Primary).Bold(true)
	if color {
		theme.Selected = theme.Selected.Foreground(palette.Accent)
	}
	return theme
}

// Readiness is how resumable a session is, computed from the preflight report.
type Readiness int

const (
	ReadinessUnknown Readiness = iota // not computed yet
	ReadinessReady                    // clean resume
	ReadinessWarn                     // resume needs acknowledgement
	ReadinessBlocked                  // resume refused, or the agent is read-only
)

// Glyph returns the themed status symbol for a readiness value.
func (t Theme) Glyph(readiness Readiness) string {
	switch readiness {
	case ReadinessReady:
		return t.Ready.Render(t.Glyphs.ReadyMark)
	case ReadinessWarn:
		return t.Warn.Render(t.Glyphs.WarnMark)
	case ReadinessBlocked:
		return t.Blocked.Render(t.Glyphs.BlockedMark)
	default:
		return t.Pending.Render(t.Glyphs.PendingMark)
	}
}

// Label returns the plain word for a readiness value, for the preview pane and
// for plain-mode output.
func (r Readiness) Label() string {
	switch r {
	case ReadinessReady:
		return "ready to resume"
	case ReadinessWarn:
		return "needs acknowledgement"
	case ReadinessBlocked:
		return "cannot resume"
	default:
		return "checking"
	}
}
