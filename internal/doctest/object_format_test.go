package doctest

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/keyring"
	"github.com/HarjjotSinghh/reinstate/internal/schema"
)

// schemaVersionPlaceholder is what docs/hop/object-format.md prints where
// a format version goes. The page is the published protocol, so it must
// not carry a version number of its own: the number belongs to the Go
// package that owns the document, this test substitutes it, and a version
// bump therefore cannot leave a stale number on a published page.
const schemaVersionPlaceholder = "<schema version>"

// TestObjectFormatExamplesMatchTheCode holds every JSON example in
// docs/hop/object-format.md to the struct the code decodes that object
// into: the field names, the format version, and the constants the page
// names.
//
// It decodes with DisallowUnknownFields, which fails on a field the page
// invents or one the code has renamed, and passes when the code grows a
// field the page's example does not show — a page may be less than the
// format, and must never be other than it.
func TestObjectFormatExamplesMatchTheCode(t *testing.T) {
	page := read(t, "docs/hop/object-format.md")
	tests := []struct {
		name    string
		heading string
		version int
		into    any
		check   func(t *testing.T, raw string)
	}{
		{
			name:    "index",
			heading: "### `manifest.age` → the index",
			version: schema.ManifestSchemaVersion,
			into:    &schema.Manifest{},
		},
		{
			name:    "snapshot envelope",
			heading: "### `snapshots/<uuid>.age` → one session",
			version: schema.EnvelopeSchemaVersion,
			into:    &schema.Envelope{},
			check: func(t *testing.T, raw string) {
				var env schema.Envelope
				if err := json.Unmarshal([]byte(raw), &env); err != nil {
					t.Fatal(err)
				}
				if env.Kind != schema.EnvelopeKind {
					t.Errorf("the snapshot envelope example names kind %q, want %q", env.Kind, schema.EnvelopeKind)
				}
			},
		},
		{
			name:    "keyring",
			heading: "### `keyring.v1.json` → the wrapped root key",
			version: keyring.SchemaVersion,
			into:    &keyring.Keyring{},
			check: func(t *testing.T, raw string) {
				var k keyring.Keyring
				if err := json.Unmarshal([]byte(raw), &k); err != nil {
					t.Fatal(err)
				}
				// current_generation names one of the generations listed:
				// the rule every reader of this object applies.
				found := false
				for _, g := range k.Generations {
					found = found || g.Number == k.CurrentGeneration
				}
				if !found {
					t.Errorf("the keyring example's current_generation %d names no generation in the example", k.CurrentGeneration)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw := firstJSONBlock(t, page, tc.heading)
			if !strings.Contains(raw, `"schema_version": `+schemaVersionPlaceholder) {
				t.Fatalf("the %s example hardcodes a schema version instead of %s:\n%s", tc.name, schemaVersionPlaceholder, raw)
			}
			// The placeholder is the only substitution; everything else on
			// the page is parsed exactly as a reader sees it.
			filled := strings.ReplaceAll(raw, schemaVersionPlaceholder, strconv.Itoa(tc.version))
			decoder := json.NewDecoder(strings.NewReader(filled))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(tc.into); err != nil {
				t.Fatalf("the %s example does not match the struct the code decodes it into: %v\n%s", tc.name, err, filled)
			}
			if tc.check != nil {
				tc.check(t, filled)
			}
		})
	}
	// The object's name in storage is a constant too, and it does not
	// change with the schema version inside it.
	if !strings.Contains(page, keyring.ObjectName) {
		t.Errorf("docs/hop/object-format.md never names the keyring object %s", keyring.ObjectName)
	}
}

// firstJSONBlock is the first fenced JSON block under heading.
func firstJSONBlock(t *testing.T, page, heading string) string {
	t.Helper()
	page = strings.ReplaceAll(page, "\r\n", "\n")
	_, section, found := strings.Cut(page, heading)
	if !found {
		t.Fatalf("docs/hop/object-format.md has no %q section", heading)
	}
	_, after, found := strings.Cut(section, "```json\n")
	if !found {
		t.Fatalf("%q holds no JSON example", heading)
	}
	block, _, found := strings.Cut(after, "```")
	if !found {
		t.Fatalf("%q holds an unterminated JSON example", heading)
	}
	return block
}
