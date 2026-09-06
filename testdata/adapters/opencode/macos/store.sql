-- Synthetic OpenCode session store seed. All identities are synthetic.
-- Hydrate with: sqlite3 opencode.db < store.sql
CREATE TABLE project (
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
INSERT INTO migration (id, time_completed) VALUES ('20260622202450_simplify_session_input',1755900000000);
INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','/Users/fixture-user/code/demo','git','demo',1755900000000,1755900060000,'[]');
INSERT INTO project_directory (project_id, directory, type, strategy, time_created) VALUES ('a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','/Users/fixture-user/code/demo','worktree','git',1755900000000);
INSERT INTO session (id, project_id, slug, directory, path, title, version, agent, model, metadata, time_created, time_updated) VALUES ('ses_fixture001','a1b2c3d4e5f60718293a4b5c6d7e8f9012345678','synthetic-fixture-session','/Users/fixture-user/code/demo',NULL,'Synthetic OpenCode fixture session','1.18.21','build','synthetic/fixture-model',NULL,1755900000000,1755900060000);
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureasst001','ses_fixture001',1755900001000,1755900002000,'{"role":"assistant","time":{"created":1755900001000,"completed":1755900002000},"parentID":"msg_fixtureuser001","agent":"build","path":{"cwd":"/Users/fixture-user/code/demo","root":"/Users/fixture-user/code/demo"}}');
INSERT INTO message (id, session_id, time_created, time_updated, data) VALUES ('msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"role":"user","time":{"created":1755900000000},"agent":"build","model":{"providerID":"synthetic","modelID":"fixture-model"}}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureasst001','msg_fixtureasst001','ses_fixture001',1755900002000,1755900002000,'{"type":"text","text":"Synthetic fixture reply."}');
INSERT INTO part (id, message_id, session_id, time_created, time_updated, data) VALUES ('prt_fixtureuser001','msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"type":"text","text":"Synthetic OpenCode fixture request about the demo workspace"}');
INSERT INTO credential (id, label, value, active, time_created, time_updated) VALUES ('cred_fixture','synthetic','synthetic-not-a-real-token',1,1755900000000,1755900000000);
INSERT INTO account (id, email, url, access_token, refresh_token, time_created, time_updated) VALUES ('acct_fixture','synthetic-account-not-an-address','https://example.invalid','synthetic-not-a-real-access-token','synthetic-not-a-real-refresh-token',1755900000000,1755900000000);
