package doctest

import (
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/hop"
)

// A sign-in refusal is a sentence a person reads in a browser and a code a
// script reads on the wire. The code is public protocol: `docs/hop.md`
// carries the table, and a client that does not speak Go reads that table
// and nothing else.
//
// This gate keeps the table and the client in step in both directions. A
// code declared in `internal/hop` and left out of the docs is a wire value
// nobody can look up; a row in the docs for a code no build declares is a
// promise about an answer that never arrives. Either way the person on the
// other end is reading something that is not true, which is the failure
// this whole ticket is about.
//
// It does not check the sentences. Those are written by the control plane
// and printed verbatim; the docs describe when each code fires, which is
// prose no test can hold to account.
func TestEverySignInRefusalCodeIsDocumented(t *testing.T) {
	const marker = "<!-- sign-in refusal codes: gated by TestEverySignInRefusalCodeIsDocumented -->"
	doc := read(t, "docs/hop.md")
	table, ok := refusalCodeTable(doc, marker)
	if !ok {
		t.Fatalf("docs/hop.md has no refusal-code table marked %q", marker)
	}

	documented := map[string]bool{}
	for _, row := range table {
		code, ok := refusalCodeCell(row)
		if !ok {
			continue
		}
		documented[code] = true
	}
	if len(documented) == 0 {
		t.Fatal("the marked table in docs/hop.md documents no refusal codes; the gate would prove nothing")
	}

	declared := map[string]bool{}
	for _, code := range hop.SignInRefusalCodes() {
		declared[code] = true
		if !documented[code] {
			t.Errorf("hop.%s is a refusal a client can receive and docs/hop.md does not document it", code)
		}
	}
	for code := range documented {
		if !declared[code] {
			t.Errorf("docs/hop.md documents refusal code %q, which internal/hop does not declare", code)
		}
	}

	// The two rules a client has to follow are stated where the table is,
	// not left to be inferred from it.
	for _, want := range []string{
		"switches on `code` and never on the status",
		"terminal",
	} {
		if !strings.Contains(strings.ToLower(sectionAfter(doc, marker)), strings.ToLower(want)) {
			t.Errorf("docs/hop.md's refusal section does not state %q", want)
		}
	}
}

// refusalCodeTable returns the markdown table rows that follow marker, up
// to the first blank line after the table starts.
func refusalCodeTable(doc, marker string) ([]string, bool) {
	index := strings.Index(doc, marker)
	if index < 0 {
		return nil, false
	}
	var rows []string
	started := false
	for _, line := range strings.Split(doc[index+len(marker):], "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			if started {
				break
			}
			continue
		}
		started = true
		rows = append(rows, trimmed)
	}
	return rows, started
}

// refusalCodeCell reads the `code` column of one table row: the second
// cell, backticked, on a row whose first cell is a backticked HTTP status.
// That last part is what tells a data row from the header, whose own second
// cell is the backticked word "code".
func refusalCodeCell(row string) (string, bool) {
	cells := strings.Split(strings.Trim(row, "|"), "|")
	if len(cells) < 2 {
		return "", false
	}
	status := strings.Trim(strings.TrimSpace(cells[0]), "`")
	if len(status) != 3 || strings.Trim(status, "0123456789") != "" {
		return "", false
	}
	cell := strings.TrimSpace(cells[1])
	if !strings.HasPrefix(cell, "`") || !strings.HasSuffix(cell, "`") {
		return "", false
	}
	return strings.Trim(cell, "`"), true
}

// sectionAfter is the documentation from marker to the next heading, which
// is where the rules that go with the table have to live.
func sectionAfter(doc, marker string) string {
	index := strings.Index(doc, marker)
	if index < 0 {
		return ""
	}
	rest := doc[index+len(marker):]
	if next := strings.Index(rest, "\n## "); next >= 0 {
		return rest[:next]
	}
	return rest
}
