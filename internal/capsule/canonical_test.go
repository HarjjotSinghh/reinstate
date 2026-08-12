package capsule

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
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

	// Self-referential and post-render derived fields are outside the identity
	// preimage, so assigning them preserves the fixed point.
	c.Identity.ID = id
	c.Identity.LineageRoot = id
	c.Projection.EstimatedBytes = 1234
	c.Projection.EstimatedTokens = 309
	c.Projection.BootstrapSHA256 = strings.Repeat("a", 64)
	c.Projection.MarkdownSHA256 = strings.Repeat("b", 64)
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
		{"ancestor_lineage_root", func(c *Capsule) { c.Identity.LineageRoot = "other" }},
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
		{"projection.included_event_ids", func(c *Capsule) { c.Projection.IncludedEventIDs = []string{"other"} }},
		{"projection.sidecar_ref", func(c *Capsule) { c.Projection.SidecarRef = "sidecar/events.jsonl" }},
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

func TestComputeIDExcludesOnlySelfAndRenderedArtifactFields(t *testing.T) {
	t.Parallel()

	base := sampleCapsule()
	want, err := ComputeID(base)
	if err != nil {
		t.Fatal(err)
	}
	excluded := []struct {
		name string
		edit func(*Capsule)
	}{
		{"identity.id", func(c *Capsule) { c.Identity.ID = "other" }},
		{"identity.self_lineage_root", func(c *Capsule) {
			c.Identity.ID = "self"
			c.Identity.LineageRoot = "self"
		}},
		{"projection.estimated_bytes", func(c *Capsule) { c.Projection.EstimatedBytes++ }},
		{"projection.estimated_tokens", func(c *Capsule) { c.Projection.EstimatedTokens++ }},
		{"projection.bootstrap_sha256", func(c *Capsule) { c.Projection.BootstrapSHA256 = "other" }},
		{"projection.markdown_sha256", func(c *Capsule) { c.Projection.MarkdownSHA256 = "other" }},
	}
	for _, tt := range excluded {
		t.Run(tt.name, func(t *testing.T) {
			c := sampleCapsule()
			tt.edit(&c)
			got, err := ComputeID(c)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("derived field changed ID: got %q, want %q", got, want)
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
		{
			name: "task.files_touched_per_transcript",
			edit: func(c *Capsule) {
				c.Task.FilesTouchedPerTranscript.Items = []string{"/home/me/repo/main.go"}
			},
		},
		{
			name: "block.ref",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0].Ref = "/var/folders/attachment.png"
			},
		},
		{
			name: "block.meta",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0].Meta = map[string]string{
					"name": "/Users/example/Desktop/shot.png",
				}
			},
		},
		{
			name: "block.tool_input_path_field",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0] = Block{
					Type: BlockTypeToolInput,
					Text: `{"file_path":"/Users/example/project/main.go"}`,
				}
			},
		},
		{
			name: "block.tool_input_nested_paths_array",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0] = Block{
					Type: BlockTypeToolInput,
					Text: `{"args":{"paths":["${REPO:x}/a.go","/etc/hosts"]}}`,
				}
			},
		},
		{
			name: "block.tool_input_bare_value",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0] = Block{
					Type: BlockTypeToolInput,
					Text: "/usr/local/bin/tool --run",
				}
			},
		},
		{
			name: "block.tool_output",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].Blocks[0] = Block{
					Type: BlockTypeToolOutput,
					Text: "/Users/example/project\nok",
				}
			},
		},
		{
			name: "conversation.full_history_ref",
			edit: func(c *Capsule) {
				c.Conversation.FullHistoryRef = "/tmp/handoff/sidecar/events.jsonl"
			},
		},
		{
			name: "projection.sidecar_ref",
			edit: func(c *Capsule) {
				c.Projection.SidecarRef = "/tmp/handoff/sidecar/events.jsonl"
			},
		},
		{
			name: "workspace.tests",
			edit: func(c *Capsule) {
				c.Workspace.Tests = []string{"/usr/local/bin/pytest -q"}
			},
		},
		{
			name: "task.tests",
			edit: func(c *Capsule) {
				c.Task.Tests.Items = []string{"/usr/local/bin/pytest -q"}
			},
		},
		{
			name: "event.native_name",
			edit: func(c *Capsule) {
				c.Conversation.Events[0].NativeName = "/Users/example/bin/tool"
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
			if !strings.Contains(err.Error(), "absolute filesystem path is not allowed in ") {
				t.Fatalf("error does not name the offending field: %v", err)
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

// TestCanonicalBytesAcceptsProse pins the other half of the contract: free text
// is not a path. A user who types a slash command, or names an absolute path in
// a sentence, wrote prose — canonicalization must carry it verbatim rather than
// abort the handoff, which is the v0.4.0-rc.1 defect this test guards.
func TestCanonicalBytesAcceptsProse(t *testing.T) {
	t.Parallel()

	prose := []string{
		"/init do the thing",
		"/compact",
		"/clear",
		"look at /etc/hosts before you continue",
		"/Users/example/project/main.go is the file I meant",
		`C:\Users\example\project\main.go on the Windows box`,
	}

	// Each case stores the prose in one field and returns the exact string the
	// capsule now holds, so the test can prove it survived byte for byte.
	cases := []struct {
		name string
		edit func(*Capsule, string) string
	}{
		{"event.text_block", func(c *Capsule, s string) string {
			c.Conversation.Events[0].Blocks[0].Text = s
			return s
		}},
		{"task.goal", func(c *Capsule, s string) string { c.Task.Goal.Text = s; return s }},
		{"task.latest_user_intent", func(c *Capsule, s string) string { c.Task.LatestUserIntent.Text = s; return s }},
		{"task.recent_user_messages", func(c *Capsule, s string) string {
			c.Task.RecentUserMessages.Items = []string{s}
			return s
		}},
		{"task.next_action", func(c *Capsule, s string) string { c.Task.NextAction.Text = s; return s }},
		{"task.open_questions", func(c *Capsule, s string) string {
			c.Task.OpenQuestions.Items = []string{s}
			return s
		}},
		{"task.constraints", func(c *Capsule, s string) string { c.Task.Constraints.Items = []string{s}; return s }},
		{"task.decisions", func(c *Capsule, s string) string { c.Task.Decisions.Items = []string{s}; return s }},
		{"task.rejected_approaches", func(c *Capsule, s string) string {
			c.Task.RejectedApproaches.Items = []string{s}
			return s
		}},
		{"security.destination_warning", func(c *Capsule, s string) string { c.Security.DestinationWarning = s; return s }},
		{"tool_input.non_path_argument", func(c *Capsule, s string) string {
			// A tool argument can be prose too: only path-typed keys are judged.
			text := `{"prompt":` + quote(s) + `,"file_path":"${REPO:github.com/example/demo}/main.go"}`
			c.Conversation.Events[0].Blocks[0] = Block{Type: BlockTypeToolInput, Text: text}
			return text
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, text := range prose {
				c := sampleCapsule()
				stored := tc.edit(&c, text)
				got, err := CanonicalBytes(c)
				if err != nil {
					t.Fatalf("CanonicalBytes rejected prose %q: %v", text, err)
				}
				if !bytes.Contains(got, []byte(quoteInner(stored))) {
					t.Fatalf("prose %q did not survive canonicalization: %s", text, got)
				}
				if err := Validate(c); err != nil {
					t.Fatalf("Validate rejected prose %q: %v", text, err)
				}
			}
		})
	}
}

func quote(s string) string {
	enc, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(enc)
}

// quoteInner is the JSON encoding of s without its surrounding quotes, which is
// how the value appears inside canonical bytes.
func quoteInner(s string) string {
	enc := quote(s)
	return enc[1 : len(enc)-1]
}

// TestPathClassificationCoversEverySerializedStringField fails when a
// serialized string-bearing field of the capsule has no recorded
// path-vs-prose classification.
//
// rejectAbsolutePathFields visits named fields instead of walking every decoded
// string, which is what lets prose stay prose. The cost of that choice is that
// a newly added field is not covered until someone classifies it, so this test
// makes forgetting a build failure. When it fails: decide whether the new field
// holds a filesystem path, add it to rejectAbsolutePathFields with p.path,
// p.opaque, or a deliberate prose exemption, and record it below.
func TestPathClassificationCoversEverySerializedStringField(t *testing.T) {
	t.Parallel()

	// classification records, for every serialized string leaf, whether
	// rejectAbsolutePathFields checks it and why.
	classification := map[string]string{
		// Path-typed: must be a portable pathmap token.
		"workspace.root":                            "path",
		"workspace.changed_files[]":                 "path",
		"task.changed_files.items[]":                "path",
		"task.files_touched_per_transcript.items[]": "path",
		"conversation.full_history_ref":             "path",
		"projection.sidecar_ref":                    "path",
		"conversation.events[].blocks[].path":       "path",
		"conversation.events[].blocks[].ref":        "path",
		"conversation.events[].blocks[].meta[]":     "path",

		// Structural scalars: identifiers, digests, enums, vendor names, and
		// derived evidence lines. None can legitimately be a path.
		"schema":                                      "opaque",
		"identity.id":                                 "opaque",
		"identity.lineage_root":                       "opaque",
		"identity.parent_session.agent":               "opaque",
		"identity.parent_session.id":                  "opaque",
		"identity.parent_session.artifact_sha256":     "opaque",
		"identity.parent_session.adapter_version":     "opaque",
		"raw_source.agent":                            "opaque",
		"raw_source.session_id":                       "opaque",
		"raw_source.artifact_sha256":                  "opaque",
		"raw_source.adapter_version":                  "opaque",
		"workspace.project_id":                        "opaque",
		"workspace.branch":                            "opaque",
		"workspace.head":                              "opaque",
		"workspace.working_tree_digest":               "opaque",
		"workspace.tests[]":                           "opaque",
		"task.completed.items[]":                      "opaque",
		"task.pending.items[]":                        "opaque",
		"task.tests.items[]":                          "opaque",
		"capabilities.source[]":                       "opaque",
		"capabilities.destination[]":                  "opaque",
		"capabilities.missing[].kind":                 "opaque",
		"capabilities.missing[].name":                 "opaque",
		"capabilities.missing[].impact":               "opaque",
		"security.redactions[].category":              "opaque",
		"security.redactions[].digest":                "opaque",
		"fidelity.overall":                            "opaque",
		"fidelity.mode":                               "opaque",
		"fidelity.components[].name":                  "opaque",
		"fidelity.components[].portability":           "opaque",
		"fidelity.components[].reason":                "opaque",
		"fidelity.unsupported[]":                      "opaque",
		"projection.policy":                           "opaque",
		"projection.included_event_ids[]":             "opaque",
		"projection.bootstrap_sha256":                 "opaque",
		"projection.markdown_sha256":                  "opaque",
		"conversation.events[].id":                    "opaque",
		"conversation.events[].actor":                 "opaque",
		"conversation.events[].kind":                  "opaque",
		"conversation.events[].native_type":           "opaque",
		"conversation.events[].native_name":           "opaque",
		"conversation.events[].call_id":               "opaque",
		"conversation.events[].linked_call_id":        "opaque",
		"conversation.events[].portability":           "opaque",
		"conversation.events[].reason":                "opaque",
		"conversation.events[].content_hash":          "opaque",
		"conversation.events[].source.agent":          "opaque",
		"conversation.events[].source.session_id":     "opaque",
		"conversation.events[].source.record_key":     "opaque",
		"conversation.events[].redactions[].category": "opaque",
		"conversation.events[].redactions[].digest":   "opaque",
		"conversation.events[].blocks[].type":         "opaque",
		"conversation.events[].blocks[].mime":         "opaque",
		"conversation.events[].blocks[].sha256":       "opaque",

		// Prose: written by a person or a model. Never judged as a path. A
		// block's text is prose only for text blocks; tool_input and json
		// payloads are parsed and their path-typed keys are checked.
		"task.goal.text":                      "prose",
		"task.latest_user_intent.text":        "prose",
		"task.next_action.text":               "prose",
		"task.recent_user_messages.items[]":   "prose",
		"task.constraints.items[]":            "prose",
		"task.decisions.items[]":              "prose",
		"task.rejected_approaches.items[]":    "prose",
		"task.open_questions.items[]":         "prose",
		"security.destination_warning":        "prose",
		"conversation.events[].blocks[].text": "prose_or_tool_arguments",

		// Portability metadata on task fields: constants from this package.
		"task.goal.portability":                         "opaque",
		"task.goal.reason":                              "opaque",
		"task.goal.label":                               "opaque",
		"task.latest_user_intent.portability":           "opaque",
		"task.latest_user_intent.reason":                "opaque",
		"task.latest_user_intent.label":                 "opaque",
		"task.next_action.portability":                  "opaque",
		"task.next_action.reason":                       "opaque",
		"task.next_action.label":                        "opaque",
		"task.recent_user_messages.portability":         "opaque",
		"task.recent_user_messages.reason":              "opaque",
		"task.recent_user_messages.label":               "opaque",
		"task.constraints.portability":                  "opaque",
		"task.constraints.reason":                       "opaque",
		"task.constraints.label":                        "opaque",
		"task.decisions.portability":                    "opaque",
		"task.decisions.reason":                         "opaque",
		"task.decisions.label":                          "opaque",
		"task.rejected_approaches.portability":          "opaque",
		"task.rejected_approaches.reason":               "opaque",
		"task.rejected_approaches.label":                "opaque",
		"task.completed.portability":                    "opaque",
		"task.completed.reason":                         "opaque",
		"task.completed.label":                          "opaque",
		"task.pending.portability":                      "opaque",
		"task.pending.reason":                           "opaque",
		"task.pending.label":                            "opaque",
		"task.changed_files.portability":                "opaque",
		"task.changed_files.reason":                     "opaque",
		"task.changed_files.label":                      "opaque",
		"task.files_touched_per_transcript.portability": "opaque",
		"task.files_touched_per_transcript.reason":      "opaque",
		"task.files_touched_per_transcript.label":       "opaque",
		"task.tests.portability":                        "opaque",
		"task.tests.reason":                             "opaque",
		"task.tests.label":                              "opaque",
		"task.open_questions.portability":               "opaque",
		"task.open_questions.reason":                    "opaque",
		"task.open_questions.label":                     "opaque",
	}

	found := stringLeaves(reflect.TypeOf(Capsule{}), "")
	for _, field := range found {
		if _, ok := classification[field]; !ok {
			t.Errorf("field %q has no path-vs-prose classification; classify it in "+
				"rejectAbsolutePathFields and record it in this test", field)
		}
	}
	seen := make(map[string]struct{}, len(found))
	for _, field := range found {
		seen[field] = struct{}{}
	}
	for field := range classification {
		if _, ok := seen[field]; !ok {
			t.Errorf("classification lists %q, which is no longer a serialized field", field)
		}
	}
}

// stringLeaves returns the dotted JSON paths of every serialized field that can
// carry a string, including strings reachable through slices, maps, and `any`.
func stringLeaves(t reflect.Type, prefix string) []string {
	switch t.Kind() {
	case reflect.String:
		return []string{prefix}
	case reflect.Interface:
		// An `any` value (a capability summary entry) can carry a string.
		return []string{prefix}
	case reflect.Pointer:
		return stringLeaves(t.Elem(), prefix)
	case reflect.Slice, reflect.Array:
		return stringLeaves(t.Elem(), prefix+"[]")
	case reflect.Map:
		return stringLeaves(t.Elem(), prefix+"[]")
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return nil
		}
		out := make([]string, 0)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "-" {
				// Private by contract: never serialized, never checked.
				continue
			}
			if name == "" {
				name = field.Name
			}
			child := name
			if prefix != "" {
				child = prefix + "." + name
			}
			out = append(out, stringLeaves(field.Type, child)...)
		}
		return out
	default:
		return nil
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
