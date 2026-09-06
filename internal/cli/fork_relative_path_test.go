package cli

import "testing"

// TestForkRelativePath pins the fork naming contract: a fork keeps its source's
// extension (so an embedded-store agent addressed as sessions/<id>.json is not
// relabelled .jsonl) and lives beside its source.
func TestForkRelativePath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		fork   string
		want   string
	}{
		{"jsonl beside source", "projects/-Users-me-demo/abc.jsonl", "fork1", "projects/-Users-me-demo/fork1.jsonl"},
		{"json keeps json", "sessions/ses_abc.json", "ses_fork", "sessions/ses_fork.json"},
		{"extensionless defaults to jsonl", "sessions/abc", "fork1", "sessions/fork1.jsonl"},
		{"top-level source", "abc.jsonl", "fork1", "fork1.jsonl"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := forkRelativePath(tc.source, tc.fork); got != tc.want {
				t.Fatalf("forkRelativePath(%q,%q)=%q want %q", tc.source, tc.fork, got, tc.want)
			}
		})
	}
}
