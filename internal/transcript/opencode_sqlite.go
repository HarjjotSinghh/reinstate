package transcript

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"
	"github.com/HarjjotSinghh/reinstate/internal/vendorsqlite"

	_ "modernc.org/sqlite"
)

// OpenCodeDatabaseName is the embedded session store inside the data root.
// Exported so the handoff destination resolves the same file this reader does,
// rather than carrying a second copy of the name.
const OpenCodeDatabaseName = "opencode.db"

// maxOpenCodeMessages bounds one session's replay.
const maxOpenCodeMessages = 20000

// databasePath returns the embedded store beside the storage tree, or "" when
// the data root cannot be resolved.
func (r *OpenCodeReader) databasePath() string {
	data := strings.TrimSpace(r.DataRoot)
	if data == "" {
		resolved, err := ResolveOpenCodeDataRoot(r.Getenv, r.Home)
		if err != nil {
			return ""
		}
		data = resolved
	}
	if data == "" {
		return ""
	}
	return filepath.Join(data, OpenCodeDatabaseName)
}

// probeDatabaseSession reports whether the embedded store directly answers
// compatibility for sessionID: true when a readable opencode.db contains the
// session, false when the store is absent, unreadable, or does not have it
// (including when DataRoot/XDG_DATA_HOME cannot be resolved at all). Probe
// uses this before falling back to the vendor CLI, so a SQLite-only install
// with the session present never has to shell out to `opencode session list`
// just to answer a compatibility check the store itself can answer — see the
// comment on snapshotDatabase for why that shell-out is also a correctness
// hazard, not just an unnecessary process launch.
func (r *OpenCodeReader) probeDatabaseSession(sessionID string) bool {
	path := r.databasePath()
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return false
	}
	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = handle.Close() }()
	return openCodeDatabaseHasSession(handle.DB, sessionID)
}

// snapshotDatabase freezes the target session's own rows in the embedded
// store as the boundary — never the shared file the store lives in.
//
// opencode.db holds every session, and OpenCode's own CLI rewrites parts of it
// as a side effect of commands as read-only as `session list` (bookkeeping in
// tables this reader never reads, unrelated to any one session's content).
// Hashing the whole file conflated that churn — and any other session's
// activity in the same store — with this session's boundary, so two dry-runs
// over an unchanged session produced different digests and therefore
// different capsule identities. The boundary digest is scoped to exactly the
// session, message, and part rows this session owns, ordered by id, matching
// the rows Parse turns into events; a change to those rows is a real edit, a
// change to some other table or session is not this session's boundary.
//
// A row-structured store has no "last complete record" the way an append-only
// JSONL transcript does: SQLite either opens and reads consistently or it does
// not. The boundary is therefore the whole owned-row set, its size and its
// digest, and is never partial. That is the honest analogue of a byte offset
// here.
func (r *OpenCodeReader) snapshotDatabase(sessionID string) (Boundary, bool) {
	path := r.databasePath()
	if path == "" {
		return Boundary{}, false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return Boundary{}, false
	}
	handle, err := vendorsqlite.Open(path)
	if err != nil {
		return Boundary{}, false
	}
	defer func() { _ = handle.Close() }()
	digest, size, ok := openCodeSessionArtifact(handle.DB, sessionID)
	if !ok {
		return Boundary{}, false
	}
	return Boundary{
		Agent:      "opencode",
		SessionID:  sessionID,
		ByteOffset: size,
		SizeBytes:  size,
		SHA256:     digest,
		ModTimeNS:  info.ModTime().UnixNano(),
		Partial:    false,
		path:       path,
	}, true
}

// openCodeDatabaseHasSession reports whether sessionID exists in db, without
// reading its message or part rows. Probe uses this to answer compatibility
// straight from the store when it can, rather than shelling out to the vendor
// CLI (which itself performs the unrelated writes snapshotDatabase's digest
// scoping above exists to ignore).
func openCodeDatabaseHasSession(db *sql.DB, sessionID string) bool {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM session WHERE id = ?`, sessionID).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

// openCodeSessionArtifact reports whether sessionID exists in db and, if so, a
// deterministic digest and byte count scoped to exactly that session's own
// rows: its session identity, and every message and part row it owns, ordered
// by id. It never reads any other session's rows or any other table.
func openCodeSessionArtifact(db *sql.DB, sessionID string) (digest string, size int64, ok bool) {
	var version string
	if err := db.QueryRow(`SELECT COALESCE(version, '') FROM session WHERE id = ?`, sessionID).Scan(&version); err != nil {
		return "", 0, false
	}
	h := sha256.New()
	write := func(field string) {
		_, _ = h.Write([]byte(field))
		_, _ = h.Write([]byte{0})
		size += int64(len(field))
	}
	writeBytes := func(field []byte) {
		_, _ = h.Write(field)
		_, _ = h.Write([]byte{0})
		size += int64(len(field))
	}
	write("session")
	write(sessionID)
	write(version)

	msgRows, err := db.Query(`SELECT id, time_created, time_updated, data FROM message WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return "", 0, false
	}
	for msgRows.Next() {
		var id string
		var created, updated int64
		var data []byte
		if err := msgRows.Scan(&id, &created, &updated, &data); err != nil {
			_ = msgRows.Close()
			return "", 0, false
		}
		write("message")
		write(id)
		write(strconv.FormatInt(created, 10))
		write(strconv.FormatInt(updated, 10))
		writeBytes(data)
	}
	msgErr := msgRows.Err()
	_ = msgRows.Close()
	if msgErr != nil {
		return "", 0, false
	}

	partRows, err := db.Query(`SELECT id, time_created, time_updated, data FROM part WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return "", 0, false
	}
	for partRows.Next() {
		var id string
		var created, updated int64
		var data []byte
		if err := partRows.Scan(&id, &created, &updated, &data); err != nil {
			_ = partRows.Close()
			return "", 0, false
		}
		write("part")
		write(id)
		write(strconv.FormatInt(created, 10))
		write(strconv.FormatInt(updated, 10))
		writeBytes(data)
	}
	partErr := partRows.Err()
	_ = partRows.Close()
	if partErr != nil {
		return "", 0, false
	}
	return hex.EncodeToString(h.Sum(nil)), size, true
}

// isOpenCodeDatabaseBoundary reports whether a boundary names the store.
func isOpenCodeDatabaseBoundary(b Boundary) bool {
	return strings.EqualFold(filepath.Base(b.Path()), OpenCodeDatabaseName)
}

// parseDatabaseMessages replays one session out of the embedded store, reusing
// the same event builder the filesystem layout uses so both produce identical
// capsule events.
func (r *OpenCodeReader) parseDatabaseMessages(b Boundary) ([]capsule.Event, error) {
	handle, err := vendorsqlite.Open(b.Path())
	if err != nil {
		return nil, fmt.Errorf("transcript: open opencode store: %w", err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.DB

	rows, err := db.Query(
		`SELECT id, data FROM message WHERE session_id = ? ORDER BY id LIMIT ?`,
		b.SessionID, maxOpenCodeMessages)
	if err != nil {
		return nil, fmt.Errorf("transcript: read opencode messages: %w", err)
	}
	type rawMessage struct {
		id   string
		data []byte
	}
	var raws []rawMessage
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			_ = rows.Close()
			return nil, err
		}
		raws = append(raws, rawMessage{id: id, data: data})
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	events := make([]capsule.Event, 0, len(raws))
	for index, raw := range raws {
		var msg openCodeMessageInfo
		if err := json.Unmarshal(raw.data, &msg); err != nil {
			return nil, fmt.Errorf("transcript: decode opencode message %s: %w", raw.id, err)
		}
		// id and sessionID are columns in this layout, not fields in the blob.
		msg.ID = raw.id
		msg.SessionID = b.SessionID
		if err := msg.validate(); err != nil {
			// A row the reader does not recognize is skipped, never guessed at.
			continue
		}
		parts, err := r.databaseParts(db, raw.id)
		if err != nil {
			return nil, err
		}
		ev, ok := openCodeMessageEvent(msg, parts, b.SessionID, index)
		if !ok {
			continue
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *OpenCodeReader) databaseParts(db *sql.DB, messageID string) ([]openCodePart, error) {
	rows, err := db.Query(`SELECT id, data FROM part WHERE message_id = ? ORDER BY id`, messageID)
	if err != nil {
		return nil, fmt.Errorf("transcript: read opencode parts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var parts []openCodePart
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		var part openCodePart
		if err := json.Unmarshal(data, &part); err != nil {
			continue
		}
		part.ID = id
		part.MessageID = messageID
		parts = append(parts, part)
	}
	return parts, rows.Err()
}
