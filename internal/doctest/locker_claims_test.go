package doctest

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/HarjjotSinghh/reinstate/internal/cli"
)

// Two claims about the locker are made in more places than anyone can
// hold in their head: what the locker holds, and when this device's
// credential goes on the wire. Both have an exception. Every round of
// review so far has found the same failure — one surface states the
// exception and another states the claim absolutely — and fixing the
// surface that was reported has left the next one standing.
//
// This file is the gate that fixes the class rather than the surface. It
// does not know which files make the claims: it finds every place in the
// repository that makes one and requires the exception beside it, so a
// page, a help string or a comment written next year is held to the same
// rule the day it is added.
//
// The claims, stated once:
//
//   - The locker holds ciphertext, EXCEPT `keyring.v1.json`, which is
//     plaintext by design. Any sentence saying the locker holds "only
//     ciphertext" or "nothing but ciphertext" has to name that object.
//   - `rein sync verify` does not send the locker credential to a
//     plaintext `http` endpoint, EXCEPT a loopback address, where the
//     request does not leave the machine. Any sentence about the
//     plaintext refusal has to name that exemption.
type claim struct {
	// name is what the failure message calls this claim.
	name string
	// made matches a sentence that makes the claim.
	made *regexp.Regexp
	// exception matches the exception that sentence must carry.
	exception *regexp.Regexp
	// missing is the sentence the failure message ends with.
	missing string
}

var lockerClaims = []claim{
	{
		name:      "what the locker holds",
		made:      regexp.MustCompile(`(?i)(only|nothing but) ciphertext|every object[^.]{0,80}is ciphertext`),
		exception: regexp.MustCompile(`keyring\.v1\.json`),
		missing:   "name `keyring.v1.json`, the one object in the locker that is plaintext by design",
	},
	{
		name:      "when the locker credential is sent",
		made:      regexp.MustCompile("(?i)plaintext `?http|cleartext `?http|unencrypted connection|unencrypted `?http"),
		exception: regexp.MustCompile(`(?i)loopback`),
		missing:   "name the loopback exemption, the one plaintext endpoint the credential is sent to",
	},
}

// claimSurfaces are the directories and files whose prose is the
// product's word: shipped documentation, the website's copy of it, and
// the Go source that holds every help string and every sentence the
// verification report prints.
var claimSurfaces = []string{"docs", "internal", "cmd", "website/src", "README.md", "CHANGELOG.md", "PRODUCT.md", "ROADMAP.md", "AGENTS.md"}

// excludedFromClaims is the whole of what this gate does not read, and
// each entry is here for a reason that is not "it failed".
//
//   - references/ is third-party research about other products, quoted as
//     it was written. It is not this product's word about its own locker.
//   - docs/testing/ holds bench records: transcripts of what a run
//     printed, on a date, kept verbatim. A record that was rewritten to
//     satisfy a linter is not a record.
//
// A released CHANGELOG section is excluded the same way, in
// changelogUnreleased below: what shipped in 0.5.0 is history.
var excludedFromClaims = []string{"references", filepath.Join("docs", "testing"), "node_modules", "dist", ".git"}

// TestLockerClaimsCarryTheirExceptions reads every claim surface in the
// repository and fails on a sentence that makes one of the two claims
// without its exception.
func TestLockerClaimsCarryTheirExceptions(t *testing.T) {
	root := repoRoot(t)
	files := claimFiles(t, root)
	if len(files) < 40 {
		t.Fatalf("only %d claim surfaces found; the walk is not reaching the documentation", len(files))
	}
	checked := 0
	for _, path := range files {
		rel, err := filepath.Rel(root, path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(body), "\r\n", "\n")
		if filepath.Base(path) == "CHANGELOG.md" {
			text = changelogUnreleased(text)
		}
		for _, paragraph := range strings.Split(text, "\n\n") {
			for _, c := range lockerClaims {
				if !c.made.MatchString(paragraph) {
					continue
				}
				checked++
				if c.exception.MatchString(paragraph) {
					continue
				}
				t.Errorf("%s makes the %q claim without its exception; the sentence has to %s:\n\n%s",
					rel, c.name, c.missing, strings.TrimSpace(paragraph))
			}
		}
	}
	// A gate that matches nothing passes for the wrong reason. Both claims
	// are made in several places and must go on being found.
	if checked < 6 {
		t.Fatalf("the gate found only %d claim sentence(s); either the claims moved or the patterns stopped matching", checked)
	}
}

// TestLockerClaimsInCommandHelp holds the rendered help text to the same
// rule, whatever it is assembled from. A Short or Long built up from
// constants would satisfy the file-level walk above and still print a
// bare claim to a person running `rein sync verify --help`.
//
// It reads Short, Long, Example and every flag's usage string, which is
// the whole of what `--help` prints. The earlier version read Short and
// Long only, so a claim in a flag description or an example was help text
// this gate did not see.
func TestLockerClaimsInCommandHelp(t *testing.T) {
	root := cli.NewRoot(cli.Options{
		Name:            "rein",
		TerminalChecker: func(io.Reader, io.Writer) bool { return false },
	})
	var walk func(cmd *cobra.Command, path string)
	walk = func(cmd *cobra.Command, path string) {
		texts := []string{cmd.Short, cmd.Long, cmd.Example}
		cmd.Flags().VisitAll(func(f *pflag.Flag) { texts = append(texts, f.Usage) })
		for _, text := range texts {
			for _, paragraph := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n\n") {
				for _, c := range lockerClaims {
					if c.made.MatchString(paragraph) && !c.exception.MatchString(paragraph) {
						t.Errorf("rein %s help makes the %q claim without its exception; it has to %s:\n\n%s",
							path, c.name, c.missing, strings.TrimSpace(paragraph))
					}
				}
			}
		}
		for _, child := range cmd.Commands() {
			walk(child, strings.TrimSpace(path+" "+child.Name()))
		}
	}
	walk(root, "")
}

// TestTheLockerClaimsAreStillTold is the other half of the gate. Every
// rule above is satisfied by deleting the sentence, and the disclosure
// these pages exist to make would go with it, so the surfaces a person
// actually reaches for have to go on making both statements.
func TestTheLockerClaimsAreStillTold(t *testing.T) {
	for _, tt := range []struct {
		page string
		want []string
	}{
		{"docs/hop.md", []string{"keyring.v1.json"}},
		{"docs/cli-reference.md", []string{"keyring.v1.json", "loopback"}},
		{"docs/hop/object-format.md", []string{"keyring.v1.json", "loopback"}},
		{"docs/hop/threat-model.md", []string{"keyring.v1.json", "loopback"}},
	} {
		page := read(t, tt.page)
		for _, want := range tt.want {
			if !strings.Contains(page, want) {
				t.Errorf("%s no longer mentions %q: the claim may have been deleted rather than qualified", tt.page, want)
			}
		}
	}
}

// claimFiles is every Markdown and Go file under the claim surfaces,
// minus test files (which quote the prose they check) and the exclusions
// above.
func claimFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	for _, surface := range claimSurfaces {
		path := filepath.Join(root, surface)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("claim surface %s is not there: %v", surface, err)
		}
		if !info.IsDir() {
			out = append(out, path)
			continue
		}
		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				for _, skip := range excludedFromClaims {
					if strings.HasSuffix(p, skip) {
						return fs.SkipDir
					}
				}
				return nil
			}
			switch {
			case strings.HasSuffix(p, "_test.go"), strings.HasSuffix(p, ".test.ts"):
			// The website's copy is not all in content/: a landing-page
			// figcaption in an .astro component shipped "stores only
			// ciphertext in your bucket" for a month while this gate read
			// only content/, and a .ts test file pinned the string. The
			// extensions are named here rather than left implicit, because
			// the list is what the gate can see.
			case strings.HasSuffix(p, ".go"), strings.HasSuffix(p, ".md"), strings.HasSuffix(p, ".mdx"),
				strings.HasSuffix(p, ".astro"), strings.HasSuffix(p, ".ts"), strings.HasSuffix(p, ".tsx"):
				out = append(out, p)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// changelogUnreleased is the part of the CHANGELOG that is still a claim
// about the product rather than a record of what shipped. Released
// sections are history and are not edited.
func changelogUnreleased(text string) string {
	_, rest, found := strings.Cut(text, "## [Unreleased]")
	if !found {
		return text
	}
	if end := regexp.MustCompile(`\n## \[\d`).FindStringIndex(rest); end != nil {
		return rest[:end[0]]
	}
	return rest
}
