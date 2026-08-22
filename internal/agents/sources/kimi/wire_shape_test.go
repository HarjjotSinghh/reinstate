package kimi

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeShapeSession lays out one session tree and returns the fixture root.
func writeShapeSession(t *testing.T, wire, state string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "sessions", "wd_fixture-user_a1b2c3d4e5f6",
		"session_01987654-3210-7890-abcd-ef0123456789")
	if err := os.MkdirAll(filepath.Join(dir, "agents", "main"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "agents", "main", "wire.jsonl"), []byte(wire), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

const shapeState = `{
  "id": "session_01987654-3210-7890-abcd-ef0123456789",
  "title": "Bound the retry budget",
  "titleKind": "replaceable",
  "isCustomTitle": false,
  "cwd": "/Users/fixture-user/code/demo",
  "createdAt": 1786870800000,
  "updatedAt": 1786871550000,
  "lastPrompt": "Bound the retry budget in agentcheck",
  "lastTurnReason": "completed",
  "agents": {"main": {"homedir": "/Users/fixture-user/.kimi-code/sessions/x/y", "type": "main"}},
  "archived": false,
  "custom": {},
  "version": 2
}`

// TestLegacyMigratedShapeStillIndexes keeps the protocol 1.0 branch covered.
//
// Both committed fixtures now carry the native 1.5 shape, because that is what
// a current install writes. A session migrated from the legacy kimi-cli store
// is the one place a role "assistant" context.append_message is real, and it
// has no turn.prompt at all, so nothing else would exercise that path.
func TestLegacyMigratedShapeStillIndexes(t *testing.T) {
	t.Parallel()
	wire := strings.Join([]string{
		`{"type":"metadata","protocol_version":"1.0","created_at":1786870800000}`,
		`{"type":"context.append_message","time":1786870801000,"message":{"id":"m1","role":"user","content":[{"type":"text","text":"Bound the retry budget in agentcheck"}],"toolCalls":[]}}`,
		`{"type":"context.append_message","time":1786870802000,"message":{"id":"m2","role":"assistant","content":[{"type":"text","text":"Reading the current budget."}],"toolCalls":[{"name":"read_file","arguments":{"path":"internal/agentcheck/agent.go"}}]}}`,
		"",
	}, "\n")

	result := scan(t, writeShapeSession(t, wire, shapeState))
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if want := []string{"internal/agentcheck/agent.go"}; !reflect.DeepEqual(record.Files, want) {
		t.Fatalf("files = %v, want %v", record.Files, want)
	}
	if record.MessageCount == 0 {
		t.Fatal("a migrated session counted no messages")
	}
}

// TestNativeShapeIndexesToolFilesAndTurns is the regression.
//
// At protocol 1.5 the assistant's whole side of a turn arrives as
// context.append_loop_event. Reading only the legacy context.append_message
// shape meant a session from a current install was indexed with no files at all
// and with every assistant turn uncounted — and the fixtures encoded the legacy
// shape too, so the tests agreed with the reader while neither agreed with the
// vendor.
func TestNativeShapeIndexesToolFilesAndTurns(t *testing.T) {
	t.Parallel()
	wire := strings.Join([]string{
		`{"type":"metadata","protocol_version":"1.5","created_at":1786870800000}`,
		`{"type":"turn.prompt","time":1786870801000,"origin":{"kind":"user"},"input":[{"type":"text","text":"Bound the retry budget in agentcheck"}]}`,
		`{"type":"context.append_message","time":1786870801100,"message":{"id":"m1","role":"user","origin":{"kind":"user"},"content":[{"type":"text","text":"Bound the retry budget in agentcheck"}],"toolCalls":[]}}`,
		`{"type":"context.append_loop_event","time":1786870802000,"event":{"type":"step.begin","uuid":"s1","turnId":"0","step":1}}`,
		`{"type":"context.append_loop_event","time":1786870803000,"event":{"type":"content.part","uuid":"p1","stepUuid":"s1","part":{"type":"text","text":"Reading the current budget."}}}`,
		`{"type":"context.append_loop_event","time":1786870804000,"event":{"type":"tool.call","uuid":"c1","stepUuid":"s1","toolCallId":"call_1","name":"Read","args":{"path":"internal/agentcheck/agent.go"}}}`,
		`{"type":"context.append_loop_event","time":1786870805000,"event":{"type":"tool.result","parentUuid":"c1","toolCallId":"call_1","result":{"output":"1\tretryBudget"}}}`,
		`{"type":"context.append_loop_event","time":1786870806000,"event":{"type":"step.end","uuid":"s1","finishReason":"end_turn"}}`,
		"",
	}, "\n")

	result := scan(t, writeShapeSession(t, wire, shapeState))
	if len(result.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(result.Records))
	}
	record := result.Records[0]
	if want := []string{"internal/agentcheck/agent.go"}; !reflect.DeepEqual(record.Files, want) {
		t.Fatalf("files = %v, want %v; a tool.call names its arguments \"args\"", record.Files, want)
	}
	if record.MessageCount < 2 {
		t.Fatalf("message_count = %d; the prompt and the assistant step should both count",
			record.MessageCount)
	}
}

// TestUnknownLoopEventIsNotConversation keeps the vocabulary open. The shipped
// bundle registers many more ops than this reader names, and an unrecognised
// one must be ignored rather than guessed at.
func TestUnknownLoopEventIsNotConversation(t *testing.T) {
	t.Parallel()
	wire := strings.Join([]string{
		`{"type":"metadata","protocol_version":"1.5","created_at":1786870800000}`,
		`{"type":"turn.prompt","time":1786870801000,"origin":{"kind":"user"},"input":[{"type":"text","text":"Bound the retry budget in agentcheck"}]}`,
		`{"type":"context.append_loop_event","time":1786870802000,"event":{"type":"some.future.event","args":{"path":"internal/should/not/appear.go"}}}`,
		"",
	}, "\n")

	record := scan(t, writeShapeSession(t, wire, shapeState)).Records[0]
	for _, file := range record.Files {
		if strings.Contains(file, "should/not/appear") {
			t.Fatalf("an unknown loop event contributed a file: %v", record.Files)
		}
	}
}

// TestModelInternalsNeverReachTheIndex guards the committed fixtures. They
// carry a system prompt and a thinking part precisely so this can be asserted:
// neither is conversation, and neither may be indexed or previewed.
func TestModelInternalsNeverReachTheIndex(t *testing.T) {
	t.Parallel()
	for _, osName := range []string{"macos", "windows"} {
		t.Run(osName, func(t *testing.T) {
			result := scan(t, fixture(t, osName))
			blob := ""
			for _, record := range result.Records {
				blob += record.Title + "\x00" + record.PromptPreview + "\x00" +
					record.SearchText + "\x00" + strings.Join(record.Files, "\x00")
			}
			for _, sentinel := range []string{"NEVER_COPY_THIS_SYSTEM_PROMPT", "NEVER_COPY_THIS_THINKING", "NEVER_COPY_THIS_INJECTION"} {
				if strings.Contains(blob, sentinel) {
					t.Fatalf("%s reached the index record", sentinel)
				}
			}
		})
	}
}
