package capability

import (
	"context"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type collector struct {
	ctx           context.Context
	items         []Item
	diagnostics   []Diagnostic
	counts        map[string]int
	limited       map[string]bool
	diagnosticSet map[Diagnostic]struct{}
}

func newCollector() *collector {
	return newCollectorContext(context.Background())
}

func newCollectorContext(ctx context.Context) *collector {
	if ctx == nil {
		ctx = context.Background()
	}
	return &collector{
		ctx:           ctx,
		counts:        make(map[string]int),
		limited:       make(map[string]bool),
		diagnosticSet: make(map[Diagnostic]struct{}),
	}
}

func (c *collector) cancelled() bool {
	return c != nil && c.ctx != nil && c.ctx.Err() != nil
}

func (c *collector) add(item Item) {
	if c.cancelled() {
		return
	}
	item.Name = sanitizeName(item.Name)
	if item.Name == "" {
		return
	}
	key := string(item.Agent) + "\x00" + string(item.Kind)
	if c.counts[key] >= maxEntries {
		if !c.limited[key] {
			c.addDiagnostic(Diagnostic{Agent: item.Agent, Kind: item.Kind, Scope: item.Scope, Code: DiagnosticLimitReached})
			c.limited[key] = true
		}
		return
	}
	c.counts[key]++
	c.items = append(c.items, item)
}

func (c *collector) addDiagnostic(d Diagnostic) {
	if c.cancelled() {
		return
	}
	if _, exists := c.diagnosticSet[d]; exists {
		return
	}
	c.diagnosticSet[d] = struct{}{}
	c.diagnostics = append(c.diagnostics, d)
}

func (c *collector) full(agent Agent, kind Kind) bool {
	return c.cancelled() || c.counts[string(agent)+"\x00"+string(kind)] >= maxEntries
}

func (c *collector) inventory() Inventory {
	if c.cancelled() {
		return cancelledInventory()
	}
	sort.Slice(c.items, func(i, j int) bool {
		a, b := c.items[i], c.items[j]
		if a.Agent != b.Agent {
			return a.Agent < b.Agent
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
		if an != bn {
			return an < bn
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.State != b.State {
			return a.State < b.State
		}
		if a.SourceKind != b.SourceKind {
			return a.SourceKind < b.SourceKind
		}
		if a.Transport != b.Transport {
			return a.Transport < b.Transport
		}
		return !a.Lazy && b.Lazy
	})
	if c.cancelled() {
		return cancelledInventory()
	}
	c.items = dedupeItems(c.items)
	if c.cancelled() {
		return cancelledInventory()
	}

	sort.Slice(c.diagnostics, func(i, j int) bool {
		a, b := c.diagnostics[i], c.diagnostics[j]
		if a.Agent != b.Agent {
			return a.Agent < b.Agent
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Code < b.Code
	})
	if c.cancelled() {
		return cancelledInventory()
	}
	c.diagnostics = dedupeDiagnostics(c.diagnostics)
	if c.cancelled() {
		return cancelledInventory()
	}
	if c.items == nil {
		c.items = []Item{}
	}
	if c.diagnostics == nil {
		c.diagnostics = []Diagnostic{}
	}

	return Inventory{Items: c.items, Diagnostics: c.diagnostics}
}

func cancelledInventory() Inventory {
	return Inventory{
		Items:       []Item{},
		Diagnostics: []Diagnostic{{Code: DiagnosticCancelled}},
	}
}

func dedupeItems(items []Item) []Item {
	out := items[:0]
	var prior string
	for _, item := range items {
		key := strings.Join([]string{
			string(item.Agent), string(item.Kind), string(item.Scope),
			strings.ToLower(item.Name), string(item.State),
			string(item.SourceKind), string(item.Transport), boolString(item.Lazy),
		}, "\x00")
		if len(out) != 0 && key == prior {
			continue
		}
		out = append(out, item)
		prior = key
	}
	return out
}

func dedupeDiagnostics(diagnostics []Diagnostic) []Diagnostic {
	out := diagnostics[:0]
	var prior Diagnostic
	for _, d := range diagnostics {
		if len(out) != 0 && d == prior {
			continue
		}
		out = append(out, d)
		prior = d
	}
	return out
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func sanitizeName(raw string) string {
	raw = stripTerminalSequences(raw)
	runes := make([]rune, 0, min(utf8.RuneCountInString(raw), maxNameRunes))
	for _, r := range raw {
		if len(runes) >= maxNameRunes {
			break
		}
		if unicode.IsControl(r) || isBidiControl(r) {
			continue
		}
		if r == '/' || r == '\\' {
			r = '_'
		}
		runes = append(runes, r)
	}
	return strings.TrimSpace(string(runes))
}

func isBidiControl(r rune) bool {
	switch r {
	case '\u061c', '\u200e', '\u200f',
		'\u202a', '\u202b', '\u202c', '\u202d', '\u202e',
		'\u2066', '\u2067', '\u2068', '\u2069':
		return true
	default:
		return false
	}
}

func stripTerminalSequences(raw string) string {
	var out strings.Builder
	for i := 0; i < len(raw); {
		if raw[i] != 0x1b {
			r, size := utf8.DecodeRuneInString(raw[i:])
			out.WriteRune(r)
			i += size
			continue
		}
		i++
		if i >= len(raw) {
			break
		}
		switch raw[i] {
		case '[': // CSI: consume through the final byte.
			i++
			for i < len(raw) {
				b := raw[i]
				i++
				if b >= 0x40 && b <= 0x7e {
					break
				}
			}
		case ']': // OSC: consume through BEL or ST.
			i++
			for i < len(raw) {
				if raw[i] == 0x07 {
					i++
					break
				}
				if raw[i] == 0x1b && i+1 < len(raw) && raw[i+1] == '\\' {
					i += 2
					break
				}
				i++
			}
		default:
			// A two-byte escape sequence has one byte after ESC.
			i++
		}
	}
	return out.String()
}
