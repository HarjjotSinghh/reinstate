package secretscan

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Match is a detected secret span. The matched bytes are never retained.
type Match struct {
	Category Category
	Start    int    // byte offset in the input
	End      int    // exclusive byte offset
	Digest   string // sha256 of the matched bytes, first 12 hex chars
}

type candidate struct {
	category Category
	start    int
	end      int
	priority int
}

// Scan returns deterministic, non-overlapping matches in source order.
// Identical input always yields identical output.
func Scan(text string) []Match {
	cands := collectCandidates(text)
	selected := resolveOverlaps(cands)
	out := make([]Match, 0, len(selected))
	for _, c := range selected {
		out = append(out, Match{
			Category: c.category,
			Start:    c.start,
			End:      c.end,
			Digest:   digestOf(text[c.start:c.end]),
		})
	}
	return out
}

// Redact replaces every match with "[redacted:<category>:<digest>]" and returns
// the redacted text plus the applied matches. The marker cannot be mistaken for
// original transcript text and is stable under a second Redact call.
func Redact(text string) (string, []Match) {
	matches := Scan(text)
	if len(matches) == 0 {
		return text, matches
	}
	var b strings.Builder
	b.Grow(len(text) + len(matches)*24)
	prev := 0
	for _, m := range matches {
		b.WriteString(text[prev:m.Start])
		b.WriteString(marker(m))
		prev = m.End
	}
	b.WriteString(text[prev:])
	return b.String(), matches
}

// Summary aggregates counts per category without exposing any matched value.
func Summary(matches []Match) map[Category]int {
	out := make(map[Category]int, len(matches))
	for _, m := range matches {
		out[m.Category]++
	}
	return out
}

func marker(m Match) string {
	return fmt.Sprintf("[redacted:%s:%s]", m.Category, m.Digest)
}

func digestOf(matched string) string {
	sum := sha256.Sum256([]byte(matched))
	return hex.EncodeToString(sum[:])[:12]
}

func collectCandidates(text string) []candidate {
	var cands []candidate
	for _, p := range structuredPatterns {
		locs := p.re.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			cands = append(cands, candidate{
				category: p.category,
				start:    loc[0],
				end:      loc[1],
				priority: p.priority,
			})
		}
	}
	cands = append(cands, findHighEntropyCandidates(text)...)
	return cands
}

// resolveOverlaps selects a non-overlapping subset deterministically:
// earliest start, then longest span, then lowest priority number.
func resolveOverlaps(cands []candidate) []candidate {
	if len(cands) == 0 {
		return nil
	}
	sort.SliceStable(cands, func(i, j int) bool {
		a, b := cands[i], cands[j]
		if a.start != b.start {
			return a.start < b.start
		}
		alen, blen := a.end-a.start, b.end-b.start
		if alen != blen {
			return alen > blen
		}
		if a.priority != b.priority {
			return a.priority < b.priority
		}
		// Final tie-break on category string for total order.
		return a.category < b.category
	})
	out := make([]candidate, 0, len(cands))
	lastEnd := 0
	first := true
	for _, c := range cands {
		if !first && c.start < lastEnd {
			continue
		}
		out = append(out, c)
		lastEnd = c.end
		first = false
	}
	return out
}
