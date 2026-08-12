package capsule

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

func TestCanonicalBytesStableAcrossShuffledMaps(t *testing.T) {
	t.Parallel()

	var first []byte
	for i := 0; i < 100; i++ {
		c := sampleCapsule()
		c.Capabilities.Source = shuffledStringMap(int64(i), map[string]any{
			"mcp_count":     2,
			"skill_count":   1,
			"tool_families": "shell,fs",
			"attachments":   true,
			"multi_root":    false,
		})
		c.Capabilities.Destination = shuffledStringMap(int64(i)+1000, map[string]any{
			"mcp_count":     1,
			"skill_count":   1,
			"tool_families": "shell,fs",
			"attachments":   true,
			"multi_root":    false,
		})
		c.Conversation.Events[0].Blocks[0].Meta = shuffledStringStringMap(int64(i)+3000, map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
			"d": "4",
		})

		got, err := CanonicalBytes(c)
		if err != nil {
			t.Fatalf("iteration %d: CanonicalBytes: %v", i, err)
		}
		if bytes.Contains(got, []byte("\r")) {
			t.Fatalf("iteration %d: canonical bytes contain CR", i)
		}
		if i == 0 {
			first = append([]byte(nil), got...)
			continue
		}
		if !bytes.Equal(first, got) {
			t.Fatalf("iteration %d: canonical bytes drifted\nfirst=%s\ngot=%s", i, first, got)
		}
	}
}

func TestComputeIDFixedPoint(t *testing.T) {
	t.Parallel()

	c := sampleCapsule()
	id, err := ComputeID(c)
	if err != nil {
		t.Fatalf("ComputeID: %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("ComputeID length = %d, want 32", len(id))
	}
	c.Identity.ID = id
	again, err := ComputeID(c)
	if err != nil {
		t.Fatalf("ComputeID with ID set: %v", err)
	}
	if again != id {
		t.Fatalf("ComputeID is not a fixed point: %q vs %q", id, again)
	}

	// Capsule with the ID set must still hash to the same ID when cleared.
	c.Identity.ID = id
	c.Identity.LineageRoot = id
	rooted, err := ComputeID(c)
	if err != nil {
		t.Fatalf("ComputeID with lineage: %v", err)
	}
	c.Identity.ID = rooted
	rootedAgain, err := ComputeID(c)
	if err != nil {
		t.Fatalf("ComputeID rooted again: %v", err)
	}
	if rootedAgain != rooted {
		t.Fatalf("rooted ComputeID not fixed: %q vs %q", rooted, rootedAgain)
	}
}

func TestComputeIDChangesWhenFieldChanges(t *testing.T) {
	t.Parallel()

	base := sampleCapsule()
	baseID, err := ComputeID(base)
	if err != nil {
		t.Fatalf("ComputeID base: %v", err)
	}

	type mutation struct {
		name string
		edit func(*Capsule)
	}
	mutations := []mutation{
		{"schema", func(c *Capsule) { c.Schema = Schema + "-x" }},
		{"lineage_root", func(c *Capsule) { c.Identity.LineageRoot = "other" }},
		{"parent.agent", func(c *Capsule) { c.Identity.Parent.Agent = "codex" }},
		{"raw_source.session_id", func(c *Capsule) { c.RawSource.SessionID = "other-session" }},
		{"task.goal", func(c *Capsule) { c.Task.Goal.Text = "other goal" }},
		{"workspace.root", func(c *Capsule) { c.Workspace.Root = "${REPO:other/repo}" }},
		{"workspace.dirty", func(c *Capsule) { c.Workspace.Dirty = !c.Workspace.Dirty }},
		{"conversation.event.text", func(c *Capsule) { c.Conversation.Events[0].Blocks[0].Text = "changed" }},
		{"conversation.event.order", func(c *Capsule) { c.Conversation.Events[0].Order = 99 }},
		{"capabilities.missing", func(c *Capsule) {
			c.Capabilities.Missing = append(c.Capabilities.Missing, MissingCapability{
				Kind: "mcp", Name: "extra", Impact: "degraded",
			})
		}},
		{"security.warning", func(c *Capsule) { c.Security.DestinationWarning = "grok_source_upload_history" }},
		{"fidelity.overall", func(c *Capsule) { c.Fidelity.Overall = PortabilityReferenced }},
		{"projection.policy", func(c *Capsule) { c.Projection.Policy = "full" }},
		{"event.portability", func(c *Capsule) {
			c.Conversation.Events[0].Portability = PortabilityNormalized
			c.Conversation.Events[0].Reason = "normalized_for_test"
		}},
	}

	for _, m := range mutations {
		m := m
		t.Run(m.name, func(t *testing.T) {
			t.Parallel()
			c := sampleCapsule()
			m.edit(&c)
			id, err := ComputeID(c)
			if err != nil {
				t.Fatalf("ComputeID: %v", err)
			}
			if id == baseID {
				t.Fatalf("ComputeID unchanged after mutating %s", m.name)
			}
		})
	}
}

func TestCanonicalBytesRejectsAbsolutePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		edit func(*Capsule)
	}{
		{
			name: "workspace.root_unix",
			edit: func(c *Capsule) { c.Workspace.Root = "/Users/example/project" },
		},
		{
			name: "workspace.root_windows",
			edit: func(c *Capsule) { c.Workspace.Root = `C:\Users\example\project` },
		},
		{
			name: "workspace.changed_files",
			edit: func(c *Capsule) {
				c.Workspace.ChangedFiles = []string{"${REPO:github.com/example/demo}/a.go", "/tmp/secret.go"}
			},
		},
		{
			name: "block.path",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0].Path = "/var/log/agent.log"
			},
		},
		{
			name: "task.changed_files",
			edit: func(c *Capsule) {
				c.Task.ChangedFiles.Items = []string{"/home/me/repo/main.go"}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := sampleCapsule()
			tc.edit(&c)
			_, err := CanonicalBytes(c)
			if err == nil {
				t.Fatal("CanonicalBytes succeeded; want absolute path error")
			}
		})
	}

	// Private absolute Path fields must not be serialized and must not fail encode.
	c := sampleCapsule()
	c.Workspace.Path = "/Users/example/project"
	c.RawSource.Path = "/Users/example/.claude/projects/x/y.jsonl"
	b, err := CanonicalBytes(c)
	if err != nil {
		t.Fatalf("CanonicalBytes with private paths: %v", err)
	}
	if bytes.Contains(b, []byte("/Users/example")) {
		t.Fatalf("private absolute path leaked into canonical bytes: %s", b)
	}
}

func TestEventIDStable(t *testing.T) {
	t.Parallel()

	p := SourcePointer{
		Agent:      "claude",
		SessionID:  "sess-1",
		RecordKey:  "uuid-1",
		ByteOffset: 42,
		Index:      7,
	}
	a := EventID(p)
	b := EventID(p)
	if a != b {
		t.Fatalf("EventID unstable: %q vs %q", a, b)
	}
	if len(a) != 32 {
		t.Fatalf("EventID length = %d, want 32", len(a))
	}
	p.Index = 8
	if EventID(p) == a {
		t.Fatal("EventID did not change when SourcePointer changed")
	}
}

func TestCanonicalTimestampNoFractionalSeconds(t *testing.T) {
	t.Parallel()

	c := sampleCapsule()
	c.Conversation.Events[0].Timestamp = time.Date(2026, 8, 12, 3, 4, 5, 123456789, time.FixedZone("IST", 5*3600+30*60))
	b, err := CanonicalBytes(c)
	if err != nil {
		t.Fatalf("CanonicalBytes: %v", err)
	}
	if bytes.Contains(b, []byte("123456789")) || bytes.Contains(b, []byte(".123")) {
		t.Fatalf("fractional timestamp survived: %s", b)
	}
	if !bytes.Contains(b, []byte(`"timestamp":"2026-08-11T21:34:05Z"`)) {
		t.Fatalf("expected truncated UTC timestamp, got %s", b)
	}
}

func sampleCapsule() Capsule {
	src := SourcePointer{
		Agent:      "claude",
		SessionID:  "sess-1",
		RecordKey:  "msg-1",
		ByteOffset: 128,
		Index:      0,
	}
	ev := Event{
		ID:          EventID(src),
		Order:       0,
		Timestamp:   time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC),
		Actor:       ActorUser,
		Kind:        KindMessage,
		NativeType:  "user",
		Blocks:      []Block{{Type: BlockTypeText, Text: "continue the handoff work"}},
		Portability: PortabilityExact,
		ContentHash: "abc123",
		Source:      src,
	}
	return Capsule{
		Schema: Schema,
		Identity: Identity{
			ID:          "",
			LineageRoot: "",
			Parent: Parent{
				Agent:          "claude",
				SessionID:      "sess-1",
				ArtifactSHA256: "deadbeef",
				AdapterVersion: "1",
			},
			SchemaVer: SchemaVersion,
		},
		RawSource: RawSource{
			Agent:          "claude",
			SessionID:      "sess-1",
			ArtifactSHA256: "deadbeef",
			AdapterVersion: "1",
			ByteOffset:     128,
			SizeBytes:      128,
		},
		Task: Task{
			Goal:               TextField{Text: "ship WP-03", Portability: PortabilityNormalized, Label: "derived_deterministic"},
			LatestUserIntent:   TextField{Text: "continue the handoff work", Portability: PortabilityExact},
			RecentUserMessages: ListField{Items: []string{"continue the handoff work"}, Portability: PortabilityExact},
			Constraints:        ListField{Portability: PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Decisions:          ListField{Portability: PortabilityOmitted, Reason: "requires_optional_summarizer"},
			RejectedApproaches: ListField{Portability: PortabilityOmitted, Reason: "requires_optional_summarizer"},
			Completed:          ListField{Items: []string{"wrote model types"}, Portability: PortabilityNormalized},
			Pending:            ListField{Portability: PortabilityOmitted, Reason: "interrupted_not_replayed"},
			ChangedFiles: ListField{
				Items:       []string{"${REPO:github.com/example/demo}/internal/capsule/model.go"},
				Portability: PortabilityExact,
			},
			FilesTouchedPerTranscript: ListField{
				Items:       []string{"${REPO:github.com/example/demo}/internal/capsule/model.go"},
				Portability: PortabilityReferenced,
				Label:       "transcript_claim",
			},
			Tests:      ListField{Items: []string{"go test ./internal/capsule"}, Portability: PortabilityReferenced},
			NextAction: TextField{Text: "continue the latest user request", Portability: PortabilityNormalized, Label: "derived_deterministic"},
		},
		Workspace: Workspace{
			ProjectID:         "github.com/example/demo",
			Root:              "${REPO:github.com/example/demo}",
			Branch:            "wp/03-capsule-model",
			Head:              "0d16aa6",
			Dirty:             true,
			WorkingTreeDigest: "sha256:feedface",
			ChangedFiles:      []string{"${REPO:github.com/example/demo}/internal/capsule/model.go"},
		},
		Conversation: Conversation{Events: []Event{ev}},
		Capabilities: CapabilityDiff{
			Source:      map[string]any{"mcp_count": 2, "skill_count": 1},
			Destination: map[string]any{"mcp_count": 1, "skill_count": 1},
			Missing: []MissingCapability{
				{Kind: "mcp", Name: "browser", Impact: "degraded"},
			},
		},
		Security: Security{
			SourceInstructionsAreUntrustedHistory: true,
			Redactions: []Redaction{
				{Category: CategoryGitHubToken, Digest: "abcdef012345"},
			},
		},
		Fidelity: Fidelity{
			Overall: PortabilityNormalized,
			Mode:    FidelityModeStructuredHandoff,
			Components: []Component{
				{Name: "user_messages", Portability: PortabilityExact, Count: 1, Bytes: 24},
				{Name: "tool_results", Portability: PortabilityNormalized, Count: 0, Bytes: 0, Reason: "none"},
			},
		},
		Projection: Projection{
			Policy:           "balanced",
			EstimatedBytes:   1024,
			EstimatedTokens:  256,
			IncludedEventIDs: []string{ev.ID},
		},
	}
}

func shuffledStringMap(seed int64, in map[string]any) map[string]any {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	out := make(map[string]any, len(in))
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}

func shuffledStringStringMap(seed int64, in map[string]string) map[string]string {
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	r := rand.New(rand.NewSource(seed))
	r.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	out := make(map[string]string, len(in))
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}
