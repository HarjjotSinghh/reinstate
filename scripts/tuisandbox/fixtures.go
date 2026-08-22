// Copyright 2026 Harjot Singh Rana. Licensed under Apache-2.0.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// writeClaude plants a Claude Code session under the sandbox home. The project
// directory name is the workspace path with every separator and colon replaced
// by a dash, which is the layout Claude Code itself uses.
//
// Two fields decide what the environment report can say about this row:
//
//   - "cwd" becomes record.Workspace, so it is what `workspace.available` and
//     every git.* check is run against. Pointing it at a directory the
//     generator never creates is how the missing-workspace block is produced.
//   - "gitBranch" becomes RecordedEnvironment.Branch with provenance
//     "claude.event.gitBranch". That is the ONLY environment field the Claude
//     reader records — there is no recorded HEAD and no recorded repository, so
//     `git.head` and `git.repository` can never be anything but info on a
//     Claude row. Those two warnings are structurally Codex-only.
//
// The "summary" event supplies the title. Without one the index falls back to
// the session identifier, which is exactly the unreadable listing the switcher
// exists to replace.
func writeClaude(home, workspace string, s session, when time.Time) error {
	encoded := encodeClaudeProject(workspace)
	dir := filepath.Join(home, ".claude", "projects", encoded)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	summary := fmt.Sprintf(`{"type":"summary","summary":%q,"leafUuid":%q}`, s.title, s.id)
	line := fmt.Sprintf(
		`{"type":"user","sessionId":%q,"cwd":%q,"gitBranch":%q,"timestamp":%q,"message":{"role":"user","content":%q}}`,
		s.id, filepath.ToSlash(workspace), s.branch, when.Format(time.RFC3339), s.title,
	)
	path := filepath.Join(dir, s.id+".jsonl")
	if err := os.WriteFile(path, []byte(summary+"\n"+line+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chtimes(path, when, when)
}

func encodeClaudeProject(workspace string) string {
	normalized := filepath.ToSlash(workspace)
	replacer := strings.NewReplacer("/", "-", ":", "-", "\\", "-", " ", "-", ".", "-")
	return replacer.Replace(normalized)
}

// writeCodex plants a Codex rollout file under the sandbox home.
//
// The session_meta payload is where all three Codex-only environment levers
// live, and each maps to exactly one check:
//
//   - git.branch      -> RecordedEnvironment.Branch      -> check git.branch
//   - git.commit_hash -> RecordedEnvironment.GitHead     -> check git.head
//   - git.repository_url -> RecordedEnvironment.RepositoryID -> check git.repository
//
// A recorded HEAD that disagrees with the live one is a WARNING. A recorded
// repository that disagrees is a BLOCK (exit 7) — repository identity is the
// one recorded field Reinstate refuses to launch through.
//
// "title" is what the index reads for the display title, and only from a
// session_meta / metadata / summary event. The previous generator wrote
// "instructions", which is ignored, so four of its eight rows rendered as raw
// UUIDs in the switcher. The user message below becomes PromptPreview and
// SearchText, never the title.
func writeCodex(home, workspace string, s session, when time.Time) error {
	dir := filepath.Join(home, ".codex", "sessions")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	gitFields := []string{fmt.Sprintf(`"branch":%q`, s.branch)}
	if s.codexHead != "" {
		gitFields = append(gitFields, fmt.Sprintf(`"commit_hash":%q`, s.codexHead))
	}
	if s.repoURL != "" {
		gitFields = append(gitFields, fmt.Sprintf(`"repository_url":%q`, s.repoURL))
	}
	name := fmt.Sprintf("rollout-%s-%s.jsonl", when.Format("2006-01-02T15-04-05"), s.id)
	meta := fmt.Sprintf(
		`{"type":"session_meta","payload":{"id":%q,"timestamp":%q,"cwd":%q,"title":%q,"git":{%s}}}`,
		s.id, when.Format(time.RFC3339), filepath.ToSlash(workspace), s.title, strings.Join(gitFields, ","),
	)
	turn := fmt.Sprintf(
		`{"type":"event_msg","payload":{"type":"user_message","message":%q}}`,
		s.title,
	)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(meta+"\n"+turn+"\n"), 0o600); err != nil {
		return err
	}
	return os.Chtimes(path, when, when)
}

// writeGrok plants a Grok Build session. This row exists to demonstrate the
// OTHER way a session becomes CANNOT RESUME: the readiness prober short-circuits
// on record.ReadOnlyReason / !record.CanResume and never runs preflight at all.
// No workspace probe, no git, no vendor binary — the verdict is a property of
// the record, not of the machine. The Grok source hardcodes CanResume:false,
// CanFork:false, ReadOnlyReason:"Grok Build sessions are source-only in
// Phase 4".
//
// On-disk shape, mirroring testdata/sessionindex/grok/macos:
//
//	<home>/.grok/sessions/<percent-encoded workspace>/<session-uuid>/summary.json
//	<home>/.grok/sessions/<percent-encoded workspace>/<session-uuid>/chat_history.jsonl
//
// Three constraints that silently delete the row if broken:
//
//   - <home>/.grok/sessions must exist; "sessions" is the marker directory the
//     agent root is discovered by.
//   - chat_format_version must be absent, 0, or 1. Any other value makes the
//     session unreadable and it vanishes from the listing with no error.
//   - info.cwd is mandatory in practice. The encoded directory name is only a
//     fallback, and the decoder is url.PathUnescape, which does not turn "+"
//     back into a space — so the directory name is percent-encoded to match the
//     shipped fixture, and info.cwd is always written anyway.
//
// GROK_HOME is deliberately NOT exported by printEnv: the source already falls
// back to $HOME/.grok, and HOME is the sandbox.
func writeGrok(home, workspace string, s session, when time.Time) error {
	dir := filepath.Join(home, ".grok", "sessions", percentEncode(filepath.ToSlash(workspace)), s.id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	stamp := when.Format(time.RFC3339)
	summary := fmt.Sprintf(`{
  "info": {
    "id": %q,
    "cwd": %q
  },
  "session_summary": %q,
  "created_at": %q,
  "updated_at": %q,
  "num_messages": 6,
  "num_chat_messages": 5,
  "chat_format_version": 1
}
`, s.id, filepath.ToSlash(workspace), s.title, stamp, stamp)
	history := fmt.Sprintf(`{"role":"user","content":%q}
{"role":"assistant","content":"Synthetic bench transcript. Nothing here is real."}
`, s.title)
	summaryPath := filepath.Join(dir, "summary.json")
	if err := os.WriteFile(summaryPath, []byte(summary), 0o600); err != nil {
		return err
	}
	historyPath := filepath.Join(dir, "chat_history.jsonl")
	if err := os.WriteFile(historyPath, []byte(history), 0o600); err != nil {
		return err
	}
	for _, path := range []string{historyPath, summaryPath} {
		if err := os.Chtimes(path, when, when); err != nil {
			return err
		}
	}
	return nil
}

// percentEncode escapes everything outside the RFC 3986 unreserved set, which
// is what the shipped Grok fixtures use ("%2F" for "/", "%5C" for "\", "%3A"
// for ":", "%20" for space). net/url's PathEscape leaves ":" and "@" alone, so
// it would not round-trip a Windows drive letter.
func percentEncode(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			out.WriteByte(c)
		default:
			fmt.Fprintf(&out, "%%%02X", c)
		}
	}
	return out.String()
}
