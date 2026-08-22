package transcript

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/HarjjotSinghh/reinstate/internal/capsule"

	_ "modernc.org/sqlite"
)

// OpenCodeDatabaseName is the embedded session store inside the data root.
const OpenCodeDatabaseName = "opencode.db"

// openCodeDatabaseName is the package-local spelling of OpenCodeDatabaseName.
const openCodeDatabaseName = OpenCodeDatabaseName

// OpenCodeReadOnlyDSN builds the only DSN any part of Reinstate may use to open
// OpenCode's store.
//
// All three guards matter and none is redundant. `mode=ro` refuses writes,
// `immutable=1` is what stops SQLite creating `-wal` and `-shm` files beside the
// vendor's database, and `query_only(1)` refuses a write attempted through a
// connection that is already open. Creating a sidecar is a write under an agent
// root, which is the one thing a read-only continuity tool must never do — so
// this lives in one place, and every reader, index source and handoff
// destination calls it rather than spelling the DSN again.
func OpenCodeReadOnlyDSN(path string) string {
	return "file:" + url.PathEscape(filepath.ToSlash(path)) + "?mode=ro&immutable=1&_pragma=query_only(1)"
}

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
	return filepath.Join(data, openCodeDatabaseName)
}

func openCodeReadOnlyDSN(path string) string { return OpenCodeReadOnlyDSN(path) }

// snapshotDatabase freezes the whole embedded store as the boundary.
//
// A row-structured store has no "last complete record" the way an append-only
// JSONL transcript does: SQLite either opens and reads consistently or it does
// not. The boundary is therefore the whole artifact, its size and its digest,
// and is never partial. That is the honest analogue of a byte offset here.
func (r *OpenCodeReader) snapshotDatabase(sessionID string) (Boundary, bool) {
	path := r.databasePath()
	if path == "" {
		return Boundary{}, false
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return Boundary{}, false
	}
	if !r.databaseHasSession(path, sessionID) {
		return Boundary{}, false
	}
	digest, err := fileDigest(path)
	if err != nil {
		return Boundary{}, false
	}
	return Boundary{
		Agent:      "opencode",
		SessionID:  sessionID,
		ByteOffset: info.Size(),
		SizeBytes:  info.Size(),
		SHA256:     digest,
		ModTimeNS:  info.ModTime().UnixNano(),
		Partial:    false,
		path:       path,
	}, true
}

func (r *OpenCodeReader) databaseHasSession(path, sessionID string) bool {
	db, err := sql.Open("sqlite", openCodeReadOnlyDSN(path))
	if err != nil {
		return false
	}
	defer func() { _ = db.Close() }()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM session WHERE id = ?`, sessionID).Scan(&count); err != nil {
		return false
	}
	return count > 0
}

func fileDigest(path string) (string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = fh.Close() }()
	sum := sha256.New()
	if _, err := io.Copy(sum, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// isOpenCodeDatabaseBoundary reports whether a boundary names the store.
func isOpenCodeDatabaseBoundary(b Boundary) bool {
	return strings.EqualFold(filepath.Base(b.Path()), openCodeDatabaseName)
}

// parseDatabaseMessages replays one session out of the embedded store, reusing
// the same event builder the filesystem layout uses so both produce identical
// capsule events.
func (r *OpenCodeReader) parseDatabaseMessages(b Boundary) ([]capsule.Event, error) {
	db, err := sql.Open("sqlite", openCodeReadOnlyDSN(b.Path()))
	if err != nil {
		return nil, fmt.Errorf("transcript: open opencode store: %w", err)
	}
	defer func() { _ = db.Close() }()

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
