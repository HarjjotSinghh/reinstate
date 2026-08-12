package handoff

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/workspace"
)

// CheckpointInput is the deterministic input to DeriveCheckpoint.
// Changed must be live Git porcelain paths — never transcript claims.
type CheckpointInput struct {
	Events    []capsule.Event
	Workspace workspace.Fingerprint
	Changed   []string // live git porcelain, NOT transcript claims
	// ChangedTruncated reports that Changed is a bounded prefix of the real
	// list. A truncated list cannot contradict a transcript claim, so it
	// suppresses the evidence-conflict marking below.
	ChangedTruncated bool
}

const (
	recentUserMessageLimit = 8

	labelDerivedDeterministic = "derived_deterministic"
	labelTranscriptClaim      = "transcript_claim"

	reasonRequiresOptionalSummarizer     = "requires_optional_summarizer"
	reasonInterruptedNotReplayed         = "interrupted_not_replayed"
	reasonEvidenceConflictsWithWorkspace = "evidence_conflicts_with_workspace"

	truncationMarker = "\n[truncated]"

	nextActionTemplate = "continue the latest user request, given current workspace state"
)

// testRunnerAllowlist is the exact sorted set of recognized test commands.
// Nothing outside this list is classified as a test.
var testRunnerAllowlist = []string{
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

// DeriveCheckpoint builds task state with zero model calls and zero network.
// It implements the architecture plan §6 derivation table exactly.
func DeriveCheckpoint(in CheckpointInput) capsule.Task {
	userTexts := nonMetaUserTexts(in.Events)
	latest := ""
	if len(userTexts) > 0 {
		latest = userTexts[len(userTexts)-1]
	}
	first := ""
	if len(userTexts) > 0 {
		first = userTexts[0]
	}

	recent := userTexts
	if len(recent) > recentUserMessageLimit {
		recent = recent[len(recent)-recentUserMessageLimit:]
	}

	goalText := first
	if first != "" && latest != "" && first != latest {
		goalText = first + "\n\n" + latest
	} else if latest != "" && first == "" {
		goalText = latest
	}
	goalText = boundRunes(goalText, capsule.MaxTaskFieldRunes)

	calls, resultsByCall := indexToolPairs(in.Events)
	completed, pending := classifyToolEvidence(calls, resultsByCall)
	touched, conflict := filesTouchedFromTranscript(calls, in.Changed, changedFilesAreComplete(in))
	tests := deriveTests(calls, resultsByCall)

	changed := append([]string(nil), in.Changed...)

	task := capsule.Task{
		Goal: capsule.TextField{
			Text:        goalText,
			Portability: capsule.PortabilityNormalized,
			Label:       labelDerivedDeterministic,
		},
		LatestUserIntent: capsule.TextField{
			Text:        boundRunes(latest, capsule.MaxTaskFieldRunes),
			Portability: capsule.PortabilityExact,
		},
		RecentUserMessages: capsule.ListField{
			Items:       recent,
			Portability: capsule.PortabilityExact,
		},
		Constraints: capsule.ListField{
			Portability: capsule.PortabilityOmitted,
			Reason:      reasonRequiresOptionalSummarizer,
		},
		Decisions: capsule.ListField{
			Portability: capsule.PortabilityOmitted,
			Reason:      reasonRequiresOptionalSummarizer,
		},
		RejectedApproaches: capsule.ListField{
			Portability: capsule.PortabilityOmitted,
			Reason:      reasonRequiresOptionalSummarizer,
		},
		Completed: capsule.ListField{
			Items:       completed,
			Portability: capsule.PortabilityNormalized,
		},
		Pending: capsule.ListField{
			Items:       pending,
			Portability: capsule.PortabilityOmitted,
			Reason:      reasonInterruptedNotReplayed,
		},
		ChangedFiles: capsule.ListField{
			Items:       changed,
			Portability: capsule.PortabilityExact,
		},
		FilesTouchedPerTranscript: capsule.ListField{
			Items:       touched,
			Portability: capsule.PortabilityReferenced,
			Label:       labelTranscriptClaim,
		},
		Tests: capsule.ListField{
			Items:       tests,
			Portability: capsule.PortabilityReferenced,
		},
		NextAction: capsule.TextField{
			Text:        nextActionTemplate,
			Portability: capsule.PortabilityNormalized,
			Label:       labelDerivedDeterministic,
		},
	}
	if conflict {
		task.FilesTouchedPerTranscript.Reason = reasonEvidenceConflictsWithWorkspace
	}
	return task
}

func nonMetaUserTexts(events []capsule.Event) []string {
	out := make([]string, 0)
	for _, ev := range events {
		if !isNonMetaUserMessage(ev) {
			continue
		}
		text := eventText(ev)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func isNonMetaUserMessage(ev capsule.Event) bool {
	if ev.Actor != capsule.ActorUser {
		return false
	}
	switch ev.Kind {
	case capsule.KindMessage:
		return true
	default:
		return false
	}
}

func eventText(ev capsule.Event) string {
	parts := make([]string, 0, len(ev.Blocks))
	for _, b := range ev.Blocks {
		if b.Type == capsule.BlockTypeText && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

type toolCallView struct {
	Event capsule.Event
	Input string
}

func indexToolPairs(events []capsule.Event) ([]toolCallView, map[string]capsule.Event) {
	calls := make([]toolCallView, 0)
	results := make(map[string]capsule.Event)
	for _, ev := range events {
		switch ev.Kind {
		case capsule.KindToolCall:
			calls = append(calls, toolCallView{
				Event: ev,
				Input: toolInputText(ev),
			})
		case capsule.KindToolResult:
			id := strings.TrimSpace(ev.LinkedCallID)
			if id == "" {
				continue
			}
			if _, exists := results[id]; exists {
				continue // first matching result wins
			}
			results[id] = ev
		}
	}
	return calls, results
}

func toolInputText(ev capsule.Event) string {
	for _, b := range ev.Blocks {
		if b.Type == capsule.BlockTypeToolInput {
			return b.Text
		}
	}
	return eventText(ev)
}

func classifyToolEvidence(calls []toolCallView, results map[string]capsule.Event) (completed, pending []string) {
	completed = make([]string, 0)
	pending = make([]string, 0)
	for _, call := range calls {
		id := strings.TrimSpace(call.Event.CallID)
		line := evidenceLine(call)
		if id == "" {
			// Unmatchable call IDs cannot be proven complete.
			pending = append(pending, line)
			continue
		}
		res, ok := results[id]
		if !ok {
			pending = append(pending, line)
			continue
		}
		if toolResultIsError(res) {
			continue // finished with error: neither completed nor pending
		}
		completed = append(completed, line)
	}
	return completed, pending
}

func evidenceLine(call toolCallView) string {
	name := strings.TrimSpace(call.Event.NativeName)
	if name == "" {
		name = "tool_call"
	}
	id := strings.TrimSpace(call.Event.CallID)
	if id == "" {
		return name
	}
	return name + " call_id=" + id
}

func toolResultIsError(ev capsule.Event) bool {
	for _, b := range ev.Blocks {
		if b.IsError {
			return true
		}
		if b.Meta != nil && (b.Meta["is_error"] == "true" || b.Meta["isError"] == "true") {
			return true
		}
	}
	return false
}

// changedFilesAreComplete reports whether live Git produced a complete
// changed-file list for this handoff.
//
// The truth hierarchy lets current workspace state contradict a transcript
// claim, but only when that state was actually observed. An unavailable,
// uncertain, or bounded observation is missing evidence, not counter-evidence,
// and marking a claim as contradicted on that basis would be its own
// over-claim.
func changedFilesAreComplete(in CheckpointInput) bool {
	if in.ChangedTruncated {
		return false
	}
	git := in.Workspace.Git
	if !git.Available || !git.Repository {
		return false
	}
	tree := git.WorkingTree
	return tree.State != workspace.WorkingTreeUnavailable &&
		!tree.Uncertain && !tree.CountsTruncated && tree.ChangedOmitted == 0
}

func filesTouchedFromTranscript(
	calls []toolCallView,
	changed []string,
	changedIsComplete bool,
) (paths []string, conflict bool) {
	seen := make(map[string]struct{})
	for _, call := range calls {
		collectFileRefs(call.Input, seen)
		for _, b := range call.Event.Blocks {
			if b.Path != "" {
				addFileRef(b.Path, seen)
			}
		}
	}
	paths = make([]string, 0, len(seen))
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	if !changedIsComplete {
		return paths, false
	}
	changedSet := make(map[string]struct{}, len(changed))
	for _, c := range changed {
		changedSet[normalizePathKey(c)] = struct{}{}
	}
	for _, p := range paths {
		if _, ok := changedSet[normalizePathKey(p)]; !ok {
			conflict = true
			break
		}
	}
	return paths, conflict
}

func collectFileRefs(input string, files map[string]struct{}) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	var value any
	if err := json.Unmarshal([]byte(input), &value); err != nil {
		return
	}
	collectStructuredFileValue(value, files)
}

func collectStructuredFileValue(value any, files map[string]struct{}) {
	if len(files) >= capsule.MaxFileReferences || value == nil {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isFileField(key) {
				switch fileValue := nested.(type) {
				case string:
					addFileRef(fileValue, files)
				case []any:
					for _, item := range fileValue {
						if path, ok := item.(string); ok {
							addFileRef(path, files)
						}
					}
				}
			}
			collectStructuredFileValue(nested, files)
		}
	case []any:
		for _, nested := range typed {
			collectStructuredFileValue(nested, files)
		}
	}
}

// isFileField defers to the capsule so the keys whose values are lifted into
// task.files_touched_per_transcript are exactly the keys the capsule validates
// as paths inside tool arguments.
func isFileField(key string) bool {
	return capsule.IsPathFieldName(key)
}

func addFileRef(value string, files map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") || len(files) >= capsule.MaxFileReferences {
		return
	}
	files[value] = struct{}{}
}

func normalizePathKey(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, `\`, "/")
	return p
}

func deriveTests(calls []toolCallView, results map[string]capsule.Event) []string {
	var last string
	for _, call := range calls {
		cmd := extractCommand(call.Input)
		runner, ok := matchTestRunner(cmd)
		if !ok {
			continue
		}
		state := "exit=unknown"
		id := strings.TrimSpace(call.Event.CallID)
		if id != "" {
			if res, ok := results[id]; ok {
				if toolResultIsError(res) {
					state = "exit=error"
				} else {
					state = "exit=0"
				}
			} else {
				state = "exit=interrupted"
			}
		}
		last = runner + " · " + cmd + " · " + state
	}
	if last == "" {
		return nil
	}
	return []string{last}
}

func extractCommand(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(input), &m); err == nil {
		for _, key := range []string{"command", "cmd", "script"} {
			if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return input
}

func matchTestRunner(cmd string) (string, bool) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", false
	}
	for _, runner := range testRunnerAllowlist {
		if cmd == runner || strings.HasPrefix(cmd, runner+" ") || strings.HasPrefix(cmd, runner+"\t") {
			return runner, true
		}
	}
	return "", false
}

func boundRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	markerRunes := utf8.RuneCountInString(truncationMarker)
	if maxRunes < markerRunes {
		return string([]rune(truncationMarker)[:maxRunes])
	}
	keep := maxRunes - markerRunes
	runes := []rune(s)
	if keep > len(runes) {
		keep = len(runes)
	}
	return string(runes[:keep]) + truncationMarker
}
