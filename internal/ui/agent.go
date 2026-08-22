// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package ui

import "github.com/charmbracelet/lipgloss"

// AgentStyle is the stable visual identity of one catalog agent: a short label
// that fits a fixed column and a colour that stays the same everywhere the
// agent appears.
//
// The keys match internal/agents/catalog descriptor keys. An agent absent from
// this table still renders — it falls back to its key and the muted colour — so
// adding a catalog entry never requires touching the UI.
type AgentStyle struct {
	Label string
	Color lipgloss.TerminalColor
}

var agentStyles = map[string]AgentStyle{
	"claude":       {Label: "claude", Color: lipgloss.AdaptiveColor{Light: "#a1440a", Dark: "#e8956b"}},
	"codex":        {Label: "codex", Color: lipgloss.AdaptiveColor{Light: "#1f2328", Dark: "#c9d1d9"}},
	"gemini":       {Label: "gemini", Color: lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"}},
	"opencode":     {Label: "opencode", Color: lipgloss.AdaptiveColor{Light: "#6639ba", Dark: "#a371f7"}},
	"grok":         {Label: "grok", Color: lipgloss.AdaptiveColor{Light: "#57606a", Dark: "#9198a1"}},
	"kimi":         {Label: "kimi", Color: lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#2dd4bf"}},
	"qwen":         {Label: "qwen", Color: lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#d29922"}},
	"cursor":       {Label: "cursor", Color: lipgloss.AdaptiveColor{Light: "#1f2328", Dark: "#e6edf3"}},
	"copilot":      {Label: "copilot", Color: lipgloss.AdaptiveColor{Light: "#1a7f37", Dark: "#3fb950"}},
	"cline":        {Label: "cline", Color: lipgloss.AdaptiveColor{Light: "#0550ae", Dark: "#79c0ff"}},
	"roo":          {Label: "roo", Color: lipgloss.AdaptiveColor{Light: "#8250df", Dark: "#d2a8ff"}},
	"aider":        {Label: "aider", Color: lipgloss.AdaptiveColor{Light: "#bf3989", Dark: "#f778ba"}},
	"amp":          {Label: "amp", Color: lipgloss.AdaptiveColor{Light: "#953800", Dark: "#ffa657"}},
	"pi":           {Label: "pi", Color: lipgloss.AdaptiveColor{Light: "#0969da", Dark: "#58a6ff"}},
	"minimax-code": {Label: "minimax", Color: lipgloss.AdaptiveColor{Light: "#82071e", Dark: "#ff7b72"}},
	"openhands":    {Label: "openhands", Color: lipgloss.AdaptiveColor{Light: "#1a7f37", Dark: "#56d364"}},
	"antigravity":  {Label: "antigrav", Color: lipgloss.AdaptiveColor{Light: "#6639ba", Dark: "#a371f7"}},
}

// AgentLabel returns the short column label for an agent key.
func AgentLabel(key string) string {
	if style, ok := agentStyles[key]; ok {
		return style.Label
	}
	return key
}

// Agent renders an agent key in its identity colour, fitted to width. Fitting
// happens before styling so ANSI sequences never count toward the column, and
// it truncates as well as pads: an unstyled key longer than the column would
// otherwise push every column after it out of alignment.
func (t Theme) Agent(key string, width int) string {
	label := AgentLabel(key)
	if width > 0 {
		label = Fit(label, width, t.Glyphs.Ellipsis)
	}
	if !t.Color {
		return label
	}
	style, ok := agentStyles[key]
	if !ok {
		return t.Muted.Render(label)
	}
	return lipgloss.NewStyle().Foreground(style.Color).Render(label)
}

// AgentKeys returns the keys that carry an explicit identity, for tests that
// assert the table stays in step with the catalog.
func AgentKeys() []string {
	keys := make([]string, 0, len(agentStyles))
	for key := range agentStyles {
		keys = append(keys, key)
	}
	return keys
}
