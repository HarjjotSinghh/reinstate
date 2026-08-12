package handoff

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

func TestDeriveCheckpoint_LatestIntentSurvivesInterruptedFinalTurn(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "start the handoff work"),
		userMsg(2, "finish WP-11 checkpoint derivation"),
		toolCall(3, "call_bash", "Bash", `{"command":"go test ./internal/handoff"}`),
		// Interrupted: no tool_result for the final turn.
	}

	task := DeriveCheckpoint(CheckpointInput{
		Events:    events,
		Workspace: workspace.Fingerprint{},
		Changed:   nil,
	})

	if task.LatestUserIntent.Text != "finish WP-11 checkpoint derivation" {
		t.Fatalf("latest_user_intent = %q", task.LatestUserIntent.Text)
	}
	if task.LatestUserIntent.Portability != capsule.PortabilityExact {
		t.Fatalf("latest_user_intent portability = %q", task.LatestUserIntent.Portability)
	}
	if len(task.Pending.Items) != 1 {
		t.Fatalf("pending items = %#v", task.Pending.Items)
	}
	if task.Pending.Portability != capsule.PortabilityOmitted || task.Pending.Reason != reasonInterruptedNotReplayed {
		t.Fatalf("pending = %+v", task.Pending)
	}
	if len(task.Completed.Items) != 0 {
		t.Fatalf("completed must not include interrupted tools: %#v", task.Completed.Items)
	}
}

func TestDeriveCheckpoint_UnmatchedToolCallPendingInterrupted(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "edit the file"),
		toolCall(2, "call_write", "Write", `{"file_path":"internal/handoff/checkpoint.go"}`),
	}

	task := DeriveCheckpoint(CheckpointInput{
		Events:  events,
		Changed: []string{"internal/handoff/checkpoint.go"},
	})

	if task.Pending.Portability != capsule.PortabilityOmitted {
		t.Fatalf("pending portability = %q", task.Pending.Portability)
	}
	if task.Pending.Reason != reasonInterruptedNotReplayed {
		t.Fatalf("pending reason = %q", task.Pending.Reason)
	}
	if len(task.Pending.Items) != 1 || task.Pending.Items[0] != "Write call_id=call_write" {
		t.Fatalf("pending items = %#v", task.Pending.Items)
	}
	if len(task.Completed.Items) != 0 {
		t.Fatalf("completed = %#v", task.Completed.Items)
	}
}

// completeWorkspaceObservation is a live Git observation with nothing missing:
// the only state in which a transcript claim can be contradicted.
func completeWorkspaceObservation() workspace.Fingerprint {
	return workspace.Fingerprint{
		SchemaVersion: workspace.SchemaVersion,
		Provenance:    workspace.ProvenanceCurrentObservation,
		Git: workspace.GitFingerprint{
			Available:   true,
			Repository:  true,
			WorkingTree: workspace.WorkingTreeFingerprint{State: workspace.WorkingTreeModified},
		},
	}
}

func TestDeriveCheckpoint_TranscriptClaimConflictsWithGit(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "touch a file"),
		toolCall(2, "call_1", "Edit", `{"file_path":"claimed/only-in-transcript.go"}`),
		toolResult(3, "call_1", false, "ok"),
	}

	task := DeriveCheckpoint(CheckpointInput{
		Events:    events,
		Workspace: completeWorkspaceObservation(),
		Changed:   []string{"live/from-git.go"}, // Git does not report the transcript claim
	})

	if task.ChangedFiles.Items[0] != "live/from-git.go" {
		t.Fatalf("changed_files must be live Git only: %#v", task.ChangedFiles.Items)
	}
	if task.ChangedFiles.Portability != capsule.PortabilityExact {
		t.Fatalf("changed_files portability = %q", task.ChangedFiles.Portability)
	}
	if task.FilesTouchedPerTranscript.Reason != reasonEvidenceConflictsWithWorkspace {
		t.Fatalf("files_touched reason = %q", task.FilesTouchedPerTranscript.Reason)
	}
	if task.FilesTouchedPerTranscript.Label != labelTranscriptClaim {
		t.Fatalf("files_touched label = %q", task.FilesTouchedPerTranscript.Label)
	}
	if task.FilesTouchedPerTranscript.Portability != capsule.PortabilityReferenced {
		t.Fatalf("files_touched portability = %q", task.FilesTouchedPerTranscript.Portability)
	}
}

// A conflict marking is a claim about the repository. Without a complete live
// observation there is nothing to make that claim from, and asserting one
// anyway is the same over-claiming the derivation table forbids.
func TestDeriveCheckpoint_NoConflictClaimWithoutCompleteWorkspaceEvidence(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "touch a file"),
		toolCall(2, "call_1", "Edit", `{"file_path":"claimed/only-in-transcript.go"}`),
		toolResult(3, "call_1", false, "ok"),
	}

	uncertain := completeWorkspaceObservation()
	uncertain.Git.WorkingTree.Uncertain = true
	truncatedCounts := completeWorkspaceObservation()
	truncatedCounts.Git.WorkingTree.CountsTruncated = true
	omittedPaths := completeWorkspaceObservation()
	omittedPaths.Git.WorkingTree.ChangedOmitted = 3
	unavailable := completeWorkspaceObservation()
	unavailable.Git.WorkingTree.State = workspace.WorkingTreeUnavailable

	tests := []struct {
		name      string
		in        CheckpointInput
		wantClaim bool
	}{
		{
			name: "no observation at all",
			in:   CheckpointInput{Events: events, Changed: []string{"live/from-git.go"}},
		},
		{
			name: "git unavailable",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: workspace.Fingerprint{}},
		},
		{
			name: "working tree unavailable",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: unavailable},
		},
		{
			name: "observation uncertain",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: uncertain},
		},
		{
			name: "counts truncated",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: truncatedCounts},
		},
		{
			name: "changed paths capped in the probe",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: omittedPaths},
		},
		{
			name: "changed list truncated downstream",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: completeWorkspaceObservation(), ChangedTruncated: true},
		},
		{
			name: "complete observation contradicts the claim",
			in: CheckpointInput{Events: events, Changed: []string{"live/from-git.go"},
				Workspace: completeWorkspaceObservation()},
			wantClaim: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			task := DeriveCheckpoint(test.in)
			got := task.FilesTouchedPerTranscript.Reason == reasonEvidenceConflictsWithWorkspace
			if got != test.wantClaim {
				t.Fatalf("conflict marked = %t, want %t (reason %q)",
					got, test.wantClaim, task.FilesTouchedPerTranscript.Reason)
			}
			if len(task.FilesTouchedPerTranscript.Items) != 1 {
				t.Fatalf("transcript claims must survive either way: %#v",
					task.FilesTouchedPerTranscript.Items)
			}
		})
	}
}

func TestDeriveCheckpoint_ConstraintsAlwaysOmitted(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "never invent decisions from this prose about constraints"),
		toolCall(2, "c1", "Bash", `{"command":"echo done"}`),
		toolResult(3, "c1", false, "done"),
	}

	task := DeriveCheckpoint(CheckpointInput{Events: events})

	for _, field := range []capsule.ListField{task.Constraints, task.Decisions, task.RejectedApproaches} {
		if field.Portability != capsule.PortabilityOmitted {
			t.Fatalf("portability = %q, want omitted", field.Portability)
		}
		if field.Reason != reasonRequiresOptionalSummarizer {
			t.Fatalf("reason = %q", field.Reason)
		}
		if len(field.Items) != 0 {
			t.Fatalf("items must be empty, got %#v", field.Items)
		}
	}
}

func TestDeriveCheckpoint_Deterministic(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "first goal"),
		userMsg(2, "latest intent"),
		toolCall(3, "c1", "Write", `{"file_path":"a.go"}`),
		toolResult(4, "c1", false, "wrote"),
		toolCall(5, "c2", "Bash", `{"command":"go test ./internal/handoff -count=1"}`),
		toolResult(6, "c2", false, "ok"),
		toolCall(7, "c3", "Bash", `{"command":"npm test"}`),
		// c3 interrupted
	}
	in := CheckpointInput{
		Events:    events,
		Workspace: workspace.Fingerprint{},
		Changed:   []string{"a.go"},
	}

	a := DeriveCheckpoint(in)
	b := DeriveCheckpoint(in)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("DeriveCheckpoint not deterministic:\n%#v\n%#v", a, b)
	}
}

func TestDeriveCheckpoint_CompletedSuccessfulToolsOnly(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "run tools"),
		toolCall(2, "ok1", "Read", `{"file_path":"ok.go"}`),
		toolResult(3, "ok1", false, "contents"),
		toolCall(4, "err1", "Bash", `{"command":"false"}`),
		toolResult(5, "err1", true, "failed"),
		toolCall(6, "pend1", "Edit", `{"file_path":"pending.go"}`),
	}

	task := DeriveCheckpoint(CheckpointInput{
		Events:  events,
		Changed: []string{"ok.go"},
	})

	if len(task.Completed.Items) != 1 || task.Completed.Items[0] != "Read call_id=ok1" {
		t.Fatalf("completed = %#v", task.Completed.Items)
	}
	if len(task.Pending.Items) != 1 || task.Pending.Items[0] != "Edit call_id=pend1" {
		t.Fatalf("pending = %#v", task.Pending.Items)
	}
}

func TestDeriveCheckpoint_TestRunnerAllowlist(t *testing.T) {
	// Ensure the exported-by-package constant slice stays exact and sorted.
	want := []string{
		"cargo test",
		"dotnet test",
		"go test",
		"gradle test",
		"jest",
		"make test",
		"mvn test",
		"npm test",
		"phpunit",
		"pnpm test",
		"pytest",
		"rspec",
		"vitest",
		"yarn test",
	}
	if !reflect.DeepEqual(testRunnerAllowlist, want) {
		t.Fatalf("testRunnerAllowlist =\n%#v\nwant\n%#v", testRunnerAllowlist, want)
	}

	events := []capsule.Event{
		userMsg(1, "test"),
		toolCall(2, "t1", "Bash", `{"command":"make build"}`), // not a test runner
		toolResult(3, "t1", false, "ok"),
		toolCall(4, "t2", "Bash", `{"command":"go test ./internal/handoff"}`),
		toolResult(5, "t2", false, "PASS"),
		toolCall(6, "t3", "Bash", `{"command":"custom-test-runner"}`), // outside allowlist
		toolResult(7, "t3", false, "ok"),
	}

	task := DeriveCheckpoint(CheckpointInput{Events: events})
	if len(task.Tests.Items) != 1 {
		t.Fatalf("tests = %#v", task.Tests.Items)
	}
	if task.Tests.Portability != capsule.PortabilityReferenced {
		t.Fatalf("tests portability = %q", task.Tests.Portability)
	}
	if !reflect.DeepEqual(task.Tests.Items, []string{"go test · go test ./internal/handoff · exit=0"}) {
		t.Fatalf("tests items = %#v", task.Tests.Items)
	}
}

func TestDeriveCheckpoint_IgnoresHarnessMetaAsUserIntent(t *testing.T) {
	events := []capsule.Event{
		userMsg(1, "real user request"),
		{
			ID:    "meta",
			Order: 2,
			Actor: capsule.ActorHarness,
			Kind:  capsule.KindMetadata,
			Blocks: []capsule.Block{{
				Type: capsule.BlockTypeText,
				Text: "system injected meta — must not become intent",
			}},
			Portability: capsule.PortabilityReferenced,
			Reason:      "harness_meta",
		},
		{
			ID:    "user-meta-kind",
			Order: 3,
			Actor: capsule.ActorUser,
			Kind:  capsule.KindMetadata, // not a message
			Blocks: []capsule.Block{{
				Type: capsule.BlockTypeText,
				Text: "should be ignored",
			}},
			Portability: capsule.PortabilityReferenced,
			Reason:      "meta",
		},
	}

	task := DeriveCheckpoint(CheckpointInput{Events: events})
	if task.LatestUserIntent.Text != "real user request" {
		t.Fatalf("latest = %q", task.LatestUserIntent.Text)
	}
}

func userMsg(order int, text string) capsule.Event {
	return capsule.Event{
		ID:    "u" + strconv.Itoa(order),
		Order: order,
		Actor: capsule.ActorUser,
		Kind:  capsule.KindMessage,
		Blocks: []capsule.Block{{
			Type: capsule.BlockTypeText,
			Text: text,
		}},
		Portability: capsule.PortabilityExact,
	}
}

func toolCall(order int, callID, name, input string) capsule.Event {
	return capsule.Event{
		ID:         "tc" + strconv.Itoa(order),
		Order:      order,
		Actor:      capsule.ActorAssistant,
		Kind:       capsule.KindToolCall,
		NativeName: name,
		CallID:     callID,
		Blocks: []capsule.Block{{
			Type: capsule.BlockTypeToolInput,
			Text: input,
		}},
		Portability: capsule.PortabilityNormalized,
		Reason:      "normalized_tool_call",
	}
}

func toolResult(order int, linkedCallID string, isError bool, text string) capsule.Event {
	return capsule.Event{
		ID:           "tr" + strconv.Itoa(order),
		Order:        order,
		Actor:        capsule.ActorTool,
		Kind:         capsule.KindToolResult,
		LinkedCallID: linkedCallID,
		Blocks: []capsule.Block{{
			Type:    capsule.BlockTypeToolOutput,
			Text:    text,
			IsError: isError,
		}},
		Portability: capsule.PortabilityNormalized,
		Reason:      "normalized_tool_result",
	}
}
