package fixture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyntheticFiles is the deterministic adapter fixture corpus.
var SyntheticFiles = map[string]string{
	"claude/macos/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"/Users/fixture-user/code/demo"}
{"type":"user","message":{"content":"Synthetic Claude fixture request"}}
`,
	"claude/windows/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"C:\\Users\\fixture-user\\code\\demo"}
{"type":"user","message":{"content":"Synthetic Claude Windows fixture request"}}
`,
	"claude/wsl/projects/fixture-project/session-syn-001.jsonl": `{"type":"meta","sessionId":"session-syn-001","cwd":"/home/fixture-user/code/demo"}
{"type":"user","message":{"content":"Synthetic Claude WSL2 fixture request"}}
`,
	"codex/macos/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"/Users/fixture-user/code/demo"}}
{"type":"message","role":"user","content":"Synthetic Codex fixture request"}
`,
	"codex/windows/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"C:\\Users\\fixture-user\\code\\demo"}}
{"type":"message","role":"user","content":"Synthetic Codex Windows fixture request"}
`,
	"codex/wsl/sessions/rollout-syn-001.jsonl": `{"type":"session_meta","payload":{"id":"rollout-syn-001","cwd":"/home/fixture-user/code/demo"}}
{"type":"message","role":"user","content":"Synthetic Codex WSL2 fixture request"}
`,
	// OpenCode keeps every session in one embedded SQLite store, so its adapter
	// fixture is a deterministic SQL seed rather than a session file. Tests
	// hydrate an opencode.db from it; the credential row proves the export path
	// never touches that table. See testdata/adapters/README.md.
	"opencode/macos/store.sql":   openCodeStoreSQL(openCodeMacOSData),
	"opencode/windows/store.sql": openCodeStoreSQL(openCodeWindowsData),
}

// openCodeStoreSchema mirrors the OpenCode 1.18.21 store as measured on a real
// install (docs/testing/results/2026-08-23-macos-opencode-t5-journey.md): the
// full session, project, message and part tables the sync adapter reads and
// writes, with every column the vendor's migrations create, plus the project_directory,
// workspace, session_message, credential and account tables that share the
// database. The extra columns are deliberate — the reader names its columns in
// SQL and must tolerate a store wider than the ones it selects; a trimmed schema
// would hide exactly that. The credential and account tables are present only so
// the exclusion contract can be tested — the adapter never selects them.
const openCodeStoreSchema = `CREATE TABLE project (
  id TEXT PRIMARY KEY,
  worktree TEXT NOT NULL,
  vcs TEXT,
  name TEXT,
  icon_url TEXT,
  icon_url_override TEXT,
  icon_color TEXT,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  time_initialized INTEGER,
  sandboxes TEXT NOT NULL,
  commands TEXT
);
CREATE TABLE project_directory (
  project_id TEXT NOT NULL,
  directory TEXT NOT NULL,
  type TEXT,
  strategy TEXT,
  time_created INTEGER NOT NULL,
  PRIMARY KEY(project_id, directory)
);
CREATE TABLE workspace (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  name TEXT DEFAULT '' NOT NULL,
  branch TEXT,
  directory TEXT,
  extra TEXT,
  project_id TEXT NOT NULL,
  time_used INTEGER NOT NULL
);
CREATE TABLE session (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  workspace_id TEXT,
  parent_id TEXT,
  slug TEXT NOT NULL,
  directory TEXT NOT NULL,
  path TEXT,
  title TEXT NOT NULL,
  version TEXT NOT NULL,
  share_url TEXT,
  summary_additions INTEGER,
  summary_deletions INTEGER,
  summary_files INTEGER,
  summary_diffs TEXT,
  metadata TEXT,
  cost REAL DEFAULT 0 NOT NULL,
  tokens_input INTEGER DEFAULT 0 NOT NULL,
  tokens_output INTEGER DEFAULT 0 NOT NULL,
  tokens_reasoning INTEGER DEFAULT 0 NOT NULL,
  tokens_cache_read INTEGER DEFAULT 0 NOT NULL,
  tokens_cache_write INTEGER DEFAULT 0 NOT NULL,
  revert TEXT,
  permission TEXT,
  agent TEXT,
  model TEXT,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  time_compacting INTEGER,
  time_archived INTEGER
);
CREATE TABLE message (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  data TEXT NOT NULL
);
CREATE TABLE part (
  id TEXT PRIMARY KEY,
  message_id TEXT NOT NULL,
  session_id TEXT NOT NULL,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  data TEXT NOT NULL
);
CREATE TABLE session_message (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  type TEXT NOT NULL,
  seq INTEGER NOT NULL,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL,
  data TEXT NOT NULL
);
CREATE TABLE credential (
  id TEXT PRIMARY KEY,
  integration_id TEXT,
  label TEXT NOT NULL,
  value TEXT NOT NULL,
  connector_id TEXT,
  method_id TEXT,
  active INTEGER,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL
);
CREATE TABLE account (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL,
  url TEXT NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  token_expiry INTEGER,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL
);
CREATE TABLE migration (id TEXT PRIMARY KEY, time_completed INTEGER NOT NULL);
CREATE INDEX message_session_time_created_id_idx ON message (session_id,time_created,id);
CREATE INDEX part_message_id_id_idx ON part (message_id,id);
CREATE INDEX part_session_idx ON part (session_id);
CREATE INDEX session_project_idx ON session (project_id);
CREATE INDEX session_workspace_idx ON session (workspace_id);
CREATE INDEX session_parent_idx ON session (parent_id);
`

// openCodeMacOSData seeds one session whose paths are macOS shaped. Row inserts
// name their columns so the seed stays valid as the schema grows.
const openCodeMacOSData = `INSERT INTO migration (id, time_completed) VALUES ('20260622202450_simplify_session_input',1755900000000);
INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','/Users/fixture-user/code/demo','git','demo',1755900000000,1755900060000,'[]');
INSERT INTO project_directory (project_id, directory, type, strategy, time_created) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','/Users/fixture-user/code/demo','worktree','git',1755900000000);
INSERT INTO session (id, project_id, slug, directory, path, title, version, agent, model, metadata, time_created, time_updated) VALUES ('ses_fixture001','a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','synthetic-fixture-session','/Users/fixture-user/code/demo',NULL,'Synthetic OpenCode fixture session','1.18.21','build','synthetic/fixture-model',NULL,1755900000000,1755900060000);
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureasst001','ses_fixture001',1755900001000,1755900002000,'{"role":"assistant","time":{"created":1755900001000,"completed":1755900002000},"parentID":"msg_fixtureuser001","agent":"build","path":{"cwd":"/Users/fixture-user/code/demo","root":"/Users/fixture-user/code/demo"}}');
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"role":"user","time":{"created":1755900000000},"agent":"build","model":{"providerID":"synthetic","modelID":"fixture-model"}}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureasst001','msg_fixtureasst001','ses_fixture001',1755900002000,1755900002000,'{"type":"text","text":"Synthetic fixture reply."}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureuser001','msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"type":"text","text":"Synthetic OpenCode fixture request about the demo workspace"}');
INSERT INTO credential (id, label, value, active, time_created, time_updated) VALUES ('cred_fixture','synthetic','synthetic-not-a-real-token',1,1755900000000,1755900000000);
INSERT INTO account (id, email, url, access_token, refresh_token, time_created, time_updated) VALUES ('acct_fixture','synthetic-account-not-an-address','https://example.invalid','synthetic-not-a-real-access-token','synthetic-not-a-real-refresh-token',1755900000000,1755900000000);
`

// openCodeWindowsData seeds the same session with Windows shaped paths so the
// macOS-to-Windows remapping has both ends.
const openCodeWindowsData = `INSERT INTO migration (id, time_completed) VALUES ('20260622202450_simplify_session_input',1755900000000);
INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','C:\Users\fixture-user\code\demo','git','demo',1755900000000,1755900060000,'[]');
INSERT INTO project_directory (project_id, directory, type, strategy, time_created) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','C:\Users\fixture-user\code\demo','worktree','git',1755900000000);
INSERT INTO session (id, project_id, slug, directory, path, title, version, agent, model, metadata, time_created, time_updated) VALUES ('ses_fixture001','a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','synthetic-fixture-session','C:\Users\fixture-user\code\demo',NULL,'Synthetic OpenCode fixture session','1.18.21','build','synthetic/fixture-model',NULL,1755900000000,1755900060000);
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureasst001','ses_fixture001',1755900001000,1755900002000,'{"role":"assistant","time":{"created":1755900001000,"completed":1755900002000},"parentID":"msg_fixtureuser001","agent":"build","path":{"cwd":"C:\\Users\\fixture-user\\code\\demo","root":"C:\\Users\\fixture-user\\code\\demo"}}');
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"role":"user","time":{"created":1755900000000},"agent":"build","model":{"providerID":"synthetic","modelID":"fixture-model"}}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureasst001','msg_fixtureasst001','ses_fixture001',1755900002000,1755900002000,'{"type":"text","text":"Synthetic fixture reply."}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureuser001','msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"type":"text","text":"Synthetic OpenCode fixture request about the demo workspace"}');
INSERT INTO credential (id, label, value, active, time_created, time_updated) VALUES ('cred_fixture','synthetic','synthetic-not-a-real-token',1,1755900000000,1755900000000);
INSERT INTO account (id, email, url, access_token, refresh_token, time_created, time_updated) VALUES ('acct_fixture','synthetic-account-not-an-address','https://example.invalid','synthetic-not-a-real-access-token','synthetic-not-a-real-refresh-token',1755900000000,1755900000000);
`

func openCodeStoreSQL(data string) string {
	return "-- Synthetic OpenCode session store seed. All identities are synthetic.\n" +
		"-- Hydrate with: sqlite3 opencode.db < store.sql\n" +
		openCodeStoreSchema + data
}

// HandoffTreeDirs are §10 directories that must exist for corpus completeness.
// adversarial/* content is owned by WP-24; golden/capsule content by WP-25.
var HandoffTreeDirs = []string{
	"claude/long-history",
	"claude/compaction",
	"claude/parallel-tools",
	"claude/subagents",
	"claude/attachments",
	"claude/absolute-paths",
	"claude/partial-final-record",
	"claude/unknown-records",
	"claude/os-roots/macos",
	"claude/os-roots/windows",
	"claude/os-roots/wsl",
	"codex/long-history",
	"codex/forks",
	"codex/parallel-tools",
	"codex/reasoning-items",
	"codex/absolute-paths",
	"codex/partial-final-record",
	"codex/unknown-records",
	"codex/os-roots/macos",
	"codex/os-roots/windows",
	"codex/os-roots/wsl",
	"gemini/rewind",
	"gemini/legacy-json",
	"gemini/jsonl",
	"opencode/storage",
	"opencode/metadata-only",
	"grok/basic",
	"grok/compacted",
	"adversarial/prompt-injection",
	"adversarial/secret-leakage",
	"adversarial/fence-breakout",
	"adversarial/oversized",
	"golden/capsule",
	"golden/projection",
}

// HandoffSyntheticFiles are small deterministic handoff fixtures regenerated by
// GenerateHandoff. Large class corpora from WP-06..10 stay committed alongside.
//
// Claude fixtures deliberately contain no <root>/version file: Claude Code does
// not write one, and a fixture that invents it hides how a real installation
// behaves. Version handling is covered where it belongs, by resolving an
// installed executable (internal/transcript/compat_test.go).
var HandoffSyntheticFiles = map[string]string{
	"claude/os-roots/macos/projects/-Users-fixture-user-code-demo/session-syn-001.jsonl": `{"type":"user","uuid":"os-macos-u1","sessionId":"00000000-0000-4000-8000-00000000os01","cwd":"/Users/fixture-user/code/demo","timestamp":"2026-08-01T12:00:00Z","message":{"role":"user","content":"Synthetic Claude macOS OS-root fixture"}}
{"type":"assistant","uuid":"os-macos-a1","sessionId":"00000000-0000-4000-8000-00000000os01","timestamp":"2026-08-01T12:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"Synthetic Claude macOS OS-root reply"}]}}
`,
	"claude/os-roots/windows/projects/C--Users-fixture-user-code-demo/session-syn-001.jsonl": `{"type":"user","uuid":"os-win-u1","sessionId":"00000000-0000-4000-8000-00000000os02","cwd":"C:\\Users\\fixture-user\\code\\demo","timestamp":"2026-08-01T12:00:00Z","message":{"role":"user","content":"Synthetic Claude Windows OS-root fixture"}}
{"type":"assistant","uuid":"os-win-a1","sessionId":"00000000-0000-4000-8000-00000000os02","timestamp":"2026-08-01T12:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"Synthetic Claude Windows OS-root reply"}]}}
`,
	"claude/os-roots/wsl/projects/-home-fixture-user-code-demo/session-syn-001.jsonl": `{"type":"user","uuid":"os-wsl-u1","sessionId":"00000000-0000-4000-8000-00000000os03","cwd":"/home/fixture-user/code/demo","timestamp":"2026-08-01T12:00:00Z","message":{"role":"user","content":"Synthetic Claude WSL OS-root fixture"}}
{"type":"assistant","uuid":"os-wsl-a1","sessionId":"00000000-0000-4000-8000-00000000os03","timestamp":"2026-08-01T12:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"Synthetic Claude WSL OS-root reply"}]}}
`,
	"codex/os-roots/macos/rollout-2026-08-01T12-00-00-00000000-0000-4000-8000-00000000os11.jsonl": `{"timestamp":"2026-08-01T12:00:00Z","type":"session_meta","payload":{"id":"00000000-0000-4000-8000-00000000os11","cwd":"/Users/fixture-user/code/demo","git":{"branch":"fixture/macos"}}}
{"timestamp":"2026-08-01T12:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"Synthetic Codex macOS OS-root fixture"}}
{"timestamp":"2026-08-01T12:00:02Z","type":"event_msg","payload":{"type":"agent_message","message":"Synthetic Codex macOS OS-root reply"}}
`,
	"codex/os-roots/windows/rollout-2026-08-01T12-00-00-00000000-0000-4000-8000-00000000os12.jsonl": `{"timestamp":"2026-08-01T12:00:00Z","type":"session_meta","payload":{"id":"00000000-0000-4000-8000-00000000os12","cwd":"C:\\Users\\fixture-user\\code\\demo","git":{"branch":"fixture/windows"}}}
{"timestamp":"2026-08-01T12:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"Synthetic Codex Windows OS-root fixture"}}
{"timestamp":"2026-08-01T12:00:02Z","type":"event_msg","payload":{"type":"agent_message","message":"Synthetic Codex Windows OS-root reply"}}
`,
	"codex/os-roots/wsl/rollout-2026-08-01T12-00-00-00000000-0000-4000-8000-00000000os13.jsonl": `{"timestamp":"2026-08-01T12:00:00Z","type":"session_meta","payload":{"id":"00000000-0000-4000-8000-00000000os13","cwd":"/home/fixture-user/code/demo","git":{"branch":"fixture/wsl"}}}
{"timestamp":"2026-08-01T12:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"Synthetic Codex WSL OS-root fixture"}}
{"timestamp":"2026-08-01T12:00:02Z","type":"event_msg","payload":{"type":"agent_message","message":"Synthetic Codex WSL OS-root reply"}}
`,
	"adversarial/prompt-injection/.gitkeep": "",
	"adversarial/secret-leakage/.gitkeep":   "",
	"adversarial/fence-breakout/.gitkeep":   "",
	"adversarial/oversized/.gitkeep":        "",
	"adversarial/README.md": `# Adversarial handoff fixtures

Owned by WP-24. Placeholders only in WP-23 so the §10 tree is complete.
`,
	"golden/capsule/.gitkeep": "",
	"golden/capsule/README.md": `# Golden capsule fixtures

Owned by WP-25 (byte-exact capsule JSON goldens). Placeholder from WP-23.
`,
}

// Generate writes the deterministic synthetic adapter corpus below root.
func Generate(root string) error {
	for relative, content := range SyntheticFiles {
		if err := writeRelative(root, relative, content); err != nil {
			return err
		}
	}
	return nil
}

// GenerateHandoff writes the regenerable handoff corpus below root.
// It creates the §10 tree, OS-root fixtures, long-history (200+ turns),
// and WP-24/WP-25 directory placeholders. Existing class fixtures from
// WP-06..10 are not deleted; callers should merge into testdata/handoff.
func GenerateHandoff(root string) error {
	for _, dir := range HandoffTreeDirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o700); err != nil {
			return err
		}
	}
	for relative, content := range HandoffSyntheticFiles {
		if err := writeRelative(root, relative, content); err != nil {
			return err
		}
	}
	if err := writeRelative(root, "claude/long-history/projects/-Users-fixture-user-code-demo/session-syn-001.jsonl", ClaudeLongHistoryJSONL()); err != nil {
		return err
	}
	if err := writeRelative(root, "codex/long-history/rollout-2026-08-01T10-00-00-00000000-0000-4000-8000-00000000aa01.jsonl", CodexLongHistoryJSONL()); err != nil {
		return err
	}
	return nil
}

func writeRelative(root, relative, content string) error {
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

const longHistoryTurns = 200

// ClaudeLongHistoryJSONL returns a deterministic 200-turn Claude JSONL corpus.
func ClaudeLongHistoryJSONL() string {
	const sessionID = "00000000-0000-4000-8000-000000000001"
	const cwd = "/Users/fixture-user/code/demo"
	base := time.Date(2026, 8, 1, 16, 0, 0, 0, time.UTC)
	var b strings.Builder
	for i := 1; i <= longHistoryTurns; i++ {
		ts := base.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		fmt.Fprintf(&b, `{"type":"user","uuid":"u%04d","sessionId":%q,"cwd":%q,"timestamp":%q,"message":{"role":"user","content":"Synthetic long-history user turn %d"}}`+"\n", i, sessionID, cwd, ts, i)
		fmt.Fprintf(&b, `{"type":"assistant","uuid":"a%04d","sessionId":%q,"timestamp":%q,"message":{"role":"assistant","content":[{"type":"text","text":"Synthetic long-history assistant turn %d"}]}}`+"\n", i, sessionID, ts, i)
	}
	return b.String()
}

// CodexLongHistoryJSONL returns a deterministic 200-turn Codex rollout JSONL.
// Turns 1–5 keep dual-representation dedup markers used by WP-07 tests;
// turns 6–200 are plain event_msg pairs.
func CodexLongHistoryJSONL() string {
	const sessionID = "00000000-0000-4000-8000-00000000aa01"
	const cwd = `/Users/fixture-user/code/demo`
	var b strings.Builder
	fmt.Fprintf(&b, `{"timestamp":"2026-08-01T10:00:00Z","type":"session_meta","payload":{"id":%q,"cwd":%q,"git":{"branch":"fixture/main"}}}`+"\n", sessionID, cwd)
	b.WriteString(`{"timestamp":"2026-08-01T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-01 fix the flaky test"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:01Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"ENVIRONMENT_DUMP_SHOULD_DROP"}]}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:02Z","type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-01 reviewing the failing assertion"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"DUPLICATE_ASSIST_SHOULD_DROP"}]}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:03Z","type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-02 also check race detector"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:03Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"ENVIRONMENT_DUMP_SHOULD_DROP_2"}]}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:04Z","type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-02 race looks clean after the wait group fix"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:05Z","type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-03 ship a regression fixture"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:06Z","type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-03 added a synthetic table-driven case"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:07Z","type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-04 document the dedup rule"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:08Z","type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-04 prefer event_msg when both exist"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:09Z","type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-05 done"}}` + "\n")
	b.WriteString(`{"timestamp":"2026-08-01T10:00:10Z","type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-05 closing out"}}` + "\n")

	next := time.Date(2026, 8, 1, 10, 0, 11, 0, time.UTC)
	for i := 6; i <= longHistoryTurns; i++ {
		userTS := next.Format(time.RFC3339)
		next = next.Add(time.Second)
		assistTS := next.Format(time.RFC3339)
		next = next.Add(time.Second)
		fmt.Fprintf(&b, `{"timestamp":%q,"type":"event_msg","payload":{"type":"user_message","message":"LONG-TURN-%02d synthetic continuation"}}`+"\n", userTS, i)
		fmt.Fprintf(&b, `{"timestamp":%q,"type":"event_msg","payload":{"type":"agent_message","message":"LONG-ASSIST-%02d synthetic continuation"}}`+"\n", assistTS, i)
	}
	return b.String()
}
