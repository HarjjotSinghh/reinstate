// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import (
	"sort"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
	// The catalog registers its descriptors in init, so the blank import is what
	// makes agents.Keys() report the shipped agents.
	_ "github.com/HarjjotSinghh/reinstate/internal/agents/catalog"
)

func TestAgentLabel(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "a known key keeps its label", key: "claude", want: "claude"},
		{name: "a long key gets a short label", key: "antigravity", want: "antigrav"},
		{name: "a suffixed key gets the product name", key: "minimax-code", want: "minimax"},
		{name: "an unknown key falls back to itself", key: "totally-unknown-agent", want: "totally-unknown-agent"},
		{name: "a future catalog key falls back to itself", key: "zcode", want: "zcode"},
		{name: "an empty key falls back to empty", key: "", want: ""},
		{name: "the fallback is case-sensitive", key: "Claude", want: "Claude"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AgentLabel(tc.key); got != tc.want {
				t.Fatalf("AgentLabel(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

// TestAgentStylesMatchTheCatalog is the drift guard. The style table is keyed
// by catalog descriptor key, and nothing else checks that: a renamed or
// mistyped key would silently lose its identity colour and render the raw key
// instead, which is a change nobody notices in review.
//
// agents.Keys() is the enumerator, and the blank import of the catalog package
// above is what populates it.
func TestAgentStylesMatchTheCatalog(t *testing.T) {
	catalogKeys := agents.Keys()
	if len(catalogKeys) == 0 {
		t.Fatal("agents.Keys() is empty: the catalog import no longer registers descriptors")
	}
	known := make(map[string]bool, len(catalogKeys))
	for _, key := range catalogKeys {
		known[key] = true
	}

	styled := AgentKeys()
	sort.Strings(styled)
	for _, key := range styled {
		t.Run(key, func(t *testing.T) {
			if !known[key] {
				t.Fatalf("agentStyles has %q, which is not a catalog key; catalog keys are %v",
					key, catalogKeys)
			}
			if _, ok := agents.Get(key); !ok {
				t.Fatalf("agents.Get(%q) reports no descriptor", key)
			}
		})
	}
}

// TestCatalogAgentsWithoutStylesStillRender covers the other direction, which
// is deliberately allowed: adding a catalog entry must never require a UI
// change, so an unstyled agent falls back to its key.
func TestCatalogAgentsWithoutStylesStillRender(t *testing.T) {
	styled := make(map[string]bool)
	for _, key := range AgentKeys() {
		styled[key] = true
	}

	theme := NewTheme(monoCapability())
	var unstyled []string
	for _, key := range agents.Keys() {
		if styled[key] {
			continue
		}
		unstyled = append(unstyled, key)
		if got := AgentLabel(key); got != key {
			t.Errorf("AgentLabel(%q) = %q, want the key itself", key, got)
		}
		if got := theme.Agent(key, 12); Width(got) != 12 {
			t.Errorf("Agent(%q, 12) = %q, width %d, want 12", key, got, Width(got))
		}
	}
	if len(unstyled) > 0 {
		t.Logf("catalog agents rendering by fallback: %s", strings.Join(unstyled, ", "))
	}
}

// TestAgentKeysReflectsTheTable keeps the accessor honest, since the drift
// guard above is only as good as what it enumerates.
func TestAgentKeysReflectsTheTable(t *testing.T) {
	keys := AgentKeys()
	if len(keys) != len(agentStyles) {
		t.Fatalf("AgentKeys() returned %d keys, want %d", len(keys), len(agentStyles))
	}
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if seen[key] {
			t.Fatalf("AgentKeys() repeated %q", key)
		}
		seen[key] = true
		if _, ok := agentStyles[key]; !ok {
			t.Fatalf("AgentKeys() returned %q, which is not in the style table", key)
		}
	}
}

// TestAgentStyleLabelsAreUsable pins the two properties a column label has to
// have: it must say something, and it must not be mistakable for another
// agent's label.
func TestAgentStyleLabelsAreUsable(t *testing.T) {
	byLabel := make(map[string]string, len(agentStyles))
	for _, key := range AgentKeys() {
		style := agentStyles[key]
		t.Run(key, func(t *testing.T) {
			if strings.TrimSpace(style.Label) == "" {
				t.Fatalf("agentStyles[%q].Label is empty", key)
			}
			if style.Color == nil {
				t.Fatalf("agentStyles[%q].Color is nil", key)
			}
			if other, clash := byLabel[style.Label]; clash {
				t.Fatalf("agentStyles[%q].Label %q collides with %q", key, style.Label, other)
			}
		})
		byLabel[style.Label] = key
	}
}

// TestThemeAgentColorsKnownAgentsOnly documents the fallback path: an unknown
// key is rendered in the muted style rather than dropped or coloured at random.
func TestThemeAgentColorsKnownAgentsOnly(t *testing.T) {
	theme := NewTheme(colorCapability())
	cases := []struct {
		name string
		key  string
		want string
	}{
		{name: "known", key: "claude", want: "claude   "},
		{name: "unknown", key: "unknown-agent", want: "unknown-…"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := theme.Agent(tc.key, 9)
			// Styling is stripped so the assertion is about the cell content,
			// not about the colour profile of whatever runs the test.
			if plain := stripEscapes(got); plain != tc.want {
				t.Fatalf("Agent(%q, 9) = %q, want the cell to read %q", tc.key, plain, tc.want)
			}
		})
	}
}

// stripEscapes removes ANSI CSI sequences so a rendered cell can be compared as
// text. lipgloss chooses a colour profile from the process environment, so the
// raw bytes are not stable across machines; the cell content is.
func stripEscapes(s string) string {
	var builder strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			for i < len(s) && !isCSITerminator(s[i]) {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		builder.WriteByte(s[i])
		i++
	}
	return builder.String()
}

func isCSITerminator(b byte) bool { return b >= 0x40 && b <= 0x7e && b != '[' }
