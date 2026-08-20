package probe

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/HarjjotSinghh/reinstate/internal/agents"
)

func TestEmptyHomeProducesCompleteArtifact(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	desc := syntheticDescriptor(home)
	art, err := Collect(context.Background(), agents.Env{
		Home:      home,
		LookupEnv: func(string) string { return "" },
	}, []agents.Descriptor{desc}, Options{
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version:  "0.5.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(art); err != nil {
		t.Fatal(err)
	}
	if len(art.Agents) != 1 {
		t.Fatalf("agents = %d", len(art.Agents))
	}
	got := art.Agents[0]
	if got.ResolvedRoot != nil {
		t.Fatalf("resolved_root = %+v", got.ResolvedRoot)
	}
	if len(got.Tree) != 0 || len(got.NameShapes) != 0 || got.ExecutableOnPath || got.RootEnvSet {
		t.Fatalf("empty probe not empty-but-complete: %+v", got)
	}
	if got.CandidateRoots == nil || got.FirstLineKeys == nil {
		t.Fatal("arrays/objects must be present")
	}
}

func TestT0DescriptorEmitsEmptyCandidateRoots(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	desc := agents.Descriptor{
		Key:         "amp",
		DisplayName: "Amp",
		Vendor:      "Sourcegraph",
		Tier:        agents.TierKnown,
		Family:      agents.FamilyRemote,
		T0Reason:    agents.T0LayoutUnverified,
	}
	art, err := Collect(context.Background(), agents.Env{
		Home:      home,
		LookupEnv: func(string) string { return "" },
	}, []agents.Descriptor{desc}, Options{
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version:  "0.5.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(art); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(art.Agents[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"candidate_roots":[]`) {
		t.Fatalf("candidate_roots not empty array: %s", raw)
	}
	if art.Agents[0].CandidateRoots == nil {
		t.Fatal("candidate_roots is nil")
	}
}

func TestPlantedSecretsNeverAppear(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, ".planted-agent")
	sessionDir := filepath.Join(root, "projects", "-Users-alice-code-secret-repo")
	if err := os.MkdirAll(filepath.Join(root, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.json"), []byte(`{"token":"sk-plantedsecretvalue"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cache", "secret.txt"), []byte("alice-password"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := `{"prompt":"hello alice from /Users/alice/code/secret-repo","api_key":"sk-plantedsecretvalue","repo":"secret-repo"}` + "\n"
	if err := os.WriteFile(filepath.Join(sessionDir, "01987654-3210-7890-abcd-ef0123456789.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	desc := syntheticDescriptor(home)
	desc.Storage.Excluded = []string{"auth.json", "cache", "**/auth.json"}
	art, err := Collect(context.Background(), agents.Env{
		Home: home,
		LookupEnv: func(key string) string {
			if key == "PLANTED_HOME" {
				return "/Users/alice/.planted-agent"
			}
			return ""
		},
	}, []agents.Descriptor{desc}, Options{
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version:  "0.5.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(art); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(art)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"alice",
		"/Users/alice",
		"sk-plantedsecretvalue",
		"hello alice",
		"secret-repo",
		"alice-password",
		home,
	}
	if hits := containsForbidden(raw, forbidden); len(hits) > 0 {
		t.Fatalf("probe leaked %v\n%s", hits, raw)
	}
	if strings.Contains(string(raw), "prompt") && strings.Contains(string(raw), "hello") {
		t.Fatal("JSON values from the session file appeared")
	}
	rec := art.Agents[0]
	if rec.ResolvedRoot == nil || rec.ResolvedRoot.RelativeTo != "home" || rec.ResolvedRoot.Suffix != ".planted-agent" {
		t.Fatalf("resolved_root = %+v", rec.ResolvedRoot)
	}
	if !rec.RootEnvSet {
		t.Fatal("root_env_set should be true when the override is present")
	}
	foundKeys := false
	for path, keys := range rec.FirstLineKeys {
		if strings.Contains(path, "projects") {
			foundKeys = true
			joined := strings.Join(keys, ",")
			if !strings.Contains(joined, "prompt") || strings.Contains(joined, "alice") {
				t.Fatalf("first_line_keys = %v", keys)
			}
		}
	}
	if !foundKeys {
		t.Fatalf("first_line_keys = %#v", rec.FirstLineKeys)
	}
}

func TestHugeTreeFinishes(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, ".planted-agent", "projects", "bucket")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3000; i++ {
		name := filepath.Join(root, "f-"+strings.Repeat("x", 8)+"-"+itoa(i)+".jsonl")
		if err := os.WriteFile(name, []byte(`{"n":1}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	art, err := Collect(ctx, agents.Env{Home: home, LookupEnv: func(string) string { return "" }},
		[]agents.Descriptor{syntheticDescriptor(home)}, Options{
			LookPath: func(string) (string, error) { return "", os.ErrNotExist },
			MaxFiles: 200,
			MaxOpens: 8,
		})
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("probe exceeded bound")
	}
	if err := Validate(art); err != nil {
		t.Fatal(err)
	}
}

func TestShapeNormalization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"01987654-3210-7890-abcd-ef0123456789", "<uuid-v4>"},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "<32-hex>"},
		{"wd_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "wd_<32-hex>"},
		{"-Users-alice-code-demo", "<slug>"},
		{"%2FUsers%2Falice%2Fcode", "<slug>"},
		{"42", "<n>"},
		{"session-001", "session-<n>"},
		{"sessions", "sessions"},
		{"state.json", "state.json"},
		// Cursor buckets projects as an absolute path with separators
		// rewritten. Every character is unremarkable, so without an explicit
		// rule the normalizer passes the home path and the repository name
		// through intact.
		{"Users-alice-Documents-Projects-demo", "<path-slug>"},
		{"var-folders-jv-85d89wh91t5132jys95scwc00000gn-T", "<path-slug>"},
		{"tmp-reinstate-argv-fix", "<path-slug>"},
		{"C-Users-alice-src-demo", "<path-slug>"},
		// A vendor prefix that merely contains segments is not a path.
		{"empty-window", "<slug>"},
		{"project-notes.json", "<slug>.json"},
		{"01987654-3210-7890-abcd-ef0123456789.runtime.json", "<uuid-v4>-runtime.json"},
		// Kimi Code's workspace bucket. The stem is the basename of the
		// working directory, so it is a repository name, and a native Windows
		// probe emitted wd_portfolio-25_6d65015f0cb0 before this rule existed.
		{"wd_portfolio-25_6d65015f0cb0", "wd_<project>_<12-hex>"},
		{"wd_probe-one_87e15ce98f3b", "wd_<project>_<12-hex>"},
		// The macOS bucket only looked like a username because that session
		// ran in the home directory, whose basename is the account name.
		{"wd_harjjotsinghh_f6c3da451c53", "wd_<project>_<12-hex>"},
		{"wd_my_project_abcdef1234567890", "wd_<project>_<16-hex>"},
		// Too short a tail to be a content hash.
		{"wd_alice_abcdef", "wd_alice_abcdef"},
		// Git object names under a marketplace checkout. The trailing-digits
		// rule used to split the hash and leave most of it verbatim.
		{"pack-8c7ffa580563b675b1fd27a53df219b761e4d0a1", "pack-<40-hex>"},
		{"pack-8c7ffa580563b675b1fd27a53df219b761e4d0a1.idx", "pack-<40-hex>.idx"},
	}
	for _, tt := range tests {
		if got := normalizeComponent(tt.in); got != tt.want {
			t.Fatalf("normalizeComponent(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestExcludedIsNotOpened(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, ".planted-agent")
	if err := os.MkdirAll(filepath.Join(root, "projects", "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.json"), []byte(`{"token":"nope"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "projects", "keep", "ok.jsonl"), []byte(`{"id":"1"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	desc := syntheticDescriptor(home)
	desc.Storage.Excluded = []string{"auth.json"}
	art, err := Collect(context.Background(), agents.Env{Home: home, LookupEnv: func(string) string { return "" }},
		[]agents.Descriptor{desc}, Options{LookPath: func(string) (string, error) { return "", os.ErrNotExist }})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(art)
	if strings.Contains(string(raw), "nope") || strings.Contains(string(raw), "token") && strings.Contains(string(raw), "nope") {
		t.Fatalf("excluded file leaked: %s", raw)
	}
}

// A bare ~/.<agent> directory is not evidence that the agent is installed.
// Unrelated tooling plants such directories, so discovery stays marker-gated
// and the candidate is reported as existing without its marker.
func TestBareRootWithoutMarkerDoesNotResolve(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".planted-agent", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	art, err := Collect(context.Background(), agents.Env{
		Home:      home,
		LookupEnv: func(string) string { return "" },
	}, []agents.Descriptor{syntheticDescriptor(home)}, Options{
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version:  "0.5.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := art.Agents[0]
	if got.ResolvedRoot != nil {
		t.Fatalf("resolved_root = %+v, want nil for a root missing its marker", got.ResolvedRoot)
	}
	if got.ExecutableOnPath {
		t.Fatal("executable_on_path must stay false when the binary is absent")
	}
	if len(got.CandidateRoots) != 1 {
		t.Fatalf("candidate_roots = %d, want 1", len(got.CandidateRoots))
	}
	if c := got.CandidateRoots[0]; !c.Exists || c.MarkerPresent {
		t.Fatalf("candidate = %+v, want exists without marker", c)
	}
	if len(got.Tree) != 0 {
		t.Fatalf("tree = %+v, want no walk of an unresolved root", got.Tree)
	}
}

// Kimi Code buckets sessions as wd_<user>_<hash>, and nothing about an
// account name looks like an identifier to the token normalizer, so it used to
// survive into a committed artifact verbatim.
func TestAccountNameIsRedactedFromShapes(t *testing.T) {
	t.Parallel()
	home := filepath.Join(t.TempDir(), "arjunmehta")
	bucket := filepath.Join(home, ".planted-agent", "projects", "wd_arjunmehta_ab12cd34-17")
	if err := os.MkdirAll(bucket, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bucket, "session.jsonl"), []byte(`{"n":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	art, err := Collect(context.Background(), agents.Env{
		Home:      home,
		LookupEnv: func(string) string { return "" },
	}, []agents.Descriptor{syntheticDescriptor(home)}, Options{
		LookPath: func(string) (string, error) { return "", os.ErrNotExist },
		Now:      func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version:  "0.5.0-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	blob, err := json.Marshal(art)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(string(blob)), "arjunmehta") {
		t.Fatalf("artifact leaked the account name: %s", blob)
	}
	shapes := art.Agents[0].NameShapes
	if len(shapes) != 1 || shapes[0].Shape != "wd_<user>_ab12cd34-<n>" {
		t.Fatalf("redaction dropped the surrounding structure: %+v", shapes)
	}
}

// The budget is per agent. A machine with a dozen agents installed spawns a
// dozen --version subprocesses, and one slow harness must not discard the
// evidence already gathered for the others.
func TestSlowAgentDoesNotDiscardTheRun(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	slow := syntheticDescriptor(home)
	slow.Key = "slow"
	slow.Process = agents.ProcessSpec{Images: []string{"slow-agent"}}
	fast := syntheticDescriptor(home)
	fast.Key = "fast"
	fast.Process = agents.ProcessSpec{Images: []string{"fast-agent"}}

	art, err := Collect(context.Background(), agents.Env{
		Home:      home,
		LookupEnv: func(string) string { return "" },
	}, []agents.Descriptor{slow, fast}, Options{
		LookPath: func(name string) (string, error) { return "/usr/local/bin/" + name, nil },
		RunVersion: func(ctx context.Context, name string, args []string) (string, error) {
			if name != "slow-agent" {
				return "9.9.9", nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(2 * time.Second):
				return "9.9.9", nil
			}
		},
		Timeout: 80 * time.Millisecond,
		Now:     func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) },
		Version: "0.5.0-dev",
	})
	if err != nil {
		t.Fatalf("a slow agent must not fail the run: %v", err)
	}
	if err := Validate(art); err != nil {
		t.Fatal(err)
	}
	if len(art.Agents) != 2 {
		t.Fatalf("agents = %d, want both recorded", len(art.Agents))
	}
	byKey := map[string]Agent{}
	for _, rec := range art.Agents {
		byKey[rec.Key] = rec
	}
	if !byKey["slow"].TimedOut {
		t.Fatal("slow: want timed_out so a reader does not treat partial fields as a finding")
	}
	if byKey["fast"].TimedOut || byKey["fast"].VersionRaw != "9.9.9" {
		t.Fatalf("fast: a neighbour's deadline must not affect it: %+v", byKey["fast"])
	}
	if !byKey["slow"].ExecutableOnPath {
		t.Fatal("slow: evidence gathered before the deadline must survive")
	}
}

func syntheticDescriptor(home string) agents.Descriptor {
	return agents.Descriptor{
		Key:         "planted",
		DisplayName: "Planted Agent",
		Vendor:      "Test",
		Tier:        agents.TierDiscover,
		Family:      agents.FamilyHomeTree,
		Storage: agents.StorageSpec{
			RootEnv: "PLANTED_HOME",
			Roots: func(h agents.HomeDir) []agents.Root {
				return []agents.Root{{Path: h.Join(".planted-agent")}}
			},
			Marker:      "projects",
			SessionGlob: "projects/**/*.jsonl",
			Excluded:    []string{"auth.json", "cache"},
		},
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// TestProbeOrderingIsTotal covers AGENT-PROBE-V1 reproducibility. A dir node
// and a file node can normalize to the same path, so ordering by path alone
// left their order to sort.Slice, which is not stable. The tree is truncated
// to maxTreeRows afterwards, so the instability changed which rows shipped.
func TestProbeOrderingIsTotal(t *testing.T) {
	t.Parallel()
	nodes := []TreeNode{
		{Path: "*/*", Kind: "file", Count: 2},
		{Path: "*", Kind: "file", Count: 1},
		{Path: "*/*", Kind: "dir", Children: 3},
		{Path: "*", Kind: "dir", Children: 4},
	}
	for i := range nodes {
		for j := range nodes {
			if i == j {
				if treeNodeLess(nodes[i], nodes[j]) {
					t.Fatalf("node %d compares less than itself", i)
				}
				continue
			}
			if treeNodeLess(nodes[i], nodes[j]) == treeNodeLess(nodes[j], nodes[i]) {
				t.Fatalf("nodes %d and %d are order-ambiguous: %+v vs %+v", i, j, nodes[i], nodes[j])
			}
		}
	}
	shapes := []NameShape{
		{Path: "*", Shape: "b", Samples: 1},
		{Path: "*", Shape: "a", Samples: 1},
	}
	if nameShapeLess(shapes[0], shapes[1]) == nameShapeLess(shapes[1], shapes[0]) {
		t.Fatal("name shapes sharing a path are order-ambiguous")
	}
}
