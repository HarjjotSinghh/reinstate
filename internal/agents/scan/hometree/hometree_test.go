package hometree

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/sessionindex"
)

func TestResolveRootPrefersExplicitThenEnvThenFirstExistingCandidate(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	first := filepath.Join(home, ".agent")
	second := filepath.Join(home, ".config", "agent")
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	override := filepath.Join(home, "override")

	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "explicit wins over env and candidates",
			cfg: Config{
				Explicit:   override,
				RootEnv:    "AGENT_HOME",
				LookupEnv:  func(string) string { return first },
				Candidates: []string{first, second},
			},
			want: override,
		},
		{
			name: "env override wins over candidates",
			cfg: Config{
				RootEnv:    "AGENT_HOME",
				LookupEnv:  func(string) string { return override },
				Candidates: []string{first, second},
			},
			want: override,
		},
		{
			name: "first existing candidate",
			cfg: Config{
				RootEnv:    "AGENT_HOME",
				LookupEnv:  func(string) string { return "" },
				Candidates: []string{filepath.Join(home, "missing"), first, second},
			},
			want: first,
		},
		{
			name: "nothing present",
			cfg: Config{
				Candidates: []string{filepath.Join(home, "missing")},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ResolveRoot(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == "" {
				if got != "" {
					t.Fatalf("ResolveRoot() = %q, want empty", got)
				}
				return
			}
			want, err := filepath.Abs(filepath.Clean(tt.want))
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("ResolveRoot() = %q, want %q", got, want)
			}
		})
	}
}

func TestDiscoverWalksGlobSkipsExcludedAndIgnoresMissingMarker(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	keep := filepath.Join(root, "sessions", "proj", "one.jsonl")
	skip := filepath.Join(root, "sessions", "subagents", "hidden.jsonl")
	other := filepath.Join(root, "sessions", "proj", "notes.txt")
	writeFile(t, keep, "{\"id\":\"1\"}\n")
	writeFile(t, skip, "{\"id\":\"hidden\"}\n")
	writeFile(t, other, "not jsonl\n")

	files, err := Walk(context.Background(), root, "sessions/**/*.jsonl", []string{"subagents"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != keep {
		t.Fatalf("Walk() = %#v, want only %q", files, keep)
	}

	_, missing, err := Discover(context.Background(), Config{
		Explicit:    root,
		Marker:      "missing-marker",
		SessionGlob: "sessions/**/*.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if missing != nil {
		t.Fatalf("missing marker returned files %#v", missing)
	}

	discoveredRoot, found, err := Discover(context.Background(), Config{
		Explicit:    root,
		Marker:      "sessions",
		SessionGlob: "sessions/**/*.jsonl",
		Excluded:    []string{"subagents"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if discoveredRoot != wantRoot {
		t.Fatalf("root = %q, want %q", discoveredRoot, wantRoot)
	}
	if len(found) != 1 || found[0].Path != keep {
		t.Fatalf("Discover() = %#v", found)
	}
}

func TestWalkIsDeterministicAndEmptyGlobFindsNothing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sessions", "b.jsonl"), "{}\n")
	writeFile(t, filepath.Join(root, "sessions", "a.jsonl"), "{}\n")

	first, err := Walk(context.Background(), root, "sessions/*.jsonl", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Walk(context.Background(), root, "sessions/*.jsonl", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].Path != second[0].Path || first[1].Path != second[1].Path {
		t.Fatalf("walks were not deterministic: %#v vs %#v", first, second)
	}
	if first[0].Path >= first[1].Path {
		t.Fatalf("files were not sorted: %#v", first)
	}
	none, err := Walk(context.Background(), root, "", nil)
	if err != nil || len(none) != 0 {
		t.Fatalf("empty glob = %#v, %v", none, err)
	}
}

func TestReadJSONLStopsAtLastCompleteLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	payload := "{\"id\":\"complete\"}\n{\"id\":\"partial\""
	writeFile(t, path, payload)

	var ids []string
	warnings, err := ReadJSONL(path, func(_ int, line []byte) error {
		ids = append(ids, string(bytes.TrimSpace(line)))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || !strings.Contains(ids[0], `"complete"`) {
		t.Fatalf("visited %v", ids)
	}
	if !hasWarningCode(warnings, "incomplete_trailing_record") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestReadJSONLBoundsOversizedLine(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "huge.jsonl")
	// One complete tiny record, then a line larger than the shared ceiling.
	var body strings.Builder
	body.WriteString("{\"ok\":true}\n")
	body.WriteString("{\"x\":\"")
	body.WriteString(strings.Repeat("a", MaxJSONLineBytes+8))
	body.WriteString("\"}\n")
	writeFile(t, path, body.String())

	var count int
	warnings, err := ReadJSONL(path, func(_ int, _ []byte) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("visited %d records, want 1", count)
	}
	if !hasWarningCode(warnings, "oversized_record") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestStampChangeDetectionUsesModTimeAndSize(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeFile(t, path, "{\"id\":\"1\"}\n")
	first, err := Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if Changed(first, first) {
		t.Fatal("identical stamps reported as changed")
	}

	time.Sleep(10 * time.Millisecond)
	writeFile(t, path, "{\"id\":\"1\"}\n{\"id\":\"2\"}\n")
	second, err := Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !Changed(second, first) {
		t.Fatalf("size/mtime change was not detected: %+v -> %+v", first, second)
	}
	if second.Size == first.Size && second.ModTime == first.ModTime {
		t.Fatal("rewritten file kept both size and mtime")
	}
}

func TestHasMarkerAndAbsentRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, err := HasMarker(root, "projects")
	if err != nil || !ok {
		t.Fatalf("HasMarker(projects) = %t, %v", ok, err)
	}
	ok, err = HasMarker(root, "sessions")
	if err != nil || ok {
		t.Fatalf("HasMarker(sessions) = %t, %v", ok, err)
	}
	ok, err = HasMarker("", "projects")
	if err != nil || ok {
		t.Fatalf("HasMarker(empty root) = %t, %v", ok, err)
	}

	root, files, err := Discover(context.Background(), Config{
		Explicit:    filepath.Join(t.TempDir(), "missing"),
		Marker:      "sessions",
		SessionGlob: "sessions/**/*.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	if files != nil {
		t.Fatalf("absent root returned files %#v (root=%q)", files, root)
	}
}

func TestWalkHonorsCanceledContext(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sessions", "a.jsonl"), "{}\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Walk(ctx, root, "sessions/*.jsonl", nil)
	if err == nil {
		t.Fatal("canceled walk succeeded")
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasWarningCode(warnings []sessionindex.Warning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}
