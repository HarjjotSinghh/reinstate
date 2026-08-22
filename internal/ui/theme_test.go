// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// allReadiness is every readiness value plus one out of range, so the default
// arm of each switch is exercised rather than assumed.
var allReadiness = []Readiness{
	ReadinessUnknown,
	ReadinessReady,
	ReadinessWarn,
	ReadinessBlocked,
	Readiness(42),
}

// colorCapability and monoCapability are the two ends of the theme decision.
// They are built directly rather than through Detect: a theme depends only on
// the resolved capability, and pinning it here keeps the ladder out of the
// theme's tests.
func colorCapability() Capability {
	return Capability{Mode: ModeFull, Color: ColorTrue, Unicode: true, Width: 120, Height: 40}
}

func monoCapability() Capability {
	return Capability{Mode: ModeCompact, Color: ColorNone, Unicode: false, Width: 60, Height: 24}
}

func TestNewThemeResolvesColorAndGlyphSet(t *testing.T) {
	cases := []struct {
		name        string
		capability  Capability
		wantColor   bool
		wantGlyphs  Glyphs
		wantPalette Palette
	}{
		{
			name:        "truecolor and unicode",
			capability:  colorCapability(),
			wantColor:   true,
			wantGlyphs:  unicodeGlyphs,
			wantPalette: colorPalette,
		},
		{
			name:        "no colour and no unicode",
			capability:  monoCapability(),
			wantColor:   false,
			wantGlyphs:  asciiGlyphs,
			wantPalette: monoPalette,
		},
		{
			name:        "sixteen colours still count as colour",
			capability:  Capability{Mode: ModeFull, Color: Color16, Unicode: true},
			wantColor:   true,
			wantGlyphs:  unicodeGlyphs,
			wantPalette: colorPalette,
		},
		{
			name:        "colour without unicode keeps the ascii glyphs",
			capability:  Capability{Mode: ModeFull, Color: Color256, Unicode: false},
			wantColor:   true,
			wantGlyphs:  asciiGlyphs,
			wantPalette: colorPalette,
		},
		{
			name:        "unicode without colour keeps the unicode glyphs",
			capability:  Capability{Mode: ModeFull, Color: ColorNone, Unicode: true},
			wantColor:   false,
			wantGlyphs:  unicodeGlyphs,
			wantPalette: monoPalette,
		},
		{
			name:        "a plain capability is monochrome ascii",
			capability:  Capability{Mode: ModePlain, Reason: ReasonNotTTY},
			wantColor:   false,
			wantGlyphs:  asciiGlyphs,
			wantPalette: monoPalette,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(tc.capability)
			if theme.Color != tc.wantColor {
				t.Fatalf("NewTheme(%+v).Color = %t, want %t", tc.capability, theme.Color, tc.wantColor)
			}
			if theme.Glyphs != tc.wantGlyphs {
				t.Fatalf("NewTheme(%+v).Glyphs = %+v, want %+v", tc.capability, theme.Glyphs, tc.wantGlyphs)
			}
			if theme.Palette != tc.wantPalette {
				t.Fatalf("NewTheme(%+v).Palette = %+v, want %+v", tc.capability, theme.Palette, tc.wantPalette)
			}
		})
	}
}

// TestMonoThemeHasNoPalette states the guarantee a NO_COLOR user is owed: the
// palette resolves to no colour at all, so no style can emit one.
func TestMonoThemeHasNoPalette(t *testing.T) {
	theme := NewTheme(monoCapability())
	fields := map[string]lipgloss.TerminalColor{
		"Ready":    theme.Palette.Ready,
		"Warn":     theme.Palette.Warn,
		"Blocked":  theme.Palette.Blocked,
		"Pending":  theme.Palette.Pending,
		"Primary":  theme.Palette.Primary,
		"Muted":    theme.Palette.Muted,
		"Border":   theme.Palette.Border,
		"Accent":   theme.Palette.Accent,
		"Inverted": theme.Palette.Inverted,
	}
	for name, color := range fields {
		if _, ok := color.(lipgloss.NoColor); !ok {
			t.Errorf("mono Palette.%s = %#v, want lipgloss.NoColor", name, color)
		}
	}
}

// TestMonoThemeGlyphsCarryNoEscapes is the assertion that matters for a piped
// or NO_COLOR session: readiness still has to be readable, and it may not do it
// with an escape sequence.
func TestMonoThemeGlyphsCarryNoEscapes(t *testing.T) {
	cases := []struct {
		name       string
		capability Capability
	}{
		{name: "ascii mono", capability: monoCapability()},
		{
			name:       "unicode mono",
			capability: Capability{Mode: ModeFull, Color: ColorNone, Unicode: true},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(tc.capability)
			for _, readiness := range allReadiness {
				got := theme.Glyph(readiness)
				if strings.ContainsRune(got, 0x1b) {
					t.Errorf("mono Glyph(%s) = %q, carries an escape byte", readiness.Label(), got)
				}
				if got == "" {
					t.Errorf("mono Glyph(%s) is empty", readiness.Label())
				}
			}
		})
	}
}

// TestGlyphsAreDistinctPerReadiness keeps the four states tellable apart. In
// the ASCII set that is the only thing separating them, since no colour is
// available to help.
func TestGlyphsAreDistinctPerReadiness(t *testing.T) {
	cases := []struct {
		name       string
		capability Capability
	}{
		{name: "unicode", capability: Capability{Unicode: true}},
		{name: "ascii", capability: Capability{Unicode: false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(tc.capability)
			seen := map[string]Readiness{}
			for _, readiness := range []Readiness{ReadinessUnknown, ReadinessReady, ReadinessWarn, ReadinessBlocked} {
				got := theme.Glyph(readiness)
				if previous, clash := seen[got]; clash {
					t.Fatalf("Glyph(%s) = %q collides with Glyph(%s)", readiness.Label(), got, previous.Label())
				}
				seen[got] = readiness
			}
			// An unknown value must read as pending rather than as a fifth state.
			if got, want := theme.Glyph(Readiness(42)), theme.Glyph(ReadinessUnknown); got != want {
				t.Fatalf("Glyph(out of range) = %q, want the pending glyph %q", got, want)
			}
		})
	}
}

// TestGlyphSetsAreSingleCell keeps the status column one cell wide in both
// sets, which is what lets a row stay aligned across terminals.
func TestGlyphSetsAreSingleCell(t *testing.T) {
	cases := []struct {
		name    string
		unicode bool
	}{
		{name: "unicode", unicode: true},
		{name: "ascii", unicode: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			theme := NewTheme(Capability{Unicode: tc.unicode})
			for _, readiness := range allReadiness {
				if got := Width(theme.Glyph(readiness)); got != 1 {
					t.Errorf("Glyph(%s) is %d cells wide, want 1", readiness.Label(), got)
				}
			}
		})
	}
}

func TestReadinessLabel(t *testing.T) {
	cases := []struct {
		readiness Readiness
		want      string
	}{
		{readiness: ReadinessUnknown, want: "checking"},
		{readiness: ReadinessReady, want: "ready to resume"},
		{readiness: ReadinessWarn, want: "needs acknowledgement"},
		{readiness: ReadinessBlocked, want: "cannot resume"},
		{readiness: Readiness(42), want: "checking"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.readiness.Label(); got != tc.want {
				t.Fatalf("Readiness(%d).Label() = %q, want %q", int(tc.readiness), got, tc.want)
			}
		})
	}
}

// TestThemeAgentIsExactlyTheColumnWidth is the column guarantee for the agent
// cell. It has to hold for a key with an identity and for one without, because
// an unknown key falls back to the raw key, which can be longer than the
// column: the switcher subtracts a fixed width for this cell and everything
// after it would shift.
func TestThemeAgentIsExactlyTheColumnWidth(t *testing.T) {
	themes := []struct {
		name  string
		theme Theme
	}{
		{name: "colour", theme: NewTheme(colorCapability())},
		{name: "mono", theme: NewTheme(monoCapability())},
	}
	keys := []string{
		"claude",
		"codex",
		"antigravity",
		"minimax-code",
		"totally-unknown-agent",
		"",
		"日本語エージェント",
	}

	for _, themeCase := range themes {
		for _, key := range keys {
			name := themeCase.name + "/" + key
			if key == "" {
				name = themeCase.name + "/empty"
			}
			t.Run(name, func(t *testing.T) {
				for width := 1; width <= 16; width++ {
					got := themeCase.theme.Agent(key, width)
					if Width(got) != width {
						t.Fatalf("Agent(%q, %d) = %q, width %d, want exactly %d",
							key, width, got, Width(got), width)
					}
				}
			})
		}
	}
}

// TestThemeAgentUnpaddedAtZeroWidth covers the preview pane, which places the
// agent inline rather than in a column and must not receive padding.
func TestThemeAgentUnpaddedAtZeroWidth(t *testing.T) {
	cases := []struct {
		name  string
		theme Theme
	}{
		{name: "colour", theme: NewTheme(colorCapability())},
		{name: "mono", theme: NewTheme(monoCapability())},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Width(tc.theme.Agent("claude", 0)); got != Width("claude") {
				t.Fatalf("Agent(claude, 0) is %d cells, want %d", got, Width("claude"))
			}
			if got := Width(tc.theme.Agent("totally-unknown-agent", 0)); got != Width("totally-unknown-agent") {
				t.Fatalf("Agent(unknown, 0) is %d cells, want %d", got, Width("totally-unknown-agent"))
			}
		})
	}
}

// TestMonoThemeAgentCarriesNoEscapes keeps the mono contract on the one method
// that reaches for lipgloss styling per agent.
func TestMonoThemeAgentCarriesNoEscapes(t *testing.T) {
	theme := NewTheme(monoCapability())
	for _, key := range append(AgentKeys(), "totally-unknown-agent") {
		got := theme.Agent(key, 9)
		if strings.ContainsRune(got, 0x1b) {
			t.Fatalf("mono Agent(%q, 9) = %q, carries an escape byte", key, got)
		}
		if strings.TrimSpace(got) == "" {
			t.Fatalf("mono Agent(%q, 9) = %q, rendered nothing", key, got)
		}
	}
}
