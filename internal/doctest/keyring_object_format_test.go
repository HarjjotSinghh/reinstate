package doctest

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/keyring"
)

// TestKeyringExampleShowsEveryFieldTheFormatHas is the direction
// TestObjectFormatExamplesMatchTheCode deliberately does not check.
//
// That test decodes with DisallowUnknownFields, which catches a page that
// invents a field or keeps a renamed one, and passes a page that shows
// less than the format. Less is normally the right latitude for a document.
// It is the wrong latitude for this one object, because the fields most
// worth omitting are the ones that carry the format's security properties,
// and omitting them is exactly what happened: `keyring.v1.json` was
// documented on the branch that added `rein sync verify`, at a time when
// the keyring had no `account_key`, no per-generation `signature` and no
// `revoked` records, and the page merged with the branch that added all
// three without a conflict marker and without failing a test. A reader of
// the published protocol would have found no authenticator described on an
// object whose whole job is to be authenticated.
//
// So for the keyring, and only for the keyring, the example has to be
// complete: every JSON field name the Go types define must appear in it.
// The check is against the types themselves rather than a fixture, so a
// ticket that adds a field to the keyring cannot land without either
// showing it here or deciding, in the open, not to.
func TestKeyringExampleShowsEveryFieldTheFormatHas(t *testing.T) {
	page := read(t, "docs/hop/object-format.md")
	raw := firstJSONBlock(t, page, "### `keyring.v1.json` → the wrapped root key")
	filled := strings.ReplaceAll(raw, schemaVersionPlaceholder, strconv.Itoa(keyring.SchemaVersion))
	var doc any
	if err := json.Unmarshal([]byte(filled), &doc); err != nil {
		t.Fatalf("the keyring example is not JSON: %v\n%s", err, filled)
	}
	shown := map[string]bool{}
	collectJSONKeys(doc, shown)
	defined := map[string]bool{}
	collectStructJSONNames(reflect.TypeOf(keyring.Keyring{}), defined, map[reflect.Type]bool{})

	var missing []string
	for name := range defined {
		if !shown[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("the published keyring example does not show %v; the format defines them, so a reader of docs/hop/object-format.md cannot see them", missing)
	}

	// "appears somewhere" is not enough for the one field that has to be on
	// every generation. An example showing two generations satisfies the
	// check above with a signature on only the second, and a reader would
	// then have the first generation described as unsigned -- which is
	// precisely the thing this object's security rests on not being true.
	doc2, ok := doc.(map[string]any)
	if !ok {
		t.Fatalf("the keyring example is not a JSON object: %T", doc)
	}
	generations, ok := doc2["generations"].([]any)
	if !ok || len(generations) == 0 {
		t.Fatalf("the keyring example shows no generations: %v", doc2["generations"])
	}
	for i, raw := range generations {
		g, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("generation %d in the example is not an object: %T", i, raw)
		}
		if _, has := g["signature"]; !has {
			t.Errorf("generation %v in the published example carries no signature; every generation carries one, the first included, and a keyring holding an unsigned generation is refused whole", g["number"])
		}
	}
}

// collectJSONKeys gathers every object key anywhere in a decoded document.
func collectJSONKeys(v any, into map[string]bool) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			into[k] = true
			collectJSONKeys(child, into)
		}
	case []any:
		for _, child := range t {
			collectJSONKeys(child, into)
		}
	}
}

// collectStructJSONNames gathers every json field name a struct tree
// defines. seen guards against a type that refers to itself; the keyring
// types do not today, and the guard costs nothing if one ever does.
func collectStructJSONNames(t reflect.Type, into map[string]bool, seen map[reflect.Type]bool) {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || seen[t] {
		return
	}
	seen[t] = true
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		into[name] = true
		collectStructJSONNames(f.Type, into, seen)
	}
}
