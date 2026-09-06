package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// answeredExempt names the functions that reach the storage backend and do
// not classify the error themselves, each with the reason. A function that
// touches the backend and is not here has to call answered, because
// reporting "the endpoint gave no answer" about an answer, or a failed
// check about an outage, is the report's whole honesty in one word.
var answeredExempt = map[string]string{
	"fetch":         "returns the error to its caller unread; ciphertextStep is the caller and classifies it",
	"isolationStep": "keeps its own case list, because at the reference locker a refusal is the result rather than an error to report around: an answer it has no case for already lands on not-applicable, never on a pass",
}

// TestEveryStorageCallClassifiesWhatCameBack is the gate the prose class
// got and the code class did not.
//
// docs/hop.md's "only ciphertext" claim is held by a test that scans for
// the claim rather than listing the pages that make it, so a page written
// next year is covered. The same reasoning applies here and was not
// applied: the no-answer rule was carried to the two storage call sites
// the ticket named, and a third added later would have been caught by
// nothing. This walks the package instead, so the enumeration is the
// source rather than a list beside it.
//
// What it cannot see, said plainly: a call that reaches storage through a
// helper of another shape, or a backend method this list does not name.
func TestEveryStorageCallClassifiesWhatCameBack(t *testing.T) {
	backendMethods := map[string]bool{"List": true, "Get": true, "Put": true, "Delete": true}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var offenders []string
	calls := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			touches, classifies := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch f := call.Fun.(type) {
				case *ast.SelectorExpr:
					if backendMethods[f.Sel.Name] && len(call.Args) > 0 {
						// b.List(ctx, …) / o.Backend.Get(ctx, …): the first
						// argument of a backend call is the context.
						if id, ok := call.Args[0].(*ast.Ident); ok && strings.EqualFold(id.Name, "ctx") {
							touches = true
						}
					}
				case *ast.Ident:
					if f.Name == "answered" || f.Name == "fetch" {
						classifies = true
					}
				}
				return true
			})
			if !touches {
				continue
			}
			calls++
			if classifies {
				continue
			}
			if _, exempt := answeredExempt[fn.Name.Name]; exempt {
				continue
			}
			offenders = append(offenders, fn.Name.Name)
		}
	}
	if calls == 0 {
		t.Fatal("no storage call was found in package verify; this gate would prove nothing")
	}
	sort.Strings(offenders)
	for _, name := range offenders {
		t.Errorf("%s reaches the storage backend and neither classifies the error with answered nor appears in answeredExempt. "+
			"A step that reports an endpoint's answer as silence, or an outage as a failed check, is the one thing this report must not do; "+
			"if this call needs no classification, record that in answeredExempt with the reason.", name)
	}
	// The other direction: an exemption for a function that no longer
	// touches storage is a note about nothing, and it is how a list beside
	// the code drifts away from it.
	for name := range answeredExempt {
		found := false
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(src), "func "+name+"(") || strings.Contains(string(src), ") "+name+"(") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("answeredExempt names %s, which this package no longer defines", name)
		}
	}
}
