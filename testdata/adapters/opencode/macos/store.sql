-- Synthetic OpenCode session store seed. All identities are synthetic.
-- Hydrate with: sqlite3 opencode.db < store.sql
CREATE TABLE project (
  id TEXT PRIMARY KEY,
  worktree TEXT NOT NULL,
  name TEXT,
  sandboxes TEXT NOT NULL DEFAULT '[]',
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL
);
CREATE TABLE session (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  slug TEXT NOT NULL,
  directory TEXT NOT NULL,
  path TEXT,
  title TEXT NOT NULL,
  version TEXT NOT NULL,
  agent TEXT,
  model TEXT,
  metadata TEXT,
  time_created INTEGER NOT NULL,
  time_updated INTEGER NOT NULL
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
CREATE TABLE credential (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  value TEXT NOT NULL
);
INSERT INTO project VALUES ('proj_fixture','/Users/fixture-user/code/demo','demo','[]',1755900000000,1755900060000);
INSERT INTO session VALUES ('ses_fixture001','proj_fixture','synthetic-fixture-session','/Users/fixture-user/code/demo',NULL,'Synthetic OpenCode fixture session','1.18.21','build','synthetic/fixture-model',NULL,1755900000000,1755900060000);
INSERT INTO message VALUES ('msg_fixtureasst001','ses_fixture001',1755900001000,1755900002000,'{"role":"assistant","time":{"created":1755900001000,"completed":1755900002000},"parentID":"msg_fixtureuser001","agent":"build","path":{"cwd":"/Users/fixture-user/code/demo","root":"/Users/fixture-user/code/demo"}}');
INSERT INTO message VALUES ('msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"role":"user","time":{"created":1755900000000},"agent":"build","model":{"providerID":"synthetic","modelID":"fixture-model"}}');
INSERT INTO part VALUES ('prt_fixtureasst001','msg_fixtureasst001','ses_fixture001',1755900002000,1755900002000,'{"type":"text","text":"Synthetic fixture reply."}');
INSERT INTO part VALUES ('prt_fixtureuser001','msg_fixtureuser001','ses_fixture001',1755900000000,1755900000000,'{"type":"text","text":"Synthetic OpenCode fixture request about the demo workspace"}');
INSERT INTO credential VALUES ('cred_fixture','synthetic','synthetic-not-a-real-token');
